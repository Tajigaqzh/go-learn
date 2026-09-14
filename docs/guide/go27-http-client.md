# 第 27 章 · HTTP 客户端与外部服务

第 26 章写的是服务端：别人来调我们。这一章反过来——**我们作为客户端去调别人的服务**。客户端代码看起来只是「发个请求、解析响应」，但线上事故大多出在这里：没有超时导致 goroutine 挂死、Body 不关导致连接泄漏、无脑重试把对方打垮、把 4xx 当成网络抖动反复重试。

这一章围绕 `net/http` 的客户端能力展开：`http.Client` 的三层超时、`Transport` 连接池与复用、请求构造（JSON / 表单 / 请求头）、响应处理（状态码、gzip、Body 关闭）、重试与退避、上传下载，以及用 `httptest` 把外部依赖变成可控的替身。所有演示都打一个本地假上游，不依赖外网。

为了让输出可复现，演示里的时钟、退避等待和抖动都可以注入，临时端口会替换成 `<upstream>` 占位符——正文里的数字和报错都来自当前仓库的真实运行结果。

本章配套代码在 `internal/chapter/go27_http_client/`，执行 `go run ./cmd/go-learn` 可以看到全部输出；只想验证这一章用 `go test ./internal/chapter/go27_http_client/`。

---

## 27.1 http.Client 超时与 context

### 结论

超时有三层，作用范围不同，**至少要设一层**：

| 层次 | 位置 | 覆盖范围 |
| --- | --- | --- |
| 整次调用 | `http.Client.Timeout` | 从发请求到读完响应体，含重定向与重试 |
| 单次请求 | `context.WithTimeout` | 同样覆盖整次调用，但能跨函数传递、可主动取消 |
| 单个阶段 | `Transport` 的 `ResponseHeaderTimeout` / `TLSHandshakeTimeout` | 只覆盖等响应头、TLS 握手这类具体阶段 |

`http.DefaultClient` 的 `Timeout` 是 0，也就是永不超时——请求可能永远挂在那里。

### 最小可运行示例

```go
// NewClient 返回配置好超时与连接池的 http.Client。
func NewClient() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: NewTransport(),
	}
}

// 每次调用再套一层调用方自己的预算
ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
defer cancel()

user, err := api.User(ctx, "1")
```

### 实测输出

```text
① Client.Timeout=100ms，上游需要 300ms:
   Get "<upstream>/api/slow?ms=300": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
   *url.Error.Timeout() = true，errors.Is(err, context.DeadlineExceeded) = true
② 用 context.WithTimeout(50ms) 控制同一次调用:
   Get "<upstream>/api/slow?ms=300": context deadline exceeded
   errors.Is(err, context.DeadlineExceeded) = true，Retryable(err) = false
③ 调用方主动取消，请求立刻结束:
   Get "<upstream>/api/slow?ms=0": context canceled
   errors.Is(err, context.Canceled) = true，Retryable(err) = false
④ 预算充足时正常返回:
   Ping(0ms) = <nil>
```

三种情况的错误都包在 `*url.Error` 里，所以判定要用 `errors.Is` / `errors.As`，不要用字符串匹配。`Client.Timeout` 的错误里会额外带一句 `Client.Timeout exceeded while awaiting headers`，告诉你超时发生在哪个阶段。

### 边界与坑

- **`Client.Timeout` 覆盖读 body**：它计时到响应体读完为止。大文件下载会被它掐断，这类接口要么把超时调大，要么改用 context 分阶段控制（连接、响应头、body 各自设预算）。
- **`Client.Timeout` 是「整次调用」**：包含重定向和内部重试，排查时别只盯着 DNS 和连接耗时。
- **两层同时设置时先到的生效**：上游要 300ms，客户端 100ms、调用方 50ms，实际会在 50ms 结束。
- **取消与超时不要重试**：`Retryable` 里明确把 `context.Canceled` 与 `context.DeadlineExceeded` 排除——调用方已经决定不要这个结果了，重试只是白花资源。
- **别用 `time.AfterFunc` 之类的土办法**：`context` 是唯一能一路传下去、也能被上层取消的机制。

---

## 27.2 Transport 与连接复用

### 结论

连接池挂在 `http.Client` 上，复用连接省掉的是 TCP 三次握手和 TLS 握手（各几十毫秒）。默认 `Transport` 的 `MaxIdleConnsPerHost` 只有 **2**，并发一高，多余的连接用完就被关掉，下次请求重新握手。工程上要显式配置 `Transport`，并且**整个进程共用一个 `http.Client`**。

### 最小可运行示例

```go
// NewTransport 返回带连接池参数的 Transport。
func NewTransport() *http.Transport {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{}
	}

	transport := base.Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 10                // 默认 2
	transport.IdleConnTimeout = 90 * time.Second      // 空闲连接多久回收
	transport.ResponseHeaderTimeout = 5 * time.Second // 只等响应头
	transport.TLSHandshakeTimeout = 5 * time.Second   // TLS 握手超时
	return transport
}
```

