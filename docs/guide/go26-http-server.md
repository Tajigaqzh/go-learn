# 第 26 章 · HTTP 服务端与 REST API

第 25 章用 `net` 包手写 TCP 服务，自己处理粘包、长度前缀和每个连接的超时。真实业务里更常见的是 HTTP：协议解析、连接复用、分块传输这些细节全部交给 `net/http`，我们只写「请求进来以后做什么」。

这一章用一个内存版任务清单服务，把服务端的常见构件串起来：`http.Handler` 接口、Go 1.22 增强过的方法 + 通配符路由、JSON 请求与响应、统一错误体、中间件链（请求 ID / 访问日志 / panic 恢复 / CORS / 限流）、`http.Server` 超时、优雅关闭，以及用 `httptest` 写不依赖真实端口的测试。

本文所有「实测输出」都来自当前仓库。演示程序把时钟和请求 ID 生成器做成可注入的，因此除了监听端口，输出可以逐字复现——这也是本章想传达的第一条工程习惯：**让不确定性集中在少数几个可替换的点上**。

本章配套代码在 `internal/chapter/go26_http_server/`，执行 `go run ./cmd/go-learn` 可以看到全部输出；只想验证这一章用 `go test ./internal/chapter/go26_http_server/`。

---

## 26.1 Handler 与 HandlerFunc

### 结论

HTTP 服务端的核心抽象只有两个：`http.Handler` 接口和 `http.HandlerFunc` 适配器。前者的方法签名是 `ServeHTTP(http.ResponseWriter, *http.Request)`；后者是一个函数类型，用 `http.HandlerFunc(f)` 就能把普通函数转换成 `http.Handler`。标准库的 `ServeMux`、中间件、`httptest` 全都建立在这两个类型上。

### 最小可运行示例

```go
// greetingHandler 用结构体实现 http.Handler。
type greetingHandler struct {
	from string
}

// ServeHTTP 让 greetingHandler 满足 http.Handler 接口。
func (h greetingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "hello, %s", h.from)
}

// 等价的函数版写法
http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello, 函数")
})
```

### 实测输出

```text
1) 结构体实现 ServeHTTP: 200 hello, 结构体
2) http.HandlerFunc 适配函数: 200 hello, 函数
3) http.HandlerFunc 本身就是 http.Handler: true
   一个 handler 只做三件事：读请求、写响应；状态码由 WriteHeader 决定
```

结构体版适合需要携带状态（配置、依赖、logger）的场景；函数版适合无状态的短逻辑。两者在 `ServeMux` 眼里没有区别。

### 边界与坑

- **状态码在写 body 之前决定，而且只能写一次**：不写时第一次 `Write` 会隐式写出 200；第二次 `WriteHeader` 会被忽略，并在服务端日志里留下 `superfluous response.WriteHeader call`（见报错 2）。
- **`HandlerFunc` 不是装饰品**：`http.HandlerFunc(f)` 之后得到的是一个实现了 `ServeHTTP` 的值，因此可以放进结构体字段、返回给调用方、当作中间件的参数。

---

## 26.2 ServeMux 与 Go 1.22 路由增强

### 结论

从 Go 1.22 开始，`http.ServeMux` 的 pattern 支持「方法 + 路径」和 `{name}` 通配符，还能自动给出 404 / 405。以前需要手写 `switch r.Method` 再解析 URL 的代码，现在可以直接声明：

```go
mux.HandleFunc("GET /api/tasks", list)
mux.HandleFunc("POST /api/tasks", create)
mux.HandleFunc("GET /api/tasks/{id}", get)      // {id} 是通配符段
mux.HandleFunc("PATCH /api/tasks/{id}", update)
mux.HandleFunc("DELETE /api/tasks/{id}", remove)
```

处理器里用 `r.PathValue("id")` 取通配符，不再需要正则或 `strings.Split`。

### 最小可运行示例

```go
// routes 注册路由。
func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", a.handleHealth)
	a.mux.HandleFunc("GET /api/tasks", a.handleListTasks)
	a.mux.HandleFunc("POST /api/tasks", a.handleCreateTask)
	a.mux.HandleFunc("GET /api/tasks/{id}", a.handleGetTask)
	a.mux.HandleFunc("PATCH /api/tasks/{id}", a.handleUpdateTask)
	a.mux.HandleFunc("DELETE /api/tasks/{id}", a.handleDeleteTask)
}

// pathID 解析 {id} 路径参数。
func pathID(r *http.Request) (int, error) {
	raw := r.PathValue("id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("id 必须是整数，收到 %q", raw)
	}
	if id <= 0 {
		return 0, fmt.Errorf("id 必须大于 0，收到 %d", id)
	}
	return id, nil
}
```

