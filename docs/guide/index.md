# 学习路线与章节规划

这套笔记按「先跑起来、再看原理」的思路组织：每一章有一个明确的主题，配一份可以直接 `go run` 的示例代码，正文里的输出都来自当前仓库的实际运行结果。

代码放在 `internal/chapter/` 下，按 `goNN_主题` 命名。例如第 1 章的包就是 `internal/chapter/go01_hello/`，在 `cmd/go-learn/main.go` 里按顺序调用。每个章节包都带自己的单元测试，可以单独验证。

## 阅读建议

1. 先读章节内容，把例子敲一遍，再看输出和你预期是否一致。
2. 每章末尾有练习，先自己做，再展开答案对照。
3. 卡在编译错误上时先看错误信息：Go 通常会直接给出文件、行号和原因。
4. 想单独跑某一章，用 `go run ./cmd/go-learn` 看全量输出，或 `go test ./internal/chapter/goNN_主题/` 只验证那一章。

## 章节规划

标记「规划中」的章节正文和示例代码尚未落地，其余章节可以直接点进正文阅读。

### 基础篇（第 1–12 章）

| 章节 | 主题 | 你会学到 | 配套代码 |
| --- | --- | --- | --- |
| 第 1 章 | [环境搭建与第一个 Go 程序](./go01-hello) | `go env`、GOPATH 与 module 的区别、`go mod init`、`go run` / `build` / `install` / `fmt` / `vet` / `doc`、`main` 包与包的初始化、项目目录布局、工具链与 IDE 配置 | `internal/chapter/go01_hello/` |
| 第 2 章 | [变量、常量与基本类型](./go02-variables) | `var` 与 `:=`、零值、批量声明、作用域与遮蔽、`const` 与 `iota`、无类型常量、整数族与溢出、浮点精度、复数、`byte` 与 `rune`、显式类型转换、类型定义与类型别名、空白标识符、命名规范 | `internal/chapter/go02_variables/` |
| 第 3 章 | [运算符与格式化输出](./go03-operators-fmt) | 算术 / 比较 / 逻辑 / 位运算与移位、运算符优先级、Go 没有三元运算符、`fmt` 家族、常用动词 `%v` `%+v` `%#v` `%T` `%q`、宽度与精度、`Stringer` 与 `error` 对输出的影响、`go vet` 抓格式化错误 | `internal/chapter/go03_operators_fmt/` |
| 第 4 章 | [控制流](./go04-control-flow) | `if` 与初始化语句、`for` 的四种形态、`range` 遍历各种类型、`break`/`continue` 与标签、`switch` 与无表达式 `switch`、`fallthrough`、`goto`、Go 1.22 循环变量语义变化 | `internal/chapter/go04_control_flow/` |
| 第 5 章 | [函数与闭包](./go05-functions) | 函数签名、多返回值、命名返回值、变参、函数是一等值、闭包与捕获、`defer` 的执行顺序与参数求值时机、`defer` 配合命名返回值的坑、递归、`init` | `internal/chapter/go05_functions/` |
| 第 6 章 | [指针、值与内存入门](./go06-pointers) | `&` 与 `*`、Go 只有值传递、用指针修改调用方数据、`new` 与 `make`、`nil` 指针、指针接收者 vs 值接收者、逃逸分析初窥（`-gcflags=-m`）、为什么没有指针运算 | `internal/chapter/go06_pointers/` |
| 第 7 章 | [数组、切片与映射](./go07-slices-maps) | 数组是值类型、切片的三要素、`append` 与扩容、切片共享底层数组的坑、`copy` 与三下标切片、nil 切片与空切片、内存滞留、map 增删查改与遍历随机序、map key 约束、并发写 map 会 panic | `internal/chapter/go07_slices_maps/` |
| 第 8 章 | [字符串、字节与 Unicode](./go08-strings) | string 是只读字节序列、`len` 数的是字节、UTF-8 与 rune、中文下标切片的坑、`strings` 与 `strconv`、`strings.Builder` 与 `bytes.Buffer`、拼接性能对比、`[]byte` 转换成本、`unicode/utf8` | `internal/chapter/go08_strings/` |
| 第 9 章 | [结构体与方法](./go09-structs-methods) | struct 定义与初始化、匿名字段与嵌入、字段标签、可比较性与相等、方法定义、值 / 指针接收者与方法集、方法值与方法表达式、`NewXxx` 惯例、组合优于继承、内存对齐 | `internal/chapter/go09_structs_methods/` |
| 第 10 章 | [接口与类型系统](./go10-interfaces) | 隐式实现、接口嵌入、`any`、类型断言与带 `ok` 形式、type switch、typed nil 陷阱、`io.Reader`/`io.Writer` 实战、小接口设计、接口值的内存表示、依赖倒置 | `internal/chapter/go10_interfaces/` |
| 第 11 章 | [错误处理](./go11-errors) | `error` 接口、`errors.New`、`fmt.Errorf` 与 `%w`、`errors.Is` / `As` / `Join` / `Unwrap`、自定义错误、哨兵错误、错误与日志的分工、`panic` 与 `recover` 的边界、错误信息书写规范 | `internal/chapter/go11_errors/` |
| 第 12 章 | [包、模块与依赖管理](./go12-modules) | package 与 import、导出规则、`init` 与初始化顺序、`internal` 包、循环依赖的解法、`go.mod` / `go.sum`、语义化版本与最小版本选择、`go mod tidy` / `why`、`replace` 与 `go work`、`vendor`、`GOPROXY` | `internal/chapter/go12_modules/` |