### 实测输出

```text
保留空闲连接: 3 次请求新建 1 条连接，复用 2 次
DisableKeepAlives: 3 次请求新建 3 条连接，复用 0 次
连接池配置: MaxIdleConns=100 MaxIdleConnsPerHost=10 IdleConnTimeout=1m30s ResponseHeaderTimeout=5s
```

统计方式是用 `httptrace` 挂钩子，`GotConn` 回调里的 `Reused` 字段告诉你这条连接是新拨的还是复用的：

```go
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				t.reused++
				return
			}
			t.newConns++
		},
	}
```

### 边界与坑

- **每次 `new(http.Transport)` 等于每次新建连接池**：最常见的性能事故就是「每个请求 new 一个 Client」。把 Client 作为依赖注入，全局复用。
- **`MaxIdleConnsPerHost` 默认 2**：对下游并发 10 时，会有 8 条连接在请求结束后被关闭，下一轮再握手。按下游并发量把它调到同一量级。
- **改配置要新建 Transport**：`Transport` 的很多字段并发读写不安全，运行期调参应当新建一个再替换，并把旧的 `CloseIdleConnections()`。
- **`ResponseHeaderTimeout` 不能替代 `Client.Timeout`**：它只管到响应头为止，body 读多久它不管。
- **`DisableKeepAlives` 只用于排查**：它能帮你确认「问题是否与复用有关」，但线上开着会让每次请求都重新握手。

---

## 27.3 构造请求：JSON、表单与请求头

### 结论

请求一律用 `http.NewRequestWithContext` 构造（**不要用 `http.Get` 拼字符串**），JSON 用 `json.Marshal` 编码后设置 `Content-Type: application/json`，表单用 `url.Values.Encode()` 并设置 `application/x-www-form-urlencoded`。返回值、路径参数、请求头都要显式处理。

### 最小可运行示例

```go
// do 构造并发送请求，调用方负责关闭响应体。
func (a *API) do(ctx context.Context, method, path string, body io.Reader, contentType string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, body)
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return a.client.Do(req)
}
```

### 实测输出

```text
JSON 回显: message=hello count=2
上游看到的头: Content-Type=application/json User-Agent=go-learn/1.0 X-Request-ID=req-1
表单回显: name=gopher tag=go,http
表单请求次数（上游侧统计）: /api/form = 1
```

最后一行是假上游侧的计数（`up.hitsFor("/api/form")`）。能这样断言「调用了几次、带了什么头」，正是把外部依赖换成 `httptest` 的价值。

### 边界与坑

- **`http.NewRequestWithContext` 传 nil context 会返回错误**：`net/http: nil Context`，不是 panic，但一定要检查这个 error（见报错 1）。
- **路径参数要转义**：`url.PathEscape(id)`。ID 里出现 `/`、空格或中文时，不转义会拆坏路径。
- **别忘了 `Content-Type`**：缺它时很多网关直接回 415，或者按 `text/plain` 解析你的 JSON。
- **幂等性头**：对「创建」类接口，带上 `Idempotency-Key`，配合服务端去重，重试才安全。
- **别在循环里反复 `json.Marshal` 同一个结构**：结构固定时预编码一次，循环里只换 URL。

---

## 27.4 处理响应：状态码、gzip 与 Body 关闭

### 结论

响应处理有三件事必须做对：**判定状态码**（`Do` 不把 4xx/5xx 当错误）、**关闭 Body**（每个响应都要关）、**按需限制读取量**（用 `io.LimitReader`）。另外 gzip 是否自动解压，取决于 `Accept-Encoding` 是谁设的。

### 最小可运行示例

```go
// checkStatus 把非 2xx 转成 *StatusError，并关闭响应体。
func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	defer resp.Body.Close()

	// 只读一小段错误信息：错误页可能很大，没必要全部读进内存。
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return &StatusError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
}

// decodeJSON 解析 JSON 响应，并保证响应体被关闭。
func decodeJSON(resp *http.Response, dst any) error {
	defer resp.Body.Close()
	return json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(dst)
}
```

### 实测输出

```text
404 响应: 上游返回 404: {"code":"Not Found","message":"由 /api/status 生成"}
errors.As 拿到 *StatusError: 状态码=404 可重试=false
503 响应: 上游返回 503: {"code":"unavailable","message":"上游暂时不可用"}
解析 Retry-After: 等待 1s，可重试=true
自动解压: Content-Encoding="" 响应体={"note":"响应体是 gzip 压缩的","status":"ok"}
手动指定 Accept-Encoding: Content-Encoding="gzip" 前两个字节=0x1f 0x8b（gzip 魔数，未解压）
响应体关闭方式对连接复用的影响:
  读完再关闭: 2 次请求新建 1 条连接，复用 1 次
  只读 1 字节就关闭（1 MB）: 2 次请求新建 2 条连接，复用 0 次
  只读 1 字节就关闭（64 KB）: 2 次请求新建 1 条连接，复用 1 次（小响应体会被排空）
```