### 实测输出

```text
路由注册（Go 1.22 起支持「方法 + 路径」模式）:
  GET /healthz
  GET /api/tasks
  POST /api/tasks
  GET /api/tasks/{id}
  PATCH /api/tasks/{id}
  DELETE /api/tasks/{id}
实际请求:
GET    /api/tasks/1                       -> 200 {"id":1,"title":"写第 26 章","done":false,"created_at":"2024-05-01T09:00:00Z"}
POST   /api/tasks/1                       -> 405 Method Not Allowed  Allow: DELETE, GET, HEAD, PATCH
GET    /api/absent                        -> 404 404 page not found
GET    /api/tasks/abc                     -> 400 {"code":"invalid_argument","message":"id 必须是整数，收到 \"abc\""}
```

三行输出对应三种情况，都是 ServeMux 自己判断出来的：路径匹配但方法不匹配是 405，`Allow` 头里列出可用方法（`HEAD` 是标准库为 `GET` 自动补上的）；没有任何模式匹配是 404，响应体是默认的 `404 page not found`；模式匹配但业务校验失败，由 handler 自己回 400。

### 边界与坑

- **模式冲突会在注册时 panic**：`GET /api/tasks/{id}` 和 `GET /api/tasks/{name}` 覆盖的请求完全一样，ServeMux 直接 panic（见下文报错 1）。这类冲突在启动时就暴露，比运行期才 404 好排查得多。
- **通配符只匹配一个路径段**：`{id}` 不会匹配 `/api/tasks/1/comments`，需要显式注册更长的模式；`{path...}` 才是「剩余全部」，适合静态文件代理而不是 ID 参数。
- **模式写法要自查**：同一模式里重复出现的通配符必须同名同值，拼错名字会 panic；方法名必须大写，`get /api/tasks` 会被当成路径的一部分。

---

## 26.3 请求解析与 JSON 响应

### 结论

HTTP 请求的输入有三个来源：路径段（`PathValue`）、查询串（`r.URL.Query()`）、请求体（`json.Decoder`）。三者都要显式校验，并且**先校验再落库**；响应统一用 `application/json`，编码用 `json.NewEncoder(w).Encode`。

### 最小可运行示例

```go
// createTaskRequest 是 POST 的请求体。
type createTaskRequest struct {
	Title string `json:"title"`
}

// updateTaskRequest 是 PATCH 的请求体，字段用指针区分「没传」和「传了零值」。
type updateTaskRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}

// decodeJSON 读取并解析请求体，把各种失败翻译成合适的错误响应。
func (a *API) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large",
				fmt.Sprintf("请求体不能超过 %d 字节", maxBodyBytes), "")
			return err
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "请求体不是合法 JSON", err.Error())
		return err
	}
	return nil
}
```

请求路径参数与查询参数的解析：

```go
done, err := parseOptionalBool(r, "done")   // ?done=true
limit, err := parseOptionalInt(r, "limit")  // ?limit=2
```

### 实测输出

```text
PATCH  /api/tasks/2                       -> 200 {"id":2,"title":"复习 pprof","done":true,"created_at":"2024-05-01T09:00:00Z"}
查询参数过滤与截断:
GET    /api/tasks                         -> 200 {"count":2,"items":[{"id":1,"title":"写第 26 章","done":false,"created_at":"2024-05-01T09:00:00Z"},{"id":2,"title":"复习 pprof","done":true,"created_at":"2024-05-01T09:00:00Z"}]}
GET    /api/tasks?done=true               -> 200 {"count":1,"items":[{"id":2,"title":"复习 pprof","done":true,"created_at":"2024-05-01T09:00:00Z"}]}
GET    /api/tasks?limit=1                 -> 200 {"count":1,"items":[{"id":1,"title":"写第 26 章","done":false,"created_at":"2024-05-01T09:00:00Z"}]}
失败路径:
POST   /api/tasks                         -> 400 {"code":"invalid_argument","message":"title 不能为空"}
POST   /api/tasks                         -> 400 {"code":"invalid_json","message":"请求体不是合法 JSON","details":"json: cannot unmarshal number into Go struct field createTaskRequest.title of type string"}
POST   /api/tasks                         -> 400 {"code":"invalid_json","message":"请求体不是合法 JSON","details":"json: unknown field \"titel\""}
GET    /api/tasks?limit=abc               -> 400 {"code":"invalid_argument","message":"limit 必须是整数，收到 \"abc\""}
```

注意最后两条 `details`：它们直接来自 `encoding/json` 的报错原文，指向具体字段。`DisallowUnknownFields()` 让「客户端把字段拼成 `titel`」这种静默丢数据的 bug 变成一次明确的 400。

