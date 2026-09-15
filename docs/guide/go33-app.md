# 第 33 章 · 综合实战：Go 服务端项目

第 32 章把工程实践拆开讲了一遍：怎么分目录、怎么分层、怎么注入依赖、怎么写配置、怎么读报错。这一章把它们**串成一个真正能跑的服务**——一个最小的「用户管理」后端，用标准库 `net/http` 加一个 `mattn/go-sqlite3` 驱动完成，不引入 Web 框架。

你会看到一条请求怎么穿过 **handler → service → repository** 三层，错误码怎么从底层一路带回到 HTTP 状态码，中间件链怎么把请求 ID、访问日志和 panic 恢复叠在一起，最后服务又怎么在收到退出信号时优雅关闭。更重要的是配套的单元测试、集成测试和基准测试——因为一个「能跑」的服务和一个「敢改」的服务之间，差的就是测试。

本章配套代码在 `internal/chapter/go33_app/`，执行 `go run ./cmd/go-learn` 可以看到全部输出；`go test ./internal/chapter/go33_app/` 跑测试，`go test ./internal/chapter/go33_app/ -bench . -benchmem` 跑基准。

## 33.1 项目骨架与依赖装配

先看结论：**一个服务端项目不是「一个大 main」**，而是把装配、路由、业务、数据、协议、运行期各自放进一个文件，边界清晰、职责单一。本章的包一共 10 个文件：

```
internal/chapter/go33_app/
├─ app.go        # App 装配 + 路由 + handler
├─ apierror.go   # 错误码 + JSON 写回
├─ user.go       # 领域对象 + service（业务层）
├─ sqlite.go     # SQLite 仓储（数据层） + 迁移
├─ middleware.go # 请求 ID / 访问日志 / panic 恢复
├─ server.go     # http.Server 超时 + 优雅关闭
├─ config.go     # 配置加载
├─ demo.go       # 本章演示入口
├─ app_test.go   # 单元 + 集成测试
└─ bench_test.go # 基准测试
```

`main` 里真正要写的装配只有三行（本章用 `Demo()` 替代了 `main`）：

```go
// db := sql.Open("sqlite3", cfg.DSN) -> Migrate -> NewSQLiteUserRepo(db)
svc := NewUserService(repo)      // 业务依赖接口
app := NewApp(svc, logger)       // 挂路由 + 中间件
```

依赖是**手动注入**的：`UserService` 只依赖 `UserRepository` 接口，不关心后面是 SQLite 还是别的。`App` 用 Functional Options 接收可选配置（`WithRequestIDGen`），既保留了简洁的零配置默认值，又给了测试一个注入点。这套「消费方定义接口 + 构造器注入 + Options」的组合，在 Go 工程里反复出现，值得在这里记住一次。

## 33.2 配置加载：默认值 < 环境变量 < 校验

配置的优先级从低到高是：**默认值 → 环境变量 → 启动时校验**。默认值让程序「开箱就能跑」，环境变量覆盖它，最后一步校验把非法值挡在启动阶段，而不是让它跑到运行时才炸。

`config.go` 把「读环境变量」抽成一个函数 `lookup`，是为了可测试：

```go
func LoadConfigFrom(lookup func(string) (string, bool)) (*Config, error) {
	cfg := &Config{Addr: ":8080", DSN: ":memory:", LogLevel: "info"}
	if v, ok := lookup("APP_ADDR"); ok && v != "" {
		cfg.Addr = v
	}
	if v, ok := lookup("APP_DSN"); ok && v != "" {
		cfg.DSN = v
	}
	if v, ok := lookup("APP_LOG_LEVEL"); ok && v != "" {
		if !validLevel(v) {
			return nil, badRequest("APP_LOG_LEVEL 非法: %q", v)
		}
		cfg.LogLevel = v
	}
	return cfg, nil
}
```

`LoadConfig()` 只是 `LoadConfigFrom(os.LookupEnv)` 的一个别名。真实输出：

```
无环境变量：   &{Addr::8080 DSN::memory: LogLevel:info}
覆盖后：       &{Addr:127.0.0.1:9090 DSN:./data.db LogLevel:debug}
非法日志级别： APP_LOG_LEVEL 非法: "verbose"
```

坑在这：`APP_LOG_LEVEL` 只有 `debug/info/warn/error` 四种，别的值直接返回错误。**越早拒绝非法配置越省事**——等到 `slog` 内部解不出来再报错，排查成本高一个量级。