这三行连接复用是本章最值得记的实测结果：

- **读完再关闭**：连接回到池里，下次复用。
- **大响应体（1 MB）只读 1 字节就关闭**：连接直接丢弃，下次新建——这就是官方文档说「Body 没读完不能复用连接」的含义。
- **小响应体（64 KB）只读 1 字节就关闭**：连接仍然可以复用，因为传输层会把剩余内容排空（`Close()` 时的排空上限是 256 KB 量级，属于实现细节，别写进业务逻辑）。**这层「排空」会掩盖问题**，所以别指望小接口上的坏习惯在大接口上不出事。

gzip 的两行说明：不设 `Accept-Encoding` 时，`Transport` 自己加上 `gzip` 并**自动解压**，同时把 `Content-Encoding` 头删掉；一旦你自己设了 `Accept-Encoding: gzip`，自动解压就关闭了，拿到的是压缩字节（`0x1f 0x8b` 是 gzip 魔数）。

### 边界与坑

- **`Do` 不会因为 4xx/5xx 返回错误**：错误只表示「请求没发出去/响应没收到」。状态码必须自己判。
- **错误响应也要关 Body**：`checkStatus` 里 `defer resp.Body.Close()`，错误页不读也得关，否则连接泄漏。
- **手动设 `Accept-Encoding` 会关掉自动解压**：需要自己包 `gzip.NewReader`，或干脆别设。
- **想复用连接就 `io.Copy(io.Discard, resp.Body)`**：只需要前面一小段时，接受连接被关闭，或者显式读完（读完再丢弃，代价是流量）。
- **限制单次读取量**：`io.LimitReader` + `resp.ContentLength` 校验，避免下游返回一个超大 body 把内存打满。
- **`resp.Body` 是流**：不要先 `ReadAll` 再判断大小，顺序反了就等于没有防御。

---

## 27.5 重试与退避

### 结论

重试的四个前提：**只重试可重试的错误**（网络层抖动、429、5xx）、**指数退避 + 抖动**、**尊重 `Retry-After`**、**有总预算**（最大次数 + context 超时）。4xx 不重试；调用方取消或超时不重试；非幂等的写请求要么不重试，要么靠幂等键。

### 最小可运行示例

```go
// Do 按策略执行 fn，返回实际尝试次数与最后一次错误。
func (p RetryPolicy) Do(ctx context.Context, fn func(context.Context) error) (int, error) {
	attempts := 0
	var lastErr error

	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		attempts = attempt
		if lastErr = fn(ctx); lastErr == nil {
			return attempts, nil
		}
		if attempt == p.MaxAttempts || !Retryable(lastErr) {
			return attempts, lastErr
		}
		if err := p.sleep(ctx, p.delay(attempt, RetryAfter(lastErr))); err != nil {
			return attempts, err
		}
	}
	return attempts, lastErr
}

// Retryable 判断错误是否值得重试。
func Retryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.Retryable()
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}
```

### 实测输出

```text
上游先失败 2 次: 尝试 3 次，退避 [10ms 20ms]，最终错误=<nil>
400 不可重试: 尝试 1 次，退避 []，最终错误=上游返回 400: {"code":"Bad Request","message":"由 /api/status 生成"}
503 带 Retry-After=1 秒: 尝试 2 次，退避 [1s]，最终错误=<nil>
退避期间取消: 尝试 1 次，最终错误=context canceled（errors.Is(context.Canceled)=true）
```

四行对应四种判定：指数退避（10ms → 20ms）、4xx 不重试、`Retry-After` 优先于指数退避、退避期间取消立刻退出。演示里的「等待」是注入的假实现，所以跑起来没有真的 sleep，输出也完全确定：

```go
	policy := NewRetryPolicy().WithSleep(func(_ context.Context, d time.Duration) error {
		delays = append(delays, d)
		return nil
	}).WithJitter(nil)
```

### 边界与坑

- **抖动（jitter）不能省**：所有客户端在同一时刻指数退避、又同时重试，会把刚恢复的下游再打一遍。`randomJitter` 在 `[d/2, d]` 之间取值。
- **重试要有总预算**：`MaxAttempts=4` 加 `BaseDelay=10ms` 大约是几十毫秒；如果底层 `Client.Timeout` 是 30 秒，最坏情况会叠加到两分钟。用 context 给整次调用设上限。
- **写请求谨慎重试**：POST 重试可能创建两份订单。要么服务端支持 `Idempotency-Key` 去重，要么只重试「请求根本没发出去」的错误。
- **不要重试所有错误**：参数错误、鉴权失败、资源不存在，重试多少次都一样。
- **`Retry-After` 可能是 HTTP 日期**：`parseRetryAfter` 两种格式都要处理，并且要封顶（否则一个 `Retry-After: 3600` 能把你挂住一小时）。
- **重试要打日志**：记录尝试次数与最终结果，否则线上只能看到「变慢了」而不知道原因。

