# 附录 A：Go 版本特性对照

本表按语言版本整理常用变化，便于回看章节代码需要的最低版本。`go.mod` 中的 `go 1.26.0` 是本仓库声明的语言版本；实际 `go version` 是工具链版本，两者可能不同。版本特性以 Go 官方 release notes 为准，平台和工具链仍需在项目 CI 中验证。

| 版本 | 代表性语言与标准库变化 | 本仓库对应章节 |
| --- | --- | --- |
| Go 1.18 | 泛型类型参数、类型约束；`any` 与 `comparable`；模糊测试进入 `testing`；`go.work` 工作区 | 第 12、13、19 章 |
| Go 1.19 | `GOMEMLIMIT` 软内存上限；`atomic` 增加类型化原子值；泛型实现和编译器优化 | 第 20、23、24 章 |
| Go 1.20 | `errors.Join`；切片转数组指针；`context` 取消原因 API；`math/rand/v2` 尚未加入标准库 | 第 11、22 章 |
| Go 1.21 | `log/slog`；`slices`、`maps`、`cmp`；`min` / `max`；`context.WithCancelCause`；PGO 正式可用 | 第 15、22、23、29 章 |
| Go 1.22 | `for range` 每轮迭代变量语义修正；`range` 直接遍历整数；`net/http.ServeMux` 方法与通配符路由；`go` 工具链自动切换 | 第 4、26 章 |
| Go 1.23 | `iter` 迭代器约定；`range` over function；`unique` 实验包；定时器 channel 语义调整；`go env -changed` | 第 15、23 章 |
| Go 1.24 | 泛型类型别名（需显式实验开关的早期阶段）；`weak` 指针；`testing.B.Loop`；`go tool` 管理工具依赖；标准库性能改进 | 第 19、23、24 章 |
| Go 1.25 | `WaitGroup.Go`；`testing/synctest`；泛型接口类型集合继续放宽；容器感知的 GOMAXPROCS 默认值改进 | 第 20、21、23 章 |
| Go 1.26 | `new(T, value)` 初始化写法；泛型接口可直接作为值类型使用的限制继续放宽；`reflect` 类型和值迭代器；`testing` artifact 目录；`slog.NewMultiHandler` | 第 14、19、29 章 |

## 版本判断的三个层次

### `go.mod` 的 `go` 行

它声明模块所依赖的语言语义和最低工具链要求。例如：

```go
module example.com/app

go 1.26.0
```

不要只根据本机编译成功就提高该版本。先确认 CI、开发机和发布镜像都有对应工具链，再提交修改。

### `toolchain` 行

需要固定某个补丁版本时可以额外写：

```go
toolchain go1.26.3
```

它影响 Go 命令选择的工具链，不等同于把代码语言版本改成更高版本。团队通常把最低语言版本写在 `go` 行，把推荐构建版本写在 CI 或工具链管理文件中。

### 构建标签与平台

版本特性不等于所有平台都支持。涉及 `cgo`、系统调用、race detector 或 WebAssembly 时，应同时检查 `GOOS`、`GOARCH`、`CGO_ENABLED` 和构建标签。跨版本兼容代码可以用文件后缀或 `//go:build go1.26` 拆分，但不要用运行时反射替代清晰的编译期边界。

## 升级检查清单

1. 阅读目标版本 release notes 和 `go doc` 变更。
2. 修改 `go.mod` 后执行 `go mod tidy`，检查依赖是否意外升级。
3. 执行 `gofmt -w .`、`go vet ./...`、`go test ./...` 和 `go test -race ./...`。
4. 检查基准测试、序列化输出、时间与定时器行为等容易受版本影响的结果。
5. 在 CI 中至少保留最低支持版本和最新稳定版本两个矩阵。

版本号解决“代码能否编译”的问题，兼容性测试解决“行为是否仍符合契约”的问题；两者不能互相替代。
