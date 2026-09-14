# 第 29 章：命令行工具、日志与配置

前几章已经把网络、HTTP 客户端和数据库串成了服务的主要数据通路，但一个程序要真正交给别人运行，还需要三个工程入口：命令行负责接收意图，配置负责描述运行环境，日志负责留下可检索的事实。本章把它们放在同一个可测试的边界里，重点不是罗列 API，而是建立稳定契约。本章配套代码在 `internal/chapter/go29_cli_logging/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 29.1 `os.Args`、`flag` 与可测试边界

结论：不要让参数解析、业务逻辑和 `os.Exit` 混在一起。真正的入口可以读取 `os.Args[1:]`，核心函数则接收 `[]string`、stdout、stderr 并返回退出码。

```go
func main() {
    os.Exit(go29_cli_logging.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

本章的核心签名是：

```go
func Run(args []string, stdout, stderr io.Writer) int
```

`flag.NewFlagSet` 比包级 `flag.CommandLine` 更适合库代码：每个子命令拥有独立参数集，`ContinueOnError` 会返回错误而不是结束进程，`SetOutput(stderr)` 则让帮助和解析错误走诊断流。

```go
flags := flag.NewFlagSet("user", flag.ContinueOnError)
flags.SetOutput(stderr)
name := flags.String("name", "访客", "显示名称")
verbose := flags.Bool("verbose", false, "输出诊断信息")
if err := flags.Parse(args); err != nil {
    return 2
}
```

实测输出：

```text
stdout: 用户: id=7 name=小林
stderr: 正在查询 user_id=7
exit code: 0
```

边界与坑：标准 `flag` 默认要求选项出现在第一个位置参数之前，因此 `user -name 小林 7` 能解析，而 `user 7 -name 小林` 会把后面的内容当位置参数。若 CLI 有多层命令、补全和复杂帮助，再评估 Cobra；简单工具直接使用标准库，依赖更少、行为更透明。

## 29.2 子命令、stdout / stderr 与退出码

结论：正常机器可消费的结果写 stdout，提示、警告和错误写 stderr；退出码 `0` 表示成功，`1` 表示命令执行失败，`2` 表示用法或参数错误。

```go
switch args[0] {
case "version":
    fmt.Fprintln(stdout, "go-learn-cli v1.0.0")
    return 0
case "user":
    return runUser(args[1:], stdout, stderr)
default:
    fmt.Fprintf(stderr, "未知子命令 %q\n", args[0])
    return 2
}
```

实测输出：

```text
args=["version"] code=0 stdout="go-learn-cli v1.0.0" stderr=""
args=["unknown"] code=2 stdout="" stderr="未知子命令 \"unknown\"\n用法: go-learn-cli <version|user> [选项]"
args=["user" "404"] code=1 stdout="" stderr="查询用户 404 失败: user not found"
```

分流让 shell 管道保持可靠：`tool user 7 > result.txt` 不会把进度信息混入结果。不要在深层函数调用 `os.Exit`，因为它不会运行 `defer`，也会直接杀死测试进程。只在最外层 `main` 调用一次。

Cobra 适合拥有大量嵌套子命令、shell 补全和统一帮助模板的产品型 CLI，但它不是正确退出码和输出分流的替代品。无论使用哪个框架，都保留可注入的输入输出边界。

### 设计一套可预测的命令语法

命令语法一旦被脚本调用，就等同于公开 API。设计时至少区分三类输入：子命令表示动作，位置参数表示动作的主要对象，选项表示可选修饰。例如 `user --verbose 7` 中，`user` 是动作，`7` 是对象，`--verbose` 只改变输出细节。

布尔选项应直接表达开关，不要设计成 `--verbose=true` 才能使用；需要互斥模式时，应使用一个有明确取值集合的选项，而不是多个可能同时为真的布尔值。选项名一旦发布就尽量保持兼容，重命名时可以保留旧名字并输出弃用警告。

帮助信息也是接口的一部分。至少应包含：一句用途说明、完整 Usage、子命令列表、选项默认值和一两个典型示例。错误路径打印简短原因和相关 Usage 即可，不要让用户在数百行全局帮助中寻找一个拼错的参数。

对自动化友好的 CLI 还应考虑以下约定：

- `--help` 和 `--version` 不依赖网络或配置文件，且快速返回
- 非交互环境下不突然等待终端输入
- 机器输出格式由显式选项选择，例如 `--output=json`
- 排序稳定，避免相同输入每次产生不同顺序
- 破坏性动作提供 `--dry-run`，真正执行前明确目标

本章没有把 `Run` 直接接入统一学习程序的 `os.Args`，因为 `cmd/go-learn` 的职责是顺序展示所有章节。实际项目通常另建 `cmd/mytool/main.go`，让每个可执行文件只负责一个入口。

## 29.3 `slog` 结构化日志与 Handler

结论：日志字段应当保留类型和稳定键名，不要先用 `fmt.Sprintf` 拼成一整句话。`slog.Logger` 负责记录，`Handler` 决定文本或 JSON 格式、最低级别和字段改写。

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
logger := slog.New(handler)
logger.Info("订单已创建", "order_id", 42, "amount", 99.5)
```

实测输出为了可复现，通过 `ReplaceAttr` 移除了时间字段：

```json
{"level":"INFO","msg":"订单已创建","order_id":42,"amount":99.5}
```

开发终端通常用 `TextHandler` 便于阅读，生产采集通常用 `JSONHandler`。字段名要稳定，例如统一使用 `request_id`，不要一处叫 `requestId`、另一处叫 `traceID`。键值参数必须成对；更严格的代码可使用 `slog.String`、`slog.Int` 和 `LogAttrs`，减少误传。

日志与错误也有不同职责：底层函数包装并返回错误，知道请求或任务结果的边界记录一次。层层记录同一个错误会制造重复日志。

### 建立字段契约

团队可以维护一份很短的字段约定，让查询和告警不依赖个人习惯：

| 字段 | 类型 | 含义 |
| --- | --- | --- |
| `request_id` | string | 单次请求的关联标识 |
| `operation` | string | 稳定的业务动作名 |
| `duration_ms` | number | 操作耗时，单位固定为毫秒 |
| `status` | number | HTTP 或业务状态码 |
| `error` | string | 已脱敏的最终错误信息 |

高基数字段需要谨慎。用户 ID、订单 ID 适合日志检索，但不适合直接做指标标签，否则监控系统会生成海量时间序列。日志与指标可以描述同一事件，却有不同的数据模型和成本边界。

调用 `logger.With("component", "billing")` 可以创建带公共字段的派生 logger。派生值应按组件长期复用，避免每条日志重复拼装相同字段。不要随意调用 `slog.SetDefault` 修改进程全局状态，尤其是在库包中；应用入口可以设置默认 logger，库更适合显式接收 logger。

记录耗时时使用单调时钟支持的 `time.Since(start)`，输出时选择固定单位。不要把人类可读字符串和数值字段混用，否则 `duration="12ms"` 无法像 `duration_ms=12` 一样直接做数值范围查询。

## 29.4 日志级别与请求上下文

结论：级别用于控制信息量，context 用于携带请求范围的关联标识，但不要把 logger 必然塞进每个 context。本章从 context 取出请求 ID，再派生 logger。

```go
ctx := WithRequestID(context.Background(), "req-29")
LoggerFromContext(ctx, logger).Info("请求完成", "status", 200)
```

当 Handler 最低级别为 `INFO` 时，`DEBUG` 被过滤。实测输出是：

```text
level=INFO msg=请求完成 request_id=req-29 status=200
```

常见级别约定：`DEBUG` 是排障细节，`INFO` 是正常生命周期事件，`WARN` 是可恢复异常，`ERROR` 是当前操作失败。使用 `slog.LevelVar` 可以在进程运行期间调整最低级别。

context 的键应使用包内自定义类型，避免与其他包冲突；值只放请求范围、跨调用链有意义的小数据。配置、数据库连接和可选参数不应塞进 context。

## 29.5 日志落盘与轮转

结论：`slog` 只要求一个 `io.Writer`，因此落盘很简单；但轮转涉及并发、重命名、保留策略和外部采集器，生产环境应明确契约。

```go
w, err := NewRotatingWriter("app.log", 10<<20)
if err != nil {
    return err
}
defer w.Close()
logger := slog.New(slog.NewJSONHandler(w, nil))
```

本章实现单进程、按大小、只保留一个 `.1` 备份的最小写入器。实测输出：

```text
超过 12 字节后: 当前=app.log 备份=app.log.1
```

写入器用互斥锁保护“检查大小、轮转、写入”的复合操作。单条记录超过限制时仍完整写入新文件，避免拆断 JSON。生产程序还要决定：保留多少份、是否压缩、磁盘写满怎么办、多个进程能否写同一文件，以及日志采集器如何识别重命名。

在容器里通常优先写 stdout/stderr，由运行平台采集和轮转；在传统主机部署里才更常直接写文件。应用内轮转与系统 `logrotate` 不要同时无协调地管理同一个文件。

## 29.6 配置优先级与环境变量

结论：先明确唯一优先级，再逐层覆盖并在启动阶段校验。本章采用“默认值 < 配置文件 < 环境变量 < 命令行”。

```go
cfg, err := LoadConfig(
    map[string]string{"host": "file.local", "port": "7000"},
    map[string]string{"port": "8000", "log_level": "warn"},
    map[string]string{"port": "9000"},
)
```

实测输出：

```text
host=file.local port=9000 level=WARN
非法配置: port "70000" 必须是 1..65535 的整数
```

真实程序读取环境变量可以用 `os.LookupEnv`。它能区分“变量不存在”和“变量存在但为空”；`os.Getenv` 对两者都返回空字符串。建议把 `APP_PORT` 等外部名字在适配层映射成内部字段，不要让环境变量散落在业务代码中。

一个简单的环境变量适配层可以这样写：

```go
func readEnvironment() map[string]string {
    values := map[string]string{}
    for envName, field := range map[string]string{
        "APP_HOST":      "host",
        "APP_PORT":      "port",
        "APP_LOG_LEVEL": "log_level",
        "APP_API_KEY":   "api_key",
    } {
        if value, ok := os.LookupEnv(envName); ok {
            values[field] = value
        }
    }
    return values
}
```

配置文件解析、环境读取和命令行解析只负责产生候选值，统一的 `LoadConfig` 负责覆盖与校验。这样同一条端口规则不会在三个入口各写一遍，也能在测试中用 map 精确构造来源组合。

对切片、map 等复合配置，要先约定覆盖还是合并。标量通常直接覆盖；列表若做合并，需要说明顺序、去重和清空方式。没有明确需求时，整项覆盖比隐式合并更容易理解。

配置输出用于排障时，只打印最终生效的非敏感字段和来源。例如可以报告 `port=9000 source=flag`，但不能把整个配置结构 `%+v` 输出，因为未来新增密码字段后，旧日志语句会在无意间泄露它。

配置失败应当让程序在启动时明确退出，不能悄悄回落到危险默认值。配置对象解析完成后尽量按只读值传递；运行中动态变化的少数字段，例如日志级别，可以使用专门的并发安全组件。

## 29.7 敏感信息脱敏

结论：密码、令牌、Cookie、Authorization、私钥和完整连接串默认不进入日志。确需关联排障时，只记录不可逆摘要或极少量尾号。

```go
logger.Info("连接外部服务",
    "api_key", RedactSecret(cfg.APIKey),
    "timeout", 2*time.Second,
)
```

实测输出：

```text
api_key=****5678
level=INFO msg="连接外部服务" api_key=****5678 timeout=2s
```

脱敏必须尽量靠近日志边界，否则调用方很容易漏掉。字段名也可能泄密，例如 URL 查询参数中含 `token`；不能只盯着值。结构化日志的优势之一是 Handler 可以集中改写敏感键，但应用仍应避免把整个请求头或配置对象直接记录下来。

## 3 个真实报错怎么读

### 报错一：参数类型错误

运行本章解析逻辑时，非法 ID 的真实诊断是：

```text
无效用户 ID "zero"：必须是正整数
```

线索在输入值 `zero` 和约束“正整数”。修复调用参数，而不是在业务层把非法值当成 `0`。

### 报错二：端口越界

```text
port "70000" 必须是 1..65535 的整数
```

错误同时保留字段名、原值和允许范围，能直接定位到配置源。修复配置中的端口；不要截断或取模。

### 报错三：`flag` 未定义选项

最小程序执行 `user -missing 1` 时，标准库返回：

```text
flag provided but not defined: -missing
Usage of user:
  -name string
        显示名称 (default "访客")
  -verbose
        输出诊断信息
```

第一行说明错误原因，后续 Usage 列出合法选项。`Run` 将其归为用法错误并返回退出码 `2`。

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 单元测试一运行就退出 | 深层函数调用了 `os.Exit` | 核心函数返回退出码，只在 `main` 退出 |
| 管道输出混入提示文字 | 所有内容都写 stdout | 结果写 stdout，诊断写 stderr |
| `flag` 忽略后面的选项 | 选项放在首个位置参数之后 | 标准库 `flag` 中先写选项，再写位置参数 |
| 日志无法按订单筛选 | 把字段拼进了 message | 使用稳定的结构化键值字段 |
| 一次错误出现多条日志 | 每层都记录后再返回 | 底层包装错误，在处理边界记录一次 |
| 配置在不同机器行为不同 | 优先级不明确或非法值静默回退 | 固定覆盖顺序，启动时集中校验 |
| JSON 日志偶尔解析失败 | 多 goroutine 自行拼接或拆分记录 | 共享并发安全 Handler，单次提交整条记录 |
| 密钥只在部分日志中脱敏 | 依赖每个调用方自觉 | 在统一日志边界过滤敏感键 |

## 练习与参考答案

### 第 1 题

给 `Run` 增加 `health` 子命令，成功时向 stdout 写 `ok` 并返回 `0`。

::: details 第 1 题参考答案

```go
case "health":
    fmt.Fprintln(stdout, "ok")
    return 0
```

同时增加表驱动测试，断言 stderr 为空。这样写更好，因为健康检查的输出可以被脚本稳定消费。

:::

### 第 2 题

为 `user` 增加 `-json` 选项，开启后输出一行 JSON。

::: details 第 2 题参考答案

```go
jsonOutput := flags.Bool("json", false, "输出 JSON")
if *jsonOutput {
    _ = json.NewEncoder(stdout).Encode(map[string]any{"id": id, "name": *name})
    return 0
}
```

真实代码应检查 `Encode` 错误。使用 Encoder 比手工拼 JSON 更好，因为它能正确转义名字中的引号和换行。

:::

### 第 3 题

用 `slog.LevelVar` 把最低级别从 `INFO` 动态改为 `DEBUG`。

::: details 第 3 题参考答案

```go
level := new(slog.LevelVar)
handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
logger := slog.New(handler)
level.Set(slog.LevelDebug)
logger.Debug("可见了")
```

复用同一个 `LevelVar` 更好，因为不必替换所有持有旧 logger 的组件。

:::

### 第 4 题

改写配置示例，使用 `os.LookupEnv("APP_PORT")` 覆盖端口。

::: details 第 4 题参考答案

```go
env := map[string]string{}
if value, ok := os.LookupEnv("APP_PORT"); ok {
    env["port"] = value
}
cfg, err := LoadConfig(nil, env, nil)
```

`LookupEnv` 比 `Getenv` 更好，因为它能区分未设置与显式空值，后者应当进入校验并报错。

:::

### 第 5 题

扩展脱敏函数：电子邮箱只保留首字符和域名。

::: details 第 5 题参考答案

```go
func RedactEmail(value string) string {
    name, domain, ok := strings.Cut(value, "@")
    if !ok || name == "" || domain == "" {
        return "****"
    }
    return name[:1] + "***@" + domain
}
```

先校验结构再切片更好，可以避免空用户名导致越界；生产代码还需考虑 Unicode 首字符。

:::

### 第 6 题

为轮转写入器补一个“关闭后写入”的测试。

::: details 第 6 题参考答案

```go
w, _ := NewRotatingWriter(filepath.Join(t.TempDir(), "app.log"), 10)
_ = w.Close()
if _, err := w.Write([]byte("x")); !errors.Is(err, os.ErrClosed) {
    t.Fatalf("期望 os.ErrClosed，实际 %v", err)
}
```

断言标准错误值比比较错误字符串更好，因为错误文案变化不会破坏测试。

:::

## 小结

- 把命令核心写成接收参数和 Writer、返回退出码的普通函数
- stdout 承载结果，stderr 承载诊断，退出码表达成功或失败类别
- 简单子命令可以组合多个 `flag.FlagSet`，复杂产品型 CLI 再评估 Cobra
- `slog` 用稳定键值记录事实，由 Handler 决定格式、级别和字段改写
- 请求 ID 等关联字段可以从 context 提取后派生 logger
- 文件轮转必须明确并发、大小、备份和部署环境契约
- 配置按固定层级覆盖，并在启动阶段完成类型转换和范围校验
- 敏感数据默认不记录，必要时只保留最少的脱敏信息

下一章将从标准库走向常用第三方生态，讨论 Web 框架、ORM、依赖注入、测试与 CLI 库该如何选型。