### 边界与坑

- **限制请求体大小**：`http.MaxBytesReader` 是必须的，否则一个 10 GB 的 body 就能把内存吃满。超过上限返回 413。
- **先 `TrimSpace` 再判空**：`{"title":"   "}` 是真实存在的输入，只判 `== ""` 会放过去。
- **指针字段区分零值**：PATCH 里 `{"done":false}` 和「没传 done」语义不同，只有指针（或 `json.RawMessage`）能区分；另外 `dec.Decode` 只读一个 JSON 值，多余内容会被忽略，严格模式要再读一次确认返回 `io.EOF`。
- **响应要设 Content-Type**：不设的话 `net/http` 会用 `http.DetectContentType` 猜，JSON 很容易被猜成 `text/plain`。

---

## 26.4 统一错误响应

### 结论

错误响应要么全部 HTML，要么全部 JSON，**不能混着来**。推荐固定一个结构：`code` 给程序判断、`message` 给人看、`details` 给排查线索。所有失败路径都从同一个函数出口，客户端只需要写一次错误处理。

### 最小可运行示例

```go
// APIError 是统一的错误响应体。
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// writeJSON 写出 JSON 响应：先设 Content-Type，再写状态码，最后编码响应体。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 是统一的错误出口。
func writeError(w http.ResponseWriter, status int, code, message, details string) {
	writeJSON(w, status, APIError{Code: code, Message: message, Details: details})
}
```

### 实测输出

```text
统一错误体（code 给程序判断，message 给人看）:
GET    /api/tasks/404                     -> 404 {"code":"not_found","message":"任务不存在"}
http.Error 的默认行为: 404 "任务不存在\n" Content-Type=text/plain; charset=utf-8
writeJSON 的结果: 400 {"code":"invalid_argument","message":"title 不能为空"}
状态码只能写一次：先 WriteHeader 再写 body，顺序颠倒会触发 superfluous 警告
```

`http.Error` 适合内部工具和调试接口（`text/plain` 加一个换行）；对外 API 用统一 JSON，客户端才能稳定地按 `code` 分支处理。

### 边界与坑

- **`WriteHeader` 必须发生在写 body 之前**：顺序颠倒时状态码仍然是第一次写的那个，但日志里会出现 superfluous 警告（见报错 2）。
- **错误细节不要泄露内部实现**：`details` 只放客户端能理解的线索（字段名、校验规则）。数据库报错、SQL 语句、堆栈一律只进日志。
- **状态码要选准**：参数错是 400、没权限是 403、不存在是 404、冲突是 409、限流是 429，选错会让客户端重试策略全乱；204 则不写任何 body（`writeJSON` 里显式返回就是为了这个）。

---

## 26.5 中间件链

### 结论

中间件的类型就是 `func(http.Handler) http.Handler`：包一层、做事、再调用下一层。链的顺序决定了执行顺序——**先注册的最先看到请求、最后看到响应**。典型顺序是：请求 ID → 访问日志 → panic 恢复 → CORS → 限流 → 业务路由。

### 最小可运行示例

```go
// Middleware 把下一个 handler 包装成新的 handler。
type Middleware func(http.Handler) http.Handler

// Chain 按书写顺序把中间件套在 h 外面。
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// Logging 输出一行结构化访问日志。
func Logging(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			logger.Printf("request_id=%s method=%s path=%s status=%d bytes=%d",
				RequestIDFrom(r.Context()), r.Method, r.URL.Path, rec.Status(), rec.bytes)
		})
	}
}
```

`statusRecorder` 是关键的一层包装：`http.ResponseWriter` 不告诉你「刚才写了什么状态码、写了多少字节」，所以要自己记录：

```go
// WriteHeader 只转发第一次调用。
func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
```

### 实测输出

```text
预检: OPTIONS /api/tasks -> 204 Allow-Origin=https://example.com Allow-Methods=GET, POST, PATCH, DELETE, OPTIONS
panic 恢复（未捕获的 panic 会掐断连接，Recover 把它变成 500）:
GET    /panic                             -> 500 {"code":"internal","message":"服务器内部错误"}
中间件写出的访问日志:
request_id=demo-1 method=POST path=/api/tasks status=201 bytes=82
request_id=demo-2 method=GET path=/api/tasks status=200 bytes=104
request_id=demo-3 method=GET path=/api/tasks/1 status=200 bytes=82
request_id=from-client method=GET path=/healthz status=200 bytes=16
request_id=demo-4 method=OPTIONS path=/api/tasks status=204 bytes=0
panic recovered: request_id=demo-5 panic=故意触发的 panic
request_id=demo-5 method=GET path=/panic status=500 bytes=54
```