---

## 27.6 上传与流式下载

### 结论

上传用 `mime/multipart` 的 `Writer`（`Content-Type` 必须用 `writer.FormDataContentType()`），下载用 `io.Copy` 直接落到目标（文件或哈希），**不要先 `ReadAll` 到内存**。

### 最小可运行示例

```go
// UploadFile 以 multipart/form-data 上传一段内容。
func (a *API) UploadFile(ctx context.Context, field, name string, content []byte) (UploadResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		return UploadResult{}, fmt.Errorf("创建上传字段: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return UploadResult{}, fmt.Errorf("写入上传内容: %w", err)
	}
	if err := writer.Close(); err != nil {
		return UploadResult{}, fmt.Errorf("结束 multipart 编码: %w", err)
	}

	resp, err := a.do(ctx, http.MethodPost, "/api/upload", &body, writer.FormDataContentType(), nil)
	// ...（检查状态码、解析响应）
}

// DownloadTo 把响应体流式写入 w，返回写入字节数与内容的 sha256。
func (a *API) DownloadTo(ctx context.Context, w io.Writer, path string) (int64, string, error) {
	// ...（发请求、检查状态码）
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(w, hasher), resp.Body)
	if err != nil {
		return written, "", fmt.Errorf("下载中断: %w", err)
	}
	return written, hex.EncodeToString(hasher.Sum(nil)), nil
}
```

### 实测输出

```text
上传 1152 字节: 文件名=notes.txt 大小=1152 上游算出 sha256=1919b04ae3d3…
本地 sha256 与上游一致: true
流式下载: 写入 8192 字节，sha256=d281ea21d2bc…（缓冲区里就是最终内容）
只要前 100 字节: 读到 100 字节就关闭，剩下的不回内存
```

`io.Copy(io.MultiWriter(w, hasher), resp.Body)` 是流式下载的标准写法：内容一边落到目标、一边喂给哈希，内存占用与文件大小无关。

### 边界与坑

- **boundary 不能手写**：`Content-Type` 必须来自 `writer.FormDataContentType()`，手写 `multipart/form-data` 会缺 `boundary`，服务端直接解析失败。
- **大文件上传要流式**：`bytes.Buffer` 会把整个文件读进内存。用 `io.Pipe` + goroutine，把文件读到管道、把管道的读端交给请求。
- **`Content-Length` 与进度**：`resp.ContentLength` 可能是 -1（chunked），做进度条前要判断；上传侧用 `io.Pipe` 时长度未知，也是 chunked。
- **下载完校验摘要**：网络中断时 `io.Copy` 会返回错误，但要防的是「响应被截断却返回 nil」这类上游问题，例如报错 5 的 `unexpected EOF`。
- **临时文件要清理**：下载到磁盘时用 `defer os.Remove`，或者写在 `os.MkdirTemp` 出来的目录里。

---

## 27.7 测试外部依赖、TLS 与重定向

### 结论

测试里**不要打真实网络**：用 `httptest.NewServer` 造一个假上游，测得出的东西比 mock 更多（真实的序列化、状态码、头、超时）。TLS 用 `httptest.NewTLSServer`，它的 `Client()` 已经信任自签证书；重定向行为用 `CheckRedirect` 控制。

### 最小可运行示例

```go
// 假上游：一个 httptest.Server + 若干处理器，还能统计每个路径被请求了几次
up := newUpstream()
defer up.Close()

api := NewAPI(up.URL(), NewClient())
user, err := api.User(ctx, "7")

// 断言「调用了几次、带了什么头」
if got := up.hitsFor("/api/users/7"); got != 1 {
	t.Errorf("上游应当只被调用一次，实际 %d", got)
}
```

### 实测输出

```text
用 httptest 的客户端访问自签 HTTPS: 200 {"scheme":"https"}
换成默认客户端: 失败=true，证书校验错误=true
默认跟随重定向: 状态=200 最终路径=/api/users/1
CheckRedirect=ErrUseLastResponse: 状态=302 Location=/api/users/1
上游侧统计: /api/users/1 被请求 1 次（测试里就能这样断言调用次数）
```

`httptest.NewTLSServer` 的 `Client()` 内置了信任该证书的 Transport，所以第一行能成功；换成 `NewClient()` 就变成证书校验失败（见报错 2）。重定向两行则说明默认策略会跟随 302，而 `CheckRedirect` 返回 `http.ErrUseLastResponse` 时会保留原始响应。

### 边界与坑