### 进阶篇（第 13–20 章）

| 章节 | 主题 | 你会学到 | 配套代码 |
| --- | --- | --- | --- |
| 第 13 章 | [泛型](./go13-generics) | 类型参数与类型约束、`comparable`、联合类型与 `~`、类型推导、泛型函数与泛型类型、泛型为什么不能有类型参数化的方法、约束中的方法集、`slices` / `maps` / `cmp` 标准库、泛型与接口的取舍 | `internal/chapter/go13_generics/` |
| 第 14 章 | [反射](./go14-reflection) | `reflect.TypeOf` / `ValueOf`、Kind 与 Type、可寻址性与 `Set`、结构体标签解析、动态调用、`reflect.DeepEqual`、反射三定律、性能代价、`encoding/json` 怎么用反射 | `internal/chapter/go14_reflection/` |
| 第 15 章 | [标准库精讲（一）：时间、数学与排序](./go15-stdlib-time-sort) | `time.Time` / `Duration` / `Location`、Layout 格式化、时区与 UTC 陷阱、`Timer` / `Ticker`、单调时钟与 `Since`、`math` 与 `math/rand/v2`、`sort` 与 `slices.SortFunc` / `cmp`、`maps`、`min` / `max` / `clear` | `internal/chapter/go15_stdlib_time_sort/` |
| 第 16 章 | [标准库精讲（二）：正则、文本与模板](./go16-stdlib-text) | `regexp` 预编译与 `MustCompile`、Find / Replace / Split、命名捕获组、贪婪与非贪婪、`text/template` 与 `html/template` 的转义、`unicode` 与 `utf8`、`encoding/csv`、`strings.NewReplacer` | `internal/chapter/go16_stdlib_text/` |
| 第 17 章 | [文件、路径与 IO](./go17-files-io) | `io.Reader` / `io.Writer` 组合、`os.ReadFile` / `WriteFile`、`bufio.Scanner` 与行长限制、`io.Copy` / `TeeReader`、`io/fs` 抽象、`filepath` 跨平台路径、`WalkDir`、临时文件与权限、`defer Close` 的错误、原子写、`//go:embed` | `internal/chapter/go17_files_io/` |
| 第 18 章 | [序列化与配置](./go18-serde-config) | `encoding/json` 标签与 `omitempty`、嵌套与指针字段、`Decoder` 流式解析、自定义 `MarshalJSON`、数字精度与 `json.Number`、`encoding/xml` / `csv` / `gob`、YAML 与 TOML、环境变量、配置分层与优先级 | `internal/chapter/go18_serde_config/` |
| 第 19 章 | [测试、基准与代码质量](./go19-testing) | `testing` 基础、表驱动测试、子测试与 `t.Run`、`t.Parallel`、`t.Helper` / `t.Cleanup` / `t.TempDir`、基准测试与 `-benchmem`、模糊测试（fuzzing）、示例测试、覆盖率、`httptest`、接口打桩、`go vet` / `staticcheck` / `golangci-lint` | `internal/chapter/go19_testing/`、`test/` |
| 第 20 章 | [并发基础](./go20-concurrency) | goroutine、`sync.WaitGroup`、channel（无缓冲 / 有缓冲 / 方向 / 关闭）、`range` 读 channel、`select` 与超时、`sync.Mutex` / `RWMutex` / `Once` / `Map`、`sync/atomic`、数据竞争与 `-race`、channel 与锁的取舍 | `internal/chapter/go20_concurrency/` |