三条信息值得注意：

- `request_id=from-client`：调用方带了 `X-Request-ID`，中间件就沿用，跨服务能串成一条链路；没带才生成。
- `panic recovered` 出现在访问日志**之前**：因为恢复中间件在日志中间件里面，它先处理完 panic 并写响应，日志中间件才拿到「已完成的响应」。
- 没有 `Recover` 中间件时，panic 会掐断连接，客户端拿到的是 `EOF`，服务端日志里是 `http: panic serving ...`（见报错 3）。

### 边界与坑

- **顺序不能乱，包装要防重复**：请求 ID 必须在日志之前（否则日志没有 ID）、日志在恢复之前（否则 panic 的请求不会被记录）；`statusRecorder` 里不判断 `status != 0`，就会把 200 之后又写 500 的调用转发给真正的 ResponseWriter，触发 superfluous 警告。
- **CORS 只回显可信来源**：生产环境不要写 `Access-Control-Allow-Origin: *` 还带着 Cookie 凭证；带凭证时必须回显具体来源，并补 `Vary: Origin` 防止缓存串味。
- **限流要能被测试，中间件里别做长耗时工作**：把时钟抽成 `func() time.Time` 注入，限流逻辑就能用「假时间」精确断言（见 26.7）；中间件每个请求都会经过，任何阻塞都会被线性放大。

---

## 26.6 服务器超时与优雅关闭

### 结论

`http.ListenAndServe` 的所有超时默认都是 0，也就是永不超时——一个只连不发的客户端就能长期占着资源。生产服务至少要显式设置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout`。关闭时用 `Shutdown(ctx)` 等在途请求处理完，而不是直接 `Close()` 掐断。

### 最小可运行示例

```go
// NewServer 返回带超时配置的 http.Server。
func NewServer(handler http.Handler, logger *log.Logger) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,  // 防 Slowloris：只读请求头的时限
		ReadTimeout:       10 * time.Second, // 整个请求（含 body）的读取时限
		WriteTimeout:      10 * time.Second, // 从读请求头开始到响应写完
		IdleTimeout:       60 * time.Second, // keep-alive 空闲连接回收
		ErrorLog:          logger,
	}
}
```

配合信号量使用，收到 Ctrl+C / SIGTERM 后优雅退出：

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

ln, err := net.Listen("tcp", ":8080")
if err != nil {
	return err
}
if err := ServeUntilCanceled(ctx, NewServer(handler, logger), ln, logger); err != nil {
	return err
}
```

### 实测输出

```text
监听地址: 127.0.0.1:57437（端口每次运行都不同）
超时配置: ReadHeaderTimeout=5s ReadTimeout=10s WriteTimeout=10s IdleTimeout=1m0s
GET    /healthz                           -> 200 {"status":"ok"}
取消 ctx 后 ServeUntilCanceled 返回: <nil>
服务器日志: request_id=demo-1 method=GET path=/healthz status=200 bytes=16
服务器已优雅关闭
关闭后再次请求失败: true
```

四行输出对应四个事实：服务真的起来了、健康检查通过、`Shutdown` 干净返回、关闭后连接被拒绝。`ServeUntilCanceled` 会先 `srv.Shutdown(ctx)`，再回收 `srv.Serve` 的返回值，避免留下一个永远阻塞在 channel 上的 goroutine。

### 边界与坑

- **`Shutdown` 会等，`Close` 不会**：`Close` 立刻关掉所有连接，正在处理的请求会失败。要用 `Shutdown` 加超时 context。
- **`Shutdown` 超时会返回 `context deadline exceeded`**：说明还有请求没处理完。此时要么继续等待，要么记录日志后 `Close`，别假装成功。
- **`srv.Serve` 返回 `http.ErrServerClosed` 是正常的**：这是关闭流程的一部分，用 `errors.Is(err, http.ErrServerClosed)` 过滤掉，否则日志里会出现一条假的「服务异常退出」（见报错 5）。
- **超时与在途请求都有边界**：`WriteTimeout` 从读完请求头开始计时，会掐断长轮询、大文件下载、SSE，这类接口要单独放宽或改用 `http.ResponseController`；`Shutdown` 只是不再接受新连接，处理中的请求仍会跑完，需要硬上界就配合 `context` 取消和 `BaseContext`。

---

## 26.7 httptest 与生产化细节

### 结论

测试 HTTP 处理器不需要真实端口：`httptest.NewRequest` 造请求、`httptest.NewRecorder` 收响应，直接调用 `handler.ServeHTTP` 即可，毫秒级完成。需要验证连接复用、超时、真实序列化时，再用 `httptest.NewServer` 起一个临时服务。

