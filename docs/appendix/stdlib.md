# 附录 C：标准库常用包速查

按任务查包，不按字母死记。下面列出本课程实际使用频率较高的 API；完整签名和边界以 `go doc 包名` 或 pkg.go.dev 为准。

| 场景 | 首选包 | 常用 API | 注意事项 |
| --- | --- | --- | --- |
| 格式化输出 | `fmt` | `Println`、`Printf`、`Sprintf`、`Fprintf`、`Errorf` | `%w` 只用于错误包装 |
| 参数与环境 | `os`、`flag` | `Args`、`LookupEnv`、`OpenFile`、`FlagSet` | 不要在深层函数 `os.Exit` |
| 结构化日志 | `log/slog` | `New`、`NewTextHandler`、`NewJSONHandler`、`LevelVar` | 字段键名保持稳定，敏感值先脱敏 |
| 错误 | `errors` | `New`、`Is`、`As`、`Join`、`Unwrap` | 用 `%w` 保留错误链 |
| 字符串与数字 | `strings`、`strconv` | `Builder`、`Cut`、`Split`、`Atoi`、`FormatInt` | `len(string)` 是字节数 |
| Unicode | `unicode`、`unicode/utf8` | `RuneCountInString`、`ValidString`、`DecodeRuneInString` | 字符索引和字节索引不同 |
| 切片与 map | `slices`、`maps` | `Sort`、`SortFunc`、`Contains`、`Clone`、`Equal` | map 遍历顺序不稳定 |
| 时间与随机数 | `time`、`math/rand/v2` | `Now`、`Since`、`NewTimer`、`NewTicker`、`New` | 测试注入时钟，避免依赖 Sleep 抢时序 |
| JSON / XML / CSV | `encoding/json`、`encoding/xml`、`encoding/csv` | `Encoder`、`Decoder`、`Marshal`、`Unmarshal` | 检查 Decode/Encode 错误，限制输入大小 |
| 文件与路径 | `os`、`io`、`bufio`、`path/filepath` | `ReadFile`、`WriteFile`、`Copy`、`Scanner`、`WalkDir` | 跨平台路径使用 filepath |
| 临时资源 | `os` | `MkdirTemp`、`CreateTemp` | 用 `defer` 清理，注意权限 |
| 测试 HTTP | `net/http/httptest` | `NewServer`、`NewRecorder` | 测试后关闭 server 和 Body |
| HTTP 服务端 | `net/http` | `Handler`、`ServeMux`、`Server` | 设置超时，区分优雅关闭和强制关闭 |
| HTTP 客户端 | `net/http` | `Client`、`NewRequestWithContext`、`Transport` | 始终关闭响应体，控制重定向与超时 |
| 网络连接 | `net`、`net/netip` | `Listen`、`Dial`、`Resolver`、`ParseAddr` | 明确 deadline 和连接关闭责任 |
| 上下文 | `context` | `WithCancel`、`WithTimeout`、`WithValue` | context 作为第一个参数，不存入结构体 |
| 并发同步 | `sync`、`sync/atomic` | `Mutex`、`WaitGroup`、`Once`、`Map`、`Int64` | 先定义所有权，再选择锁或 channel |
| 并发任务 | `golang.org/x/sync/errgroup` | `WithContext`、`Go`、`Wait` | 本仓库已有依赖，需同步 go.mod/go.sum |
| 反射 | `reflect` | `TypeOf`、`ValueOf`、`StructOf`、`VisibleFields` | 先确认 Kind、CanSet 和 nil |
| 性能分析 | `runtime`、`runtime/pprof`、`runtime/trace` | `ReadMemStats`、`StartCPUProfile`、`Lookup` | 先测量再优化，报告版本与参数 |
| 测试与基准 | `testing` | `Run`、`Helper`、`Cleanup`、`TempDir`、`B.Loop` | 测试隔离全局状态，基准避免无效优化 |
| 加密与摘要 | `crypto/sha256`、`crypto/tls` | `Sum256`、`New`、`Config` | 摘要不是密码存储；TLS 校验不要关闭 |
| 压缩与归档 | `compress/gzip`、`archive/zip` | `NewReader`、`NewWriter`、`OpenReader` | 限制解压大小，关闭 Reader/Writer |
| 模板 | `text/template`、`html/template` | `Parse`、`Execute` | HTML 场景必须用 `html/template` |
| 正则 | `regexp` | `Compile`、`MustCompile`、`FindStringSubmatch` | 用户输入不要直接 `MustCompile` |

## 选包的四条原则

1. 先看标准库是否已经提供稳定抽象。
2. 选择能表达资源所有权和错误边界的 API。
3. 对外部输入设置大小、时间和并发上限。
4. 记录包的版本要求，并用最小示例验证实际行为。

命令行工具、HTTP 服务和后台任务经常同时使用 `os`、`context`、`log/slog`、`errors` 与 `io`。优先把这些基础接口传入构造函数，业务层就能在不启动真实网络或文件系统的情况下测试。