- **测试要断言副作用**：只断言「返回了 200」不够，还要断言上游收到的请求次数、方法、路径、请求体（`up.hitsFor` 就是干这个的）。
- **`httptest.NewTLSServer` 的证书只对该客户端有效**：别把 `srv.Client()` 的 Transport 用到生产代码里。
- **`CheckRedirect` 里别做重活**：它的返回值决定是否继续跟随；返回 `http.ErrUseLastResponse` 表示「不跟随，也不报错」，返回别的 error 会让请求以错误结束。
- **代理配置**：`Transport.Proxy` 支持环境变量（`HTTP_PROXY`）和自定义函数；代理不可用时错误形态是 `proxyconnect tcp: ...`（见报错 6），很容易被误判成「对方挂了」。
- **测试超时要用短超时**：`httptest` 的假上游里 `time.Sleep` 一下，就能稳定复现超时路径，比打真实网络可靠。
- **连接复用也可以断言**：像 27.2 那样用 `httptrace` 统计，把「复用」写成测试，防止有人不小心改成每个请求 new 一个 Client。

---

## 6 个真实报错怎么读

下面每一条都是在本仓库实测跑出来的输出。路径里的目录名取决于你把示例放在哪儿。

### 报错 1：context 传 nil → 拿到一个 error

```go
// 忘记传 context，直接给了 nil
req, err := http.NewRequestWithContext(nil, http.MethodGet, "http://example.com", nil)
```

```text
err = net/http: nil Context req = <nil>
```

注意它不是 panic，而是**返回了一个 error 和一个 nil 的请求**。如果代码写成 `req, _ := http.NewRequestWithContext(...)`，下一步调用 `req.Header` 就会空指针 panic，而且堆栈离真正的原因很远。**修复**：检查这个 error，或者用 `context.Background()` / 上层传下来的 ctx。

### 报错 2：自签证书 → 证书校验失败

```go
server := httptest.NewTLSServer(handler) // 自签证书
resp, err := http.Get(server.URL)        // 用默认客户端访问
```

```text
请求失败: Get "https://127.0.0.1:59841": tls: failed to verify certificate: x509: certificate signed by unknown authority
```

链路是「TLS 校验失败 → 具体原因是 x509 找不到签发者」。**修复分场景**：测试里用 `server.Client()`（已信任该证书）；访问内网自签服务时把 CA 装进 `Transport.TLSClientConfig.RootCAs`；**不要**在生产代码里写 `InsecureSkipVerify: true`。

### 报错 3：用 HTTPS 客户端去连明文 HTTP 服务

```go
server := httptest.NewServer(handler)                                  // 明文 HTTP
target := strings.Replace(server.URL, "http://", "https://", 1)        // 客户端按 https 访问
```

```text
请求失败: Get "https://127.0.0.1:59845": http: server gave HTTP response to HTTPS client
```

这句话的意思是「我以为对面会说 TLS，结果对面直接回了一句 HTTP」。常见于协议配错（`http://` 写成 `https://`，或反向代理没有开 TLS 终止）。**修复**：确认端点协议；如果确实要连明文，就用 `http://`。

### 报错 4：重定向成环 → stopped after 10 redirects

```go
// 永远跳回自己
http.Redirect(w, r, "http://"+r.Host+"/loop", http.StatusFound)
```

```text
请求失败: Get "http://127.0.0.1:59877/loop": stopped after 10 redirects
```

默认策略最多跟随 10 次重定向，第 11 次直接报错。**修复**：先修服务端的跳转逻辑；确实需要更多次时用 `CheckRedirect` 自定义，但更常见的是「不该跟随」——比如内部网关的 302 你只想要它本身，就返回 `http.ErrUseLastResponse`。

### 报错 5：声明 100 字节却只写了 10 字节 → unexpected EOF

```go
w.Header().Set("Content-Length", "100")
_, _ = w.Write([]byte("0123456789"))
```

```text
读到 10 字节，错误: unexpected EOF
```

读到了 10 个字节，但 `Content-Length` 承诺的是 100，客户端在等待剩余数据时发现连接结束了。**修复**：读的时候必须检查 error（`io.Copy` / `ReadAll` 都会返回它），把「下载失败」和「下载成功但内容不完整」区分开；下载完再校验摘要或长度。

### 报错 6：代理不可用 → proxyconnect 失败

```go
transport := http.DefaultTransport.(*http.Transport).Clone()
transport.Proxy = http.ProxyURL(proxy) // http://127.0.0.1:1
```

```text
请求失败: Get "http://example.com/": proxyconnect tcp: dial tcp 127.0.0.1:1: connectex: No connection could be made because the target machine actively refused it.
```