### 最小可运行示例

```go
// newTestAPI 构造一个日志丢弃、时钟固定的 API，供测试使用。
func newTestAPI(options ...Option) *API {
	return NewAPI(newStoreWithClock(demoTime), log.New(io.Discard, "", 0), options...)
}

// 表驱动测试：一个 handler，多组输入输出。
for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.wantStatus {
			t.Errorf("状态码：期望 %d，实际 %d", tt.wantStatus, rec.Code)
		}
	})
}
```

限流的可测试写法——把时钟注入进去：

```go
// newLimiterWithClock 允许注入时钟，测试和演示因此可以精确控制「过了多久」。
func newLimiterWithClock(rate float64, burst int, now func() time.Time) *Limiter {
	// ...
}
```

### 实测输出

```text
httptest.NewRecorder: GET /healthz -> 200 {"status":"ok"}
响应头里的请求 ID 自动生成: true
令牌桶（容量 3，1 个/秒）连打 5 次: 200 200 200 429 429 ；前进 1 秒后: 200
2 KB 请求体: 413 {"code":"request_too_large","message":"请求体不能超过 1024 字节"}（Content-Type=application/json; charset=utf-8）
```

令牌桶的结果完全确定：桶容量 3，连打 5 次就是「3 个通过 + 2 个 429」，把假时钟往前拨 1 秒就又能通过一次。如果直接用 `time.Now`，这段断言只能靠 sleep 去凑，测试会变慢且偶发失败。

### 边界与坑

- **`Recorder` 覆盖不到真实网络**：它测不出超时、连接复用、TLS，这些要用 `httptest.NewServer`；而 `NewServer` 用完必须 `Close()`，否则残留的监听 socket 会让 `go test` 挂住。
- **别忘了 `defer resp.Body.Close()`**：客户端不关闭 body，连接不会回到连接池，压测时表现为句柄耗尽。
- **测「错误路径」比测「成功路径」更重要**：400 / 404 / 405 / 413 / 429 / 500 每一条都要有断言，这些才是线上真正会踩的分支。
- **限流按 key 分桶**：示例用的是全局令牌桶；按 IP 或用户限流时换成 `map[string]*Limiter`，并给这个 map 加锁 + 定期清理，否则内存会随 key 无限增长。

---

## 6 个真实报错怎么读

下面每一条都是在本仓库实测跑出来的输出。路径里的目录名取决于你把示例放在哪儿，行号精确对应上面的最小示例。

### 报错 1：路由模式冲突 → 注册时 panic

```go
mux.HandleFunc("GET /api/tasks/{id}", handler)
// 同一个形状的通配符再注册一次，两个模式互相冲突
mux.HandleFunc("GET /api/tasks/{name}", handler)
```

```text
panic: pattern "GET /api/tasks/{name}" (registered at C:/Users/123/Desktop/demo-project/go-learn/.tmp-errors/muxconflict/main.go:16) conflicts with pattern "GET /api/tasks/{id}" (registered at .../muxconflict/main.go:14):
	GET /api/tasks/{name} matches the same requests as GET /api/tasks/{id}

goroutine 1 [running]:
net/http.(*ServeMux).register(...)
	.../src/net/http/server.go:2957
net/http.(*ServeMux).HandleFunc(0x0?, {...}, ...)
	.../src/net/http/server.go:2931 +0x5e
main.main()
	.../.tmp-errors/muxconflict/main.go:16 +0x55
exit status 2
```

报错把两个模式的注册位置都标了出来（第 16 行与第 14 行），最后一行说明冲突原因：两个模式匹配的请求集合完全相同。**修复方向**是让它们能被区分开，比如 `GET /api/tasks/{id}` 与 `GET /api/tasks/{id}/comments`。

### 报错 2：重复写状态码 → superfluous 警告

```go
w.WriteHeader(http.StatusBadRequest)        // 第一次写状态码
w.WriteHeader(http.StatusInternalServerError) // 第二次是多余的
fmt.Fprintln(w, "body")
```

```text
2026/09/14 17:14:00 http: superfluous response.WriteHeader call from main.main.func1 (main.go:13)
客户端看到的状态码: 400
```

第一次写下的 400 已经发给客户端了，第二次只能被忽略，并在服务端日志里留下 `main.go:13` 这个准确位置。**修复思路**：状态码只在统一出口写一次，把「业务结果」与「写响应」分开——先算出 `status` 和 payload，最后调一次 `writeJSON`。

### 报错 3：handler 里的 panic 没被恢复 → 连接被掐断

```go
srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
	panic("boom")
}))
```