### 并发与运行时篇（第 21–24 章）

| 章节 | 主题 | 你会学到 | 配套代码 |
| --- | --- | --- | --- |
| 第 21 章 | [并发模式与陷阱](./go21-concurrency-patterns) | worker pool、生产者消费者、fan-in / fan-out、pipeline、取消与超时组合、`errgroup`、带并发上限的批量任务、goroutine 泄漏的成因与排查、死锁、channel 关闭原则、并发代码怎么测 | `internal/chapter/go21_concurrency_patterns/` |
| 第 22 章 | [context 与生命周期管理](./go22-context) | `context.Background` / `TODO`、`WithCancel` / `WithTimeout` / `WithDeadline` / `WithValue`、取消传播与取消树、`ctx.Done()` 与 `ctx.Err()`、context 作为第一个参数、不要存进结构体、Value 的类型化 key 与边界、超时链路与资源释放 | `internal/chapter/go22_context/` |
| 第 23 章 | [运行时、调度与内存模型](./go23-runtime) | GMP 调度模型与抢占、goroutine 栈增长、GC 三色标记与写屏障、`GOGC` / `GOMEMLIMIT`、内存分配与逃逸分析、内存模型与 happens-before、`sync/atomic` 的语义、编译链接与构建标签、`runtime` 包与 `GODEBUG` | `internal/chapter/go23_runtime/` |
| 第 24 章 | [性能分析与优化](./go24-performance) | benchmark 的正确写法与结果解读、用 `runtime.MemStats` 度量分配次数、`pprof`（CPU / heap / goroutine / block / mutex）、`go tool trace`、切片与 map 预分配、`strings.Builder`、`string` 与 `[]byte` 零拷贝转换、`sync.Pool`、锁竞争与原子操作、逃逸分析与内联 | `internal/chapter/go24_performance/` |

### 工程与生态篇（第 25–34 章）