前缀 `proxyconnect` 是关键：出错的是**连代理**这一步，不是连目标站点。Windows 上显示 `connectex: ...`，Linux 上通常是 `connect: connection refused`。**修复**：检查代理地址/端口、代理进程是否存活、`NO_PROXY` 是否把不该走代理的地址排除掉了。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 请求永远不返回，或大文件下载到一半失败 | `DefaultClient` 没设超时（默认 0）；`Client.Timeout` 覆盖到读 body 阶段 | 设 `Client.Timeout` 并给每次调用传带 deadline 的 context；大文件接口单独放宽或用 context 分阶段控制 |
| 重试把下游打垮 | 没有抖动，所有客户端同时重试 | 指数退避 + `[d/2, d]` 抖动，并设最大次数 |
| 4xx 还在重试 | 没做可重试判定 | 只重试网络错误、429、5xx；4xx 直接失败 |
| 创建接口重试出两份数据 | POST 非幂等 | 服务端支持 `Idempotency-Key`，或只重试发送阶段错误 |
| 句柄/TIME_WAIT 飙升，或连接复用率低 | 响应体没关、每次 new 一个 Client；`MaxIdleConnsPerHost` 默认只有 2 | 每个响应 `defer resp.Body.Close()`、全局复用 Client，并按下游并发调大连接池、设置 `IdleConnTimeout` |
| 大响应体不读完，连接每次新建 | Body 没读满就关闭 | 读完再关；只想取头部就接受连接被关闭 |
| 拿到了乱码或解压失败 | 自己设了 `Accept-Encoding: gzip`，自动解压被关掉 | 不要手设，或用 `gzip.NewReader` 自行解包 |
| 415 / 400 各占一半 | 缺 `Content-Type` 或 JSON 字段不匹配 | 显式设置 `Content-Type`，用 `DisallowUnknownFields` 校验 |
| 测试偶发失败 | 打真实网络、依赖真实延时 | 用 `httptest` 假上游 + 注入时钟，断言请求次数与请求头 |
| 证书错误只在生产出现 | 内网自签 CA 没装进 `RootCAs` | 把 CA 加进 `TLSClientConfig.RootCAs`，不要关校验 |

---

## 练习

### 第 1 题

下面这段客户端代码有四个问题，找出来并修复：

```go
func FetchUser(id string) (User, error) {
	resp, err := http.Get("https://api.example.com/users/" + id)
	if err != nil {
		return User{}, err
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return User{}, err
	}
	return user, nil
}
```

::: details 第 1 题参考答案

```go
func FetchUser(ctx context.Context, client *http.Client, id string) (User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.example.com/users/"+url.PathEscape(id), nil)
	if err != nil {
		return User{}, fmt.Errorf("构造请求: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req) // client 注入进来，全局复用同一个连接池
	if err != nil {
		return User{}, fmt.Errorf("请求用户 %s: %w", id, err)
	}
	defer resp.Body.Close() // 无论成功失败都要关

	if err := checkStatus(resp); err != nil { // http.Get 不会把 404 当错误
		return User{}, err
	}

	var user User
	if err := decodeJSON(resp, &user); err != nil {
		return User{}, fmt.Errorf("解析用户响应: %w", err)
	}
	return user, nil
}
```

**四个问题**：① `http.Get` 用的是 `DefaultClient`（没有超时，而且连接池不可控），要注入 `*http.Client` 并传 context；② 没有检查状态码——404 的响应体会被当作用户数据解析，报出「解析失败」这种误导性错误；③ `resp.Body` 没有关闭，连接泄漏；④ 错误没有上下文（谁、哪个 id 失败了）。另外路径参数用 `url.PathEscape` 转义，避免 id 里有 `/` 时请求到别的路径。

:::

### 第 2 题

把本章的 `RetryPolicy` 改造成「总预算优先」的版本：除了最大次数，再接受一个 `MaxElapsed`，累计等待时间超过预算就停止重试。

::: details 第 2 题参考答案

```go
// RetryPolicy 描述重试策略：最多试几次、退避多长、总预算多久。
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxElapsed  time.Duration // 总预算：累计等待时间超过它就放弃

	sleep  func(context.Context, time.Duration) error
	jitter func(time.Duration) time.Duration
}

// Do 按策略执行 fn；返回尝试次数、最后一次错误与已等待的总时间。
func (p RetryPolicy) Do(ctx context.Context, fn func(context.Context) error) (int, error) {
	attempts := 0
	var elapsed time.Duration
	var lastErr error

	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		attempts = attempt
		if lastErr = fn(ctx); lastErr == nil {
			return attempts, nil
		}
		if attempt == p.MaxAttempts || !Retryable(lastErr) {
			return attempts, lastErr
		}

		delay := p.delay(attempt, RetryAfter(lastErr))
		if p.MaxElapsed > 0 && elapsed+delay > p.MaxElapsed {
			return attempts, fmt.Errorf("重试预算 %v 用尽: %w", p.MaxElapsed, lastErr)
		}
		if err := p.sleep(ctx, delay); err != nil {
			return attempts, err
		}
		elapsed += delay
	}
	return attempts, lastErr
}
```

**为什么这样写更好**：只限次数不限总时长，遇到 `Retry-After: 60` 这类响应会挂很久；反过来只限时长不限次数，可能在毫秒级错误上疯狂重试。两个一起用才完整。错误用 `%w` 包住最后一次失败，`errors.Is` / `errors.As` 依然能拿到原始原因；把「预算用尽」和「请求失败」区分开，调用方才能决定是降级还是报错。