```text
2026/09/14 17:14:00 http: panic serving 127.0.0.1:57596: boom
goroutine 23 [running]:
net/http.(*conn).serve.func1()
	.../src/net/http/server.go:1939 +0xbb
panic({0x7ff6d216b378?, 0x7ff6d1ec2870?})
	.../src/runtime/panic.go:859 +0x125
main.main.func1({0x7ff6d1e08169?, 0x10?}, 0x7ff6d217b070?)
	.../.tmp-errors/unrecoveredpanic/main.go:12 +0x25
...
客户端拿到: Get "http://127.0.0.1:57611": EOF
```

服务端日志里的第一行是 `http: panic serving <对端地址>: boom`，紧跟的调用栈直接指到 `main.go:12`。客户端这一侧只有一句 `EOF`——**没有状态码、没有错误体**，因为它等的是一个已经断掉的连接。这就是 `Recover` 中间件存在的意义：把 panic 转成 500 + JSON，让客户端能按正常错误处理。

### 报错 4：客户端超时 → context deadline exceeded

```go
// 服务端处理 500ms，客户端只等 100ms
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(500 * time.Millisecond)
	fmt.Fprintln(w, "done")
}))
client := &http.Client{Timeout: 100 * time.Millisecond}
```

```text
请求失败: Get "http://127.0.0.1:57600": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

括号里那句话点明了超时发生在哪个阶段：**等待响应头**；读 body 阶段的超时则显示 `reading response body`。**修复方向**：把服务端耗时压到超时以内，或者按接口分别设置客户端超时，而不是所有调用共用一个值。

### 报错 5：正常关闭被当成故障 → `http: Server closed`

```go
serveErr := make(chan error, 1)
go func() { serveErr <- srv.Serve(ln) }()

// ... 收到退出信号，执行 Shutdown ...

err = <-serveErr
fmt.Println("Serve 返回:", err)
```

```text
Serve 返回: http: Server closed
errors.Is(err, http.ErrServerClosed) = true
```

`srv.Serve` 在 `Shutdown` 之后一定会返回 `http.ErrServerClosed`，这是**关闭流程的正常结果**。直接 `log.Fatal(err)` 会让每次优雅重启都留下一条假的「服务异常」日志；**修复方式**是用 `errors.Is(err, http.ErrServerClosed)` 过滤。

### 报错 6：端口被占用 → bind 失败

```go
ln, _ := net.Listen("tcp", "127.0.0.1:0")
defer ln.Close()

// 同一个地址再监听一次
second, err := net.Listen("tcp", ln.Addr().String())
```

```text
第二次监听失败: listen tcp 127.0.0.1:57604: bind: Only one usage of each socket address (protocol/network address/port) is normally permitted.
```

这是 Windows 上的文案（Linux 上显示 `bind: address already in use`），含义都是地址已被占用，常见原因是上一个进程没退干净或代码被启动了两次。**修复方向**：确认端口占用者，或直接用 `127.0.0.1:0` 让内核分配端口（本章演示就是这么做的）。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 日志里出现 `superfluous response.WriteHeader call` | 写了两次状态码，或先写 body 再写状态码 | 状态码只在统一出口写一次，且必须在写 body 之前 |
| 客户端收到 `EOF`、没有错误响应 | handler 里 panic 没被恢复 | 加 `Recover` 中间件，把 panic 转成 500 |
| 服务启动就 panic：`pattern ... conflicts` | 两个路由模式匹配同一批请求 | 让模式可区分，或用更长的路径区分 |
| 路径参数永远取不到，或 POST 一直返回 415 | 用了 `r.URL.Query()` / 自己切字符串；客户端没带 `Content-Type: application/json` | 用 `r.PathValue("id")` 并确认 pattern 写的是 `{id}`；请求端补 Content-Type，服务端显式校验而不是默默解析 |
| 字段拼错却「没有报错」，或内存被大请求体吃满 | 默认解码器忽略未知字段；没有限制 body 大小 | `dec.DisallowUnknownFields()` 配合 `http.MaxBytesReader(w, r.Body, limit)`，超限返回 413 |
| 慢连接长时间占资源 | `http.ListenAndServe` 超时默认为 0 | 显式设置 `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` / `IdleTimeout` |
| 重启时请求失败，日志里还有「服务异常退出」 | 用了 `Close()` 而不是 `Shutdown(ctx)`；把 `http.ErrServerClosed` 当错误 | `Shutdown` + 超时 context，并用 `errors.Is(err, http.ErrServerClosed)` 过滤正常关闭 |
| 浏览器报 CORS 错误 | 缺 `Access-Control-Allow-Origin` 或预检没处理 | 回显可信来源、处理 `OPTIONS`、带凭证时不要用 `*` |
| 测试偶发失败、需要 sleep，或 `go test` 卡住不退出 | 限流/超时逻辑依赖 `time.Now`；`httptest.NewServer` 没 `Close()`、`resp.Body` 没关 | 把时钟注入（`func() time.Time`）用假时间断言；补上 `defer server.Close()` 和 `defer resp.Body.Close()` |

---

## 练习

### 第 1 题

下面的 handler 有两个问题，指出它们并修复：

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintln(w, "bad request")
	}

	task := store.Create(req.Title)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("created"))
	fmt.Println(task.ID)
}
```