## 33.3 数据库迁移

迁移的最小可靠形态就是一句 `CREATE TABLE IF NOT EXISTS`：幂等，可以反复执行。`sqlite.go` 里：

```go
const schema = `
CREATE TABLE IF NOT EXISTS users (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	email      TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);`
```

跑完 `Migrate` 之后，直接查 `sqlite_master` 就能看到**实际建出来的**表结构：

```
CREATE TABLE users (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	email      TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
)
```

要点：`IF NOT EXISTS` 让迁移幂等；真实项目里迁移会拆成带版本号的多个文件，由工具（golang-migrate、goose 等）按序应用，而不是把建表语句堆在代码里。

这里有一个**内存 SQLite 特有的坑**必须讲透。`:memory:` 模式下，`database/sql` 连接池里的**每个连接各自拥有一个独立的数据库**。如果不加限制，第 1 次请求建了表、第 2 次请求换了个连接，就会报「no such table: users」。所以本例在打开连接后立刻限定单连接：

```go
db.SetMaxOpenConns(1) // :memory: 每个连接各有独立库，限定单连接才能共享数据
```

这行注释里的「为什么」比「做了什么」更重要——去掉它，程序就会在看似随机的时候报错。

## 33.4 分层调用链

一条创建请求穿过三层，每一层只干自己的事：

```go
// handler 层：解析 JSON、调 service、写响应
func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, badRequest("请求体不是合法 JSON: %v", err))
		return
	}
	u, err := a.svc.Create(r.Context(), req.Name, req.Email)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}
```

service 层做**参数校验**，repository 层做**数据访问**，谁的错误谁负责归一。实测输出（先建两个用户，再查回来）：

```
GET  /api/v1/users/1  -> 200 {"id":1,"name":"张三","email":"zhang@example.com"}
GET  /api/v1/users    -> 200 [{"id":1,"name":"张三","email":"zhang@example.com"},{"id":2,"name":"李四","email":"li@example.com"}]
```

注意第二行：空列表也会被编码成 `[]`，而不是 `null`，这是 `repository.List` 里 `users := make([]User, 0)` 的功劳。

## 33.5 REST 接口与统一错误码

失败响应统一成一个形状，机器按 `code` 分支、人看 `message`：

```json
{"code":"already_exists","message":"邮箱 \"zhang@example.com\" 已被占用"}
```

`apierror.go` 定义了贯穿三层的 `AppError`：