:::

### 第 3 题

用 `httptest` 为「超时后重试」写一个测试：假上游第一次故意慢（超过单次超时），第二次立刻返回 200，断言最终成功且尝试了 2 次。

::: details 第 3 题参考答案

```go
func TestRetryAfterTimeout(t *testing.T) {
	var calls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			time.Sleep(200 * time.Millisecond) // 第一次超过客户端超时
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	defer up.Close()

	client := NewClient()
	client.Timeout = 50 * time.Millisecond

	api := NewAPI(up.URL(), client)
	policy := NewRetryPolicy().WithSleep(func(context.Context, time.Duration) error {
		return nil // 测试里不真的等
	}).WithJitter(nil)

	attempts, err := policy.Do(context.Background(), func(ctx context.Context) error {
		resp, err := client.Get(up.URL())
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return checkStatus(resp)
	})
	if err != nil {
		t.Fatalf("第二次应当成功，实际 %v", err)
	}
	if attempts != 2 {
		t.Errorf("尝试次数：期望 2，实际 %d", attempts)
	}
	if calls.Load() != 2 {
		t.Errorf("上游应当被调用 2 次，实际 %d", calls.Load())
	}
}
```

**为什么这样写更好**：用假上游控制「第一次慢、第二次快」，超时与重试两条路径都能稳定复现，不需要依赖真实网络抖动；`atomic.Int32` 统计上游被调用的次数，把「客户端重试了」这件事从「看起来成功了」变成可断言的事实；注入空实现的 `sleep` 让测试不真的等待，跑得快也不会偶发失败。注意这里重试的是**超时**——它属于网络层错误，可重试；如果是 context 被取消，就不该重试。

:::

### 第 4 题

写一个流式上传：把一个 10 MB 的本地文件传到 `/api/upload`，要求内存占用与文件大小无关。

::: details 第 4 题参考答案

```go
// UploadFileStream 用 io.Pipe 把文件「边读边发」，不在内存里放整个文件。
func (a *API) UploadFileStream(ctx context.Context, path, field string) (UploadResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return UploadResult{}, fmt.Errorf("打开文件: %w", err)
	}
	defer file.Close()

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)

	// 在另一个 goroutine 里编码 multipart，主 goroutine 同时把它作为请求体发出去
	go func() {
		defer writer.Close()

		part, err := multipartWriter.CreateFormFile(field, filepath.Base(path))
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			writer.CloseWithError(err)
			return
		}
		if err := multipartWriter.Close(); err != nil {
			writer.CloseWithError(err)
		}
	}()

	resp, err := a.do(ctx, http.MethodPost, "/api/upload", reader, multipartWriter.FormDataContentType(), nil)
	if err != nil {
		return UploadResult{}, fmt.Errorf("上传: %w", err)
	}
	if err := checkStatus(resp); err != nil {
		return UploadResult{}, err
	}

	var result UploadResult
	if err := decodeJSON(resp, &result); err != nil {
		return UploadResult{}, fmt.Errorf("解析上传响应: %w", err)
	}
	return result, nil
}
```

**为什么这样写更好**：`io.Pipe` 把「编码 multipart」和「发送 HTTP 请求」变成两条并发的流水线，任何时刻内存里只有一个缓冲区；`CreateFormFile` 的字段名与文件名都来自参数，`FormDataContentType()` 保证 boundary 正确。两个注意点：① 出错时要用 `CloseWithError` 把错误传给读端，否则请求会挂住；② 因为长度未知，请求会走 chunked 编码，服务端就不能依赖 `Content-Length`。

:::

### 第 5 题

写一个测试，断言「使用同一个 `http.Client` 时，连续 5 次请求只建立 1 条 TCP 连接」。

::: details 第 5 题参考答案

```go
func TestClientReusesSingleConnection(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	defer up.Close()

	counter := &countingTransport{base: NewTransport()}
	client := &http.Client{Timeout: 3 * time.Second, Transport: counter}

	for i := 0; i < 5; i++ {
		resp, err := client.Get(up.URL)
		if err != nil {
			t.Fatalf("第 %d 次请求失败: %v", i+1, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body) // 必须读完，否则连接不会回到池里
		resp.Body.Close()
	}

	newConns, reused := counter.stats()
	if newConns != 1 {
		t.Errorf("期望只建立 1 条连接，实际 %d 条", newConns)
	}
	if reused != 4 {
		t.Errorf("期望复用 4 次，实际 %d 次", reused)
	}
}
```

**为什么这样写更好**：把「连接复用」从经验变成了可执行的断言——一旦有人把代码改成每个请求 new 一个 `http.Client`，这个测试立刻变红。两个细节：① 必须 `io.Copy(io.Discard, resp.Body)` 把 body 读干净，否则连接不会回到池里（27.4 实测过）；② 用 `httptrace` 统计比看耗时可靠，后者会被机器负载干扰。