::: details 第 1 题参考答案

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10) // 限制大小

	var req createTaskRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "请求体不是合法 JSON", err.Error())
		return // 必须返回，不能继续往下走
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "invalid_argument", "title 不能为空", "")
		return
	}

	task := store.Create(req.Title)
	w.Header().Set("Location", "/api/tasks/"+strconv.Itoa(task.ID))
	writeJSON(w, http.StatusCreated, task)
}
```

**为什么这样写更好**：原代码解析失败后没有 `return`，会用空的 `req` 继续建任务——这是真实事故的常见形态；状态码在创建之前就写死了 201，后面出错也改不了；响应体是 `text/plain` 而接口按 JSON 约定；`fmt.Println` 把日志写到了标准输出，应该交给 logger 并在响应之后再记录。

:::

### 第 2 题

给本章的任务 API 增加「标记完成」接口：`POST /api/tasks/{id}/done`，要求用 ServeMux 的模式匹配实现，并处理「任务不存在」。

::: details 第 2 题参考答案

```go
// 注册：注意 pattern 里同时用了通配符和固定后缀段
a.mux.HandleFunc("POST /api/tasks/{id}/done", a.handleTaskDone)

func (a *API) handleTaskDone(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error(), "")
		return
	}

	done := true
	task, ok := a.store.Update(id, nil, &done)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "任务不存在", "")
		return
	}
	writeJSON(w, http.StatusOK, task)
}
```

**为什么这样写更好**：路径里带上动作（`/done`）比让客户端 `PATCH {"done":true}` 更直白，也不需要客户端知道字段名；`pathID` 复用了同一套 ID 校验，错误码与既有接口一致。注意 `{id}/done` 与 `{id}` 不会冲突——前者匹配更长的路径，ServeMux 按「更具体的模式优先」选择。

:::

### 第 3 题

写一个中间件，把每个请求的耗时写进访问日志，并保证它和已有的 `Logging` 中间件能组合使用。

::: details 第 3 题参考答案

```go
// Latency 记录请求耗时，输出到 logger。
func Latency(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Printf("path=%s duration=%s", r.URL.Path, time.Since(start))
		})
	}
}

// 组合顺序：请求 ID -> 访问日志 -> 耗时 -> 恢复 -> 路由
handler := Chain(mux, RequestID(), Logging(logger), Latency(logger), Recover(logger))
```

**为什么这样写更好**：耗时单独一个中间件，职责清晰，也方便临时关掉；`time.Since(start)` 用单调时钟，即使系统时间被调整也不会算出负数。**别把耗时合并进 `Logging`**：日志格式一旦拆成两个中间件，就能分别测试，也能对慢请求单独采样。测试时不要断言具体耗时，只断言日志里出现了 `duration=`。

:::

### 第 4 题

用表驱动 + `httptest` 为 `GET /api/tasks?limit=abc` 这类查询参数错误写测试，要求覆盖 `limit` 和 `done` 两个参数。

::: details 第 4 题参考答案

```go
func TestQueryValidation(t *testing.T) {
	api := newTestAPI()
	handler := api.Handler()

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   string
	}{
		{"limit 不是整数", "/api/tasks?limit=abc", http.StatusBadRequest, "invalid_argument"},
		{"limit 是负数", "/api/tasks?limit=-1", http.StatusBadRequest, "invalid_argument"},
		{"done 不是布尔", "/api/tasks?done=maybe", http.StatusBadRequest, "invalid_argument"},
		{"参数合法", "/api/tasks?done=false&limit=10", http.StatusOK, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("状态码：期望 %d，实际 %d（%s）", tt.wantStatus, rec.Code, rec.Body.String())
			}
			if tt.wantCode == "" {
				return
			}
			var apiErr APIError
			if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
				t.Fatalf("错误体解析失败: %v", err)
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("错误码：期望 %q，实际 %q", tt.wantCode, apiErr.Code)
			}
		})
	}
}
```

**为什么这样写更好**：一组输入输出用一张表描述，新增用例只加一行；成功用例和失败用例写在一起，能防止「为了过测试把校验删掉」。断言里带上响应体，失败时不用再跑一遍就能看出服务端返回了什么。

:::

### 第 5 题

把示例里的全局令牌桶改成「按客户端 IP 限流」，并说明怎么避免这个 map 无限增长。

::: details 第 5 题参考答案

```go
// IPRateLimiter 按 key 维护各自的令牌桶。
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*Limiter
	rate     float64
	burst    int
}