```go
type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"` // 状态码走 HTTP 传输层，不进响应体
}
```

底层用 `badRequest/notFound/conflict/internalErr` 四个构造器造错误，`writeError` 是**所有失败路径的唯一出口**，用 `errors.As` 把任意 `error` 映射成三要素：

```go
func responseOf(err error) (int, Code, string) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Status, ae.Code, ae.Message
	}
	return http.StatusInternalServerError, CodeInternal, "服务器内部错误"
}
```

关键：**非 `*AppError` 的错误一律归为 500/internal**，绝不让底层 SQL 报错泄露到响应体里。实测几种失败路径：

```
POST /api/v1/users          -> 400 {"code":"invalid_argument","message":"name 不能为空"}
POST /api/v1/users          -> 400 {"code":"invalid_argument","message":"email 格式错误: \"bad-email\""}
POST /api/v1/users          -> 201 {"id":1,"name":"张三","email":"zhang@example.com"}
POST /api/v1/users          -> 409 {"code":"already_exists","message":"邮箱 \"zhang@example.com\" 已被占用"}
GET  /api/v1/users/999      -> 404 {"code":"not_found","message":"用户 999 不存在"}
GET  /api/v1/users/abc      -> 400 {"code":"invalid_argument","message":"id 必须是整数: \"abc\""}
```

`code` → `status` 的对应关系是固定的：`invalid_argument → 400`、`not_found → 404`、`already_exists → 409`、`internal → 500`。用户重复邮箱触发的是 SQLite 唯一约束，`isUniqueViolation` 用驱动暴露的错误码判断，而不是比对错误字符串：

```go
var se sqlite3.Error
return errors.As(err, &se) && se.Code == sqlite3.ErrConstraint
```

这里 `errors.As` 的目标是 `sqlite3.Error` **值**而不是 `*sqlite3.Error` 指针——因为该驱动的 `Error()` 方法挂在值接收者上，返回的是一个值。类型搞错指针/值，`As` 会静默失败，冲突就永远识别不出来。

## 33.6 中间件：请求 ID / 访问日志 / panic 恢复

三个中间件叠成一条链，`chain` 保证列表第一个在最外层：

```go
func (a *App) Handler() http.Handler {
	return chain(a.mux, a.accessLogMW, a.recoverMW, a.requestIDMW)
}
```

展开就是 `accessLog(recover(requestID(mux)))`。为什么是这个顺序？

1. **requestID（最内层）**：最靠近业务 handler，先给响应头写 `X-Request-ID`。
2. **recover（中间）**：捕获业务 panic，转成 500，避免单项故障掐死整条连接。
3. **accessLog（最外层）**：包住 recover，所以**被 panic 救回的请求也能写出一条 `status=500` 的访问日志**。

访问日志能读到状态码，靠的是 `statusRecorder` 这个包装：

```go
type statusRecorder struct {
	http.ResponseWriter
	status int
}
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
```

实测输出（先请求 `/healthz`，再请求一条故意 panic 的路由）：

```
GET /healthz -> 200 {"status":"ok"}（响应头带 X-Request-ID）
GET /panic  -> 500 {"code":"internal","message":"panic: 故意触发的 panic"}
写出的访问日志（logger 经 With 去掉了时间戳）：
level=INFO msg=access method=GET path=/healthz status=200 request_id=req-1
level=ERROR msg="panic recovered" panic="故意触发的 panic"
level=INFO msg=access method=GET path=/panic status=500 request_id=req-2
```

注意最后一行：`/panic` 虽然 500 了，仍然有访问日志。如果顺序反过来——`recover(accessLog(...))`——panic 会直接从 `accessLog` 的 `next.ServeHTTP` 里穿出去，**那条访问日志就丢了**，这是下一节「反例」要展开的。

## 33.7 优雅关闭

HTTP 服务最容易漏掉两件事：**超时**和**优雅关闭**。`server.go` 把四类超时写死成常量，因为默认值全是 0（永不超时）——一个慢慢发请求头的连接就能一直占着一个 goroutine：

```go
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 5 * time.Second
)
```

优雅关闭的骨架是：`select` 监听 `ctx.Done()`，取消后调 `srv.Shutdown` 等待在途请求处理完，再回收 `Serve` 的返回值，避免 goroutine 泄漏：

```go
case <-ctx.Done():
shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
defer cancel()
if err := srv.Shutdown(shutdownCtx); err != nil {
	return fmt.Errorf("graceful shutdown: %w", err)
}
```

生产环境里 `ctx` 通常来自 `signal.NotifyContext`，收到 `SIGINT/SIGTERM` 就触发。演示是在 `127.0.0.1` 上一个随机端口拉起服务、请求一次、再取消 ctx：

```
监听地址（端口每次不同）: 127.0.0.1:64251
GET /healthz -> 200 {"status":"ok"}
取消 ctx 后 serveOn 返回: <nil>
服务器日志：level=INFO msg=access method=GET path=/healthz status=200 request_id=req-1
level=INFO msg=服务器已优雅关闭
```

端口「每次不同」是因为 `net.Listen("tcp", "127.0.0.1:0")` 让内核分配空闲端口，正文里点到这一点，避免读者把 `64251` 当成固定值抄。

## 33.8 测试与基准

这一章的质量保障分三层：

| 层 | 工具 | 覆盖 |
| --- | --- | --- |
| 单元测试 | `fakeRepo` 替身 | service 的校验逻辑与数据来源解耦 |
| 集成测试 | `httptest` + 内存 SQLite | 完整 HTTP 链路，含错误码 |
| 基准测试 | `testing.B` | 单次创建请求的纳秒开销 |

`fakeRepo` 实现了 `UserRepository` 接口，于是 `TestUserServiceCreate` 可以只验「空 name 报错、坏 email 报错、正常放行」，完全不碰数据库：

```go
func (f *fakeRepo) Create(_ context.Context, name, email string) (User, error) {
	f.nextID++
	u := User{ID: f.nextID, Name: name, Email: email}
	f.users = append(f.users, u)
	return u, nil
}
```

8 个测试全绿：

```
--- PASS: TestUserServiceCreate (0.00s)
--- PASS: TestUserServiceGet (0.00s)
--- PASS: TestConfigLoadFrom (0.00s)
--- PASS: TestResponseOf (0.00s)
--- PASS: TestAppCreateGetListFlow (0.00s)
--- PASS: TestAppErrorPaths (0.00s)
--- PASS: TestMiddlewareRequestID (0.00s)
--- PASS: TestMiddlewarePanicRecovery (0.00s)
```

基准测试度量单次创建请求的完整开销（本机 i7-14700F，每次数值都会浮动，这里只代表数量级）：

```
BenchmarkCreateUser-28    148880      7710 ns/op    7987 B/op      57 allocs/op
```

`7987 B/op、57 allocs/op` 提醒你：JSON 编解码 + `context` 传递是有固定内存开销的，热路径要减少这类分配。数字会随机器和 Go 版本变化，请以自己跑出来的为准。

## 33.9 Docker 打包

把服务装进容器，多阶段构建是标准姿势：编译期用 `golang` 镜像，运行期用精简镜像，最终只留二进制：

```dockerfile
# 阶段一：编译
FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/go-learn