| 章节 | 主题 | 你会学到 | 配套代码 |
| --- | --- | --- | --- |
| 第 25 章 | [网络编程基础与 TCP/UDP](./go25-net) | `net` 包、`Dial` / `Listen` / `Accept`、每连接一个 goroutine、粘包与自定义协议（长度前缀 / 换行分隔）、Deadline 超时、UDP、DNS 解析、连接关闭、写一个 echo 服务 | `internal/chapter/go25_net/` |
| 第 26 章 | [HTTP 服务端与 REST API](./go26-http-server) | `net/http`、`Handler` / `HandlerFunc`、`ServeMux` 与 Go 1.22 路由增强（方法 + `{id}` 通配符）、中间件链（请求 ID / 日志 / panic 恢复 / CORS / 限流）、请求解析与 JSON 响应、统一错误体与状态码、`http.Server` 四个超时、`os/signal` 与优雅关闭、`httptest` 与可测试性 | `internal/chapter/go26_http_server/` |
| 第 27 章 | [HTTP 客户端与外部服务](./go27-http-client) | `http.Client` 的三层超时（`Timeout` / context / `Transport` 阶段超时）、`Transport` 连接池与复用（`httptrace` 验证）、请求构造（JSON / 表单 / 请求头）、响应处理（状态码、gzip 自动解压、`resp.Body` 关闭与连接复用）、重试与指数退避、`Retry-After`、multipart 上传与流式下载、`httptest` 测外部依赖、TLS 与重定向策略 | `internal/chapter/go27_http_client/` |
| 第 28 章 | [数据库编程](./go28-database) | `database/sql` 与驱动、`sql.DB` 连接池参数、`Query` / `QueryRow` / `Exec`、预处理语句与 SQL 注入、NULL 与 `sql.NullXxx`、事务与回滚、context 超时、`sql.ErrNoRows`、SQLite 实战、仓储层测试 | `internal/chapter/go28_database/` |
| 第 29 章 | [命令行工具、日志与配置](./go29-cli-logging) | `os.Args` 与 `flag`、子命令与 cobra、退出码、stdout / stderr 分流、`log/slog` 结构化日志与 Handler / Level、请求 ID 与日志上下文、日志落盘与轮转、配置优先级与环境变量、敏感信息脱敏 | `internal/chapter/go29_cli_logging/` |
| 第 30 章 | [常用第三方生态与选型](./go30-ecosystem) | Web 框架（gin / echo / fiber / 标准库）、ORM 与 SQL 构建器（GORM / ent / sqlc / sqlx）、依赖注入（wire / fx / 手动注入）、参数校验（validator）、测试库（testify / gomock）、日志库、配置库、CLI 库、gRPC / WebSocket / 消息队列概览、选型原则与避免过度依赖 | `internal/chapter/go30_ecosystem/` |
| 第 31 章 | [unsafe、cgo 与代码生成](./go31-unsafe-cgo) | `unsafe.Sizeof` / `Offsetof` / `Alignof`、`unsafe.Pointer` 与 `uintptr` 的规则、`runtime.KeepAlive`、cgo 类型映射与内存生命周期、`C.CString` 配 `C.free`、`//export` 反向导出、cgo 的代价与构建约束、`CGO_ENABLED=0`、`//go:generate`、`//go:embed` | `internal/chapter/go31_unsafe_cgo/`、`examples/cgo/` |
| 第 32 章 | [工程实践与项目结构](./go32-engineering) | 项目布局（`cmd` / `internal` / `pkg`）、分层与依赖方向、依赖注入与可测试性、Functional Options 等 Go 惯用法、配置与环境区分、错误与日志规范、`go generate`、Makefile、lint 规则、版本注入（`-ldflags`）、依赖治理与安全扫描 | `internal/chapter/go32_engineering/` |
| 第 33 章 | [综合实战：Go 服务端项目](./go33-app) | 分层架构（handler → service → repository）、手动依赖注入、配置加载、SQLite 迁移、REST 接口与统一错误码、中间件（请求 ID / 访问日志 / panic 恢复）、优雅关闭、单元与集成测试、基准测试、多阶段 Docker 打包 | `internal/chapter/go33_app/` |
| 第 34 章 | 部署、可观测性与运维（规划中） | 交叉编译与 `CGO_ENABLED=0`、多阶段 Docker 构建与镜像瘦身、`-ldflags` 注入版本、容器探针与健康检查、在线 `pprof`、指标（expvar / Prometheus）、结构化日志采集、优雅重启、压测与容量评估、发布清单与回滚 | `internal/chapter/go34_ops/` |

### 附录

正文之外的速查页，随章节推进逐步补齐：

| 附录 | 内容 | 文件 |
| --- | --- | --- |
| 附录 A | [Go 版本特性对照](../appendix/versions) | 1.18 泛型、1.21 `slog`、1.22 路由与循环变量、1.26 新增 API | `docs/appendix/versions.md` |
| 附录 B | [常见编译错误与运行时报错速查](../appendix/errors) | 编译器、`go vet`、panic、`-race`、测试失败与错误包装 | `docs/appendix/errors.md` |
| 附录 C | [标准库常用包速查](../appendix/stdlib) | 按 CLI、HTTP、IO、并发、测试、性能和文本处理场景索引 | `docs/appendix/stdlib.md` |

## 本地跑文档站

```bash
pnpm install
pnpm docs:dev     # 打开 http://localhost:5173/go-learn/
pnpm docs:build   # 构建静态站点到 docs/.vitepress/dist
```