func (l *IPRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	limiter, ok := l.limiters[key]
	if !ok {
		limiter = NewLimiter(l.rate, l.burst)
		l.limiters[key] = limiter
	}
	l.mu.Unlock()
	return limiter.Allow()
}

// 定期清理不再活跃的桶，避免 map 随 key 无限增长。
func (l *IPRateLimiter) Cleanup(idle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for key, limiter := range l.limiters {
		if now.Sub(limiter.LastSeen()) > idle {
			delete(l.limiters, key)
		}
	}
}
```

取 key 的方式：

```go
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
```

**为什么这样写更好**：全局令牌桶会让一个恶意客户端耗尽所有人的配额；按 key 分桶把影响面限制在单个客户端。**必须配清理**：`map[string]*Limiter` 的 key 来自外部输入，不清理就是内存泄漏。清理可以用 `time.Ticker` 定期跑，也可以在 `Allow` 里顺便做惰性清理。注意反向代理场景下 `RemoteAddr` 是代理地址，此时应读 `X-Forwarded-For`，并只信任自己网关写入的那个值。

:::

### 第 6 题

服务上线后要在不丢请求的前提下重启。写出从「收到信号」到「进程退出」的完整流程，并说明每一步的作用。

::: details 第 6 题参考答案

```go
func main() {
	// 1. 收到 SIGINT/SIGTERM 时取消 ctx
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. 先监听，失败就不用启动了
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	// 3. 带超时的服务器
	srv := NewServer(handler, logger)

	// 4. 在 ctx 被取消后停止接收新连接，并等待在途请求结束
	if err := ServeUntilCanceled(ctx, srv, ln, logger); err != nil {
		log.Printf("服务退出: %v", err)
		os.Exit(1)
	}
	log.Println("已优雅退出")
}
```

**每步的作用**：

1. `signal.NotifyContext` 把信号转成 context 取消，不需要自己写 channel 和 `select`。
2. **先 `net.Listen` 再进服务循环**：端口占用、权限不足这类错误在启动阶段就暴露；同时监听器已经就绪，负载均衡探活不会打空。
3. 显式超时配置，避免慢连接拖住关闭流程。
4. `Shutdown(ctx)` 会关闭监听、等待在途请求，超过 `shutdownTimeout`（本章是 5 秒）返回 `context deadline exceeded`；`Serve` 的 `http.ErrServerClosed` 属于正常返回，要过滤掉。

配合容器与负载均衡：先让探针失败（`/healthz` 提前返回 503）→ 等待摘流 → 再发 SIGTERM → `Shutdown` 等在途请求 → 退出。这样客户端几乎感知不到重启。

:::

---

## 小结

- **两个核心类型**：`http.Handler` 定义能力，`http.HandlerFunc` 把函数适配成接口；状态码由 `WriteHeader` 决定，默认 200。
- **路由用 Go 1.22 增强后的 ServeMux**：`"GET /api/tasks/{id}"` + `r.PathValue("id")`，404/405 由标准库自动给出，模式冲突在启动时 panic。
- **三个输入来源都要显式校验**：路径参数、查询参数、JSON body。`MaxBytesReader` 限制大小，`DisallowUnknownFields` 拦住拼错的字段。
- **统一错误体**：`code` 给程序、`message` 给人、`details` 给排查；状态码只在统一出口写一次，写两次会触发 superfluous 警告。
- **中间件就是 `func(http.Handler) http.Handler`**：请求 ID → 日志 → 恢复 → CORS → 限流 → 路由，顺序决定行为；`statusRecorder` 让日志能记下状态码和字节数，`Recover` 把 panic 变成 500（否则客户端只拿到 `EOF`）。
- **超时和优雅关闭都要显式配置**：四个超时字段缺一不可（`ListenAndServe` 默认全是 0），关闭用 `Shutdown` 加超时 context，并用 `errors.Is` 过滤掉 `http.ErrServerClosed`。
- **可测试性来自注入**：时钟、随机 ID、存储都做成可替换的，测试就能用 `httptest` 断言精确结果；边界能力（限流、CORS）则要按 key 分桶、只回显可信来源。

下一章将讲解 **HTTP 客户端与外部服务**：`http.Client` 超时与连接复用、重试与退避、`resp.Body` 的正确关闭方式、文件上传下载，以及用 `httptest.Server` 测试调用外部服务的代码。