# 阶段二：运行
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
```

要点说明：`CGO_ENABLED=0` 编出静态二进制，才能塞进没有 glibc 的 distroless 镜像；SQLite 这里用的是 CGO 版驱动（`mattn/go-sqlite3`），若要 `CGO_ENABLED=0`，得换成纯 Go 的 `modernc.org/sqlite` 驱动——这是第 34 章「部署与可观测性」会展开的话题。

> 本章演示环境未安装 Docker，因此上面的 `docker build` 未在本机执行验证；Dockerfile 以可复现的静态编译和镜像瘦身为目标给出，实际构建与镜像体积请以你本机 `docker build` 的实测为准。

## 3 个真实报错怎么读

**1. 未使用的导入：`"context" imported and not used`**

```
# go-learn/internal/chapter/go33_app
internal\chapter\go33_app\app.go:11:2: "context" imported and not used
```

`app.go` 一开始 import 了 `context`，但 handler 里用的是 `r.Context()`——它返回 `context.Context`，方法调用的返回值类型**不需要**显式 import。修复线索在行号：`app.go:11:2`，把那一行删掉即可。

**2. 未使用的导入：`"log/slog" imported and not used`**

```
internal\chapter\go33_app\middleware.go:4:2: "log/slog" imported and not used
```

同样的问题：`middleware.go` 里 `a.logger.Info(...)`、`a.logger.Error(...)` 只是方法调用，字段 `logger` 的类型声明在 `app.go`，所以这个文件不需要 `slog`。Go 对「导入未使用」直接报错而不是警告，反而让这类残留导入藏不住。

**3. 反例：把 recover 放到 accessLog 外面，会丢访问日志**

如果把链改成 `recover(accessLog(requestID(mux)))`，recover 在最外层。此时 `/panic` 的 panic 从 `accessLogMW` 的 `next.ServeHTTP` 里直接穿出去，`accessLogMW` 里那句 `a.logger.Info("access", ...)` 根本执行不到。结果：响应是 500 没错，但**这条失败请求在访问日志里消失了**。修复就是本章 33.6 的顺序——让 accessLog 包住 recover。中间件的顺序不是「都套上就行」，谁包谁直接决定日志和状态码的可见性。

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 内存 SQLite 偶发 `no such table: users` | `:memory:` 每个连接各自有独立库 | `db.SetMaxOpenConns(1)` 限定单连接共享 |
| 重复邮箱没被识别成 409，走成了 500 | `errors.As` 目标用成了 `*sqlite3.Error` 指针 | 驱动返回的是**值**，用 `var se sqlite3.Error` |
| `/panic` 后访问日志少一条 | recover 排在 accessLog 外层，panic 穿过了日志 | accessLog 放最外层，包住 recover |
| 空列表在 JSON 里是 `null` | 直接 `var users []User`（nil 切片） | 用 `make([]User, 0)` 返回空切片 |
| 非法配置拖到运行时才报错 | 不做启动校验 | `LoadConfigFrom` 里 `validLevel` 直接返回 error |
| 服务退出时在途请求被掐断 | 直接 `return`，没调 `Shutdown` | `srv.Shutdown` 等待在途请求 + 回收 `Serve` 返回值 |

## 练习与参考答案

::: details 第 1 题
给这个服务加一个 `DELETE /api/v1/users/{id}`：删除指定用户，成功返回 204 无响应体，id 非法返回 400，用户不存在返回 404。

**参考答案**（思路）：
```go
// user.go 里 UserRepository 加一行
Delete(ctx context.Context, id int64) error