:::

### 第 6 题

生产环境要调用一个第三方 API，对方要求：每个请求带 `Authorization` 头、限流时返回 429 + `Retry-After`、偶发 502。写出你的客户端配置与调用策略，并说明每一层的取舍。

::: details 第 6 题参考答案

```go
type ThirdParty struct {
	base   string
	token  string
	client *http.Client
	policy RetryPolicy
}

func NewThirdParty(base, token string) *ThirdParty {
	transport := NewTransport()
	transport.MaxIdleConnsPerHost = 20 // 对方是主要依赖，连接池按并发量给足

	return &ThirdParty{
		base:  strings.TrimRight(base, "/"),
		token: token,
		client: &http.Client{
			Timeout:   8 * time.Second, // 单次调用预算：对方 SLA 通常也有超时
			Transport: transport,
		},
		policy: NewRetryPolicy().WithSleep(sleepContext), // 线上用真实等待
	}
}

func (t *ThirdParty) Call(ctx context.Context, path string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second) // 整次调用（含重试）的总预算
	defer cancel()

	var resp *http.Response
	_, err := t.policy.Do(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.base+path, nil)
		if err != nil {
			return fmt.Errorf("构造请求: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+t.token)
		req.Header.Set("Accept", "application/json")

		var err error
		resp, err = t.client.Do(req)
		if err != nil {
			return err // 网络层错误：可重试
		}
		if err := checkStatus(resp); err != nil {
			return err // 429/5xx：可重试；4xx：Retryable 会返回 false
		}
		return nil
	})
	return resp, err
}
```

**分层取舍**：

1. **连接池**（`Transport`）：对方是主要依赖，`MaxIdleConnsPerHost` 按并发量配置，避免每次请求重新握手。
2. **单次超时**（`Client.Timeout=8s`）：不能比对方的 SLA 还宽，否则重试时旧请求还占着资源。
3. **总预算**（`context.WithTimeout(20s)`）：包住「单次超时 × 重试次数 + 退避」的最坏情况，防止一次上游抖动把调用方的 goroutine 全挂住。
4. **重试策略**：429 与 502 可重试，尊重 `Retry-After`（客户端用 `parseRetryAfter` 解析），指数退避 + 抖动，最大 4 次。
5. **鉴权头**：`Authorization` 只在客户端加，别写进日志；`checkStatus` 里保存错误体时要注意脱敏。
6. **可观测**：每次重试记录一次日志（路径、状态码、尝试次数、耗时），并给上游调用加指标，否则线上只能看到「偶发变慢」。

**为什么不能更简单**：只设 `Client.Timeout` 会在重试叠加后远超调用方预期；只做重试不做预算会在上游长时间不可用时放大故障；把 4xx 也重试则纯属浪费配额（对方的 400 不会因为重试变成 200）。

:::

---

## 小结

- **超时分三层**：`Client.Timeout`（整次调用）、context（可传递可取消）、`Transport` 的阶段超时；至少设一层，`DefaultClient` 的默认值是永不超时。
- **取消和超时不重试**：`errors.Is` 能区分 `context.Canceled` / `context.DeadlineExceeded`，这两类属于调用方的决定；网络层错误与 429/5xx 才值得重试。
- **连接池是 Client 级别的**：全局复用一个 `http.Client`，把 `MaxIdleConnsPerHost` 从默认的 2 调到与并发匹配，并设置 `IdleConnTimeout`；复用情况用 `httptrace` 的 `GotConn.Reused` 验证，最好写成测试。
- **请求一律用 `NewRequestWithContext`**：路径参数转义、`Content-Type` 显式设置、错误要检查（nil context 会返回 `net/http: nil Context`）。
- **`Do` 不把 4xx/5xx 当错误**：状态码必须自己判，失败路径也要关 Body，错误体只用 `io.LimitReader` 读一小段；gzip 的自动解压有条件——自己设了 `Accept-Encoding` 就拿到压缩字节，要么不设，要么自己 `gzip.NewReader`。
- **想复用连接就读完 Body**：大响应体不读完会丢连接（实测 256 KB 只读 1 字节即断开），小响应体会被排空而掩盖问题。
- **重试要能收敛**：指数退避 + 抖动、尊重 `Retry-After`、有最大次数和总预算，写请求靠幂等键。
- **上传下载都走流**：multipart 的 `Content-Type` 来自 `FormDataContentType()`，下载用 `io.Copy` + `MultiWriter` 边落盘边算摘要。
- **测试用 `httptest` 造替身**：断言请求次数、请求头、状态码与超时路径；TLS 用 `srv.Client()`，重定向用 `CheckRedirect` 控制。

下一章将讲解 **数据库编程**：`database/sql` 与驱动、`sql.DB` 连接池参数、预处理语句与 SQL 注入、NULL 值处理、事务与回滚，以及用 context 给查询设置超时。