// sqlite.go 里实现：先 ExecContext("DELETE FROM users WHERE id = ?", id)，
// 用 RowsAffected() 判断 0 时返回 notFound；net/http 里 204 用 w.WriteHeader(http.StatusNoContent)。
```
为什么这样写更好：删除的「不存在」语义和查询「不存在」一样，都归一成 `notFound`，让调用方和 HTTP 层只认 `code`，不认 SQL 细节。
:::

::: details 第 2 题
给 `List` 加上 `limit` 和 `offset` 两个查询参数，并补上分页测试。

**参考答案**（思路）：
```go
// handler 用 r.URL.Query().Get("limit") 解析，缺省 limit=10，非法值返回 badRequest；
// service 校验 limit ∈ [1, 100]，防止一次拉全表；
// repo 的 SQL 改成 "SELECT ... ORDER BY id LIMIT ? OFFSET ?"。
```
为什么这样写更好：校验放在 service 层而不是 SQL 里，既保护了数据库，又让「limit 超范围」以 `invalid_argument` 的语义暴露给客户端，而不是一个数据库层的 500。
:::

::: details 第 3 题
为什么判断 email 重复要用 `errors.As` 提取 `sqlite3.Error` 的错误码，而不是 `strings.Contains(err.Error(), "UNIQUE constraint")`？

**参考答案**：错误字符串是人读的，驱动实现、版本或语言包一变就会失效；`ErrConstraint` 是驱动导出的稳定错误码，`errors.As` 走的是类型判定而不是文本匹配。用字符串判错误只在「没有更好办法」时才用，这里明显有更好的办法。
:::

::: details 第 4 题
把 33.6 的中间件顺序改成 `recover(accessLog(requestID(mux)))`，重跑 `go test ./internal/chapter/go33_app/`，`TestMiddlewarePanicRecovery` 还过吗？再观察访问日志少了什么。

**参考答案**：测试仍然会过——因为 `TestMiddlewarePanicRecovery` 只断言响应是 500/internal，不检查访问日志。但 `demo33_6` 的输出里 `/panic` 的 `msg=access ... status=500` 那一条会消失。这正好说明：**只测响应不测日志，顺序 bug 就漏网了**；给访问日志补一条断言（例如往 logger 的 writer 里检查 `path=/panic`）才能守住这个不变量。
:::

::: details 第 5 题
用 `fakeRepo` 写一个测试：`Create` 传空 name 时，返回的错误经 `errors.As` 后 `Code == CodeInvalidArgument`。

**参考答案**：
```go
func TestCreateEmptyNameCode(t *testing.T) {
	_, err := NewUserService(&fakeRepo{}).Create(context.Background(), "", "a@b.com")
	var ae *AppError
	if !errors.As(err, &ae) || ae.Code != CodeInvalidArgument {
		t.Fatalf("空 name 应返回 invalid_argument，得到 %v", err)
	}
}
```
为什么这样写更好：断言打到 `code` 这一稳定契约上，而不是脆弱的错误字符串；`code` 被客户端程序直接依赖，是最值得测的东西。
:::

## 小结

- 服务端项目拆成「装配 / 业务 / 数据 / 协议 / 运行期」多文件，手动依赖注入 + 消费方定义接口，替换实现即替换数据源。
- 配置按「默认值 < 环境变量 < 校验」三级走，非法值在启动阶段就拒绝。
- `:memory:` SQLite 必须限定单连接，否则每个连接一个库；迁移用 `IF NOT EXISTS` 保持幂等。
- 错误码贯穿三层：handler 解析、service 校验、repo 归一，`writeError` 是失败路径唯一出口，非 `AppError` 一律 500/internal。
- 中间件顺序是「谁包谁」的问题：accessLog 放最外层，recover 在内层，才能既救回 panic 又保留访问日志。
- 优雅关闭 = 超时配置 + `ctx.Done()` 触发的 `Shutdown` + 回收 `Serve` 返回值。
- 测试分三层：`fakeRepo` 测业务、`httptest` 测全链路、`testing.B` 测开销；断言优先打到 `code` 这类稳定契约上。

下一章（第 34 章）把这台服务推向部署：交叉编译、Docker 多阶段构建、健康检查、`pprof`、指标与日志采集，补上「上线」这最后一块拼图。