# go-learn

按章节由浅入深学 Go 的笔记仓库：每一章配一份 Markdown 正文和一个可以直接 `go run` 的示例包，正文里的结论都来自实际编译与运行结果。

## 目录结构

```text
cmd/go-learn/       统一入口，按章节顺序调用各章的 Demo
internal/chapter/   每章一个包，形如 goNN_主题
test/               跨包的集成测试
examples/           独立的小工程示例（模块、cgo 等）
docs/guide/         每章一份正文，形如 goNN-主题.md
docs/.vitepress/    文档站配置（VitePress）
.github/workflows/  CI 与 GitHub Pages 部署工作流
AGENTS.md           协作规范：新增章节清单、正文模板、代码与注释风格、验证要求
CLAUDE.md           同一套规范的另一个入口，内容与 AGENTS.md 保持一致
```

## 环境要求

- Go：1.24 及以上（仓库在 go1.26.3 上验证，示例会用到 `range` over int、`min`/`max`、`slices` 等较新的标准库能力）
- Node.js 20+ 与 pnpm：只在构建文档站时需要

## 常用命令

```bash
# Go 示例代码
go run ./cmd/go-learn        # 依次运行第 1 章到最新一章的演示
go build ./...               # 只编译
go test ./...                # 跑单元测试与集成测试
go doc ./internal/chapter/go05_functions   # 查看某一章的包文档

# VitePress 文档站
pnpm install         # 安装依赖
pnpm docs:dev        # 本地预览 http://localhost:5173/go-learn/
pnpm docs:build      # 构建静态站点到 docs/.vitepress/dist
pnpm docs:preview    # 预览构建结果
```

## 章节进度

完整路线与每章知识点见[学习路线与章节规划](docs/guide/index.md)。正文写在 `docs/guide/`，配套代码写在 `internal/chapter/`，两者章节号一一对应。

下表随章节推进更新，「规划中」表示正文和代码尚未落地。

| 章节 | 主题 | 正文 | 配套代码 |
| --- | --- | --- | --- |
| 第 1 章 | 环境搭建与第一个 Go 程序 | [go01-hello](docs/guide/go01-hello.md) | `internal/chapter/go01_hello/` |
| 第 2 章 | 变量、常量与基本类型 | [go02-variables](docs/guide/go02-variables.md) | `internal/chapter/go02_variables/` |
| 第 3 章 | 运算符与格式化输出 | [正文](docs/guide/go03-operators-fmt.md) | `internal/chapter/go03_operators_fmt/` |
| 第 4 章 | 控制流 | [go04-control-flow](docs/guide/go04-control-flow.md) | `internal/chapter/go04_control_flow/` |
| 第 5 章 | 函数与闭包 | [正文](docs/guide/go05-functions.md) | `internal/chapter/go05_functions/` |
| 第 6 章 | 指针、值与内存入门 | [正文](docs/guide/go06-pointers.md) | `internal/chapter/go06_pointers/` |
| 第 7 章 | 数组、切片与映射 | [go07-slices-maps](docs/guide/go07-slices-maps.md) | `internal/chapter/go07_slices_maps/` |
| 第 8 章 | 字符串、字节与 Unicode | [go08-strings](docs/guide/go08-strings.md) | `internal/chapter/go08_strings/` |
| 第 9 章 | 结构体与方法 | [go09-structs-methods](docs/guide/go09-structs-methods.md) | `internal/chapter/go09_structs_methods/` |
| 第 10 章 | 接口与类型系统 | [正文](docs/guide/go10-interfaces.md) | `internal/chapter/go10_interfaces/` |
| 第 11 章 | 错误处理 | [go11-errors](docs/guide/go11-errors.md) | `internal/chapter/go11_errors/` |
| 第 12 章 | 包、模块与依赖管理 | [go12-modules](docs/guide/go12-modules.md) | `internal/chapter/go12_modules/` |
| 第 13 章 | 泛型 | [go13-generics](docs/guide/go13-generics.md) | `internal/chapter/go13_generics/` |
| 第 14 章 | 反射 | [go14-reflection](docs/guide/go14-reflection.md) | `internal/chapter/go14_reflection/` |
| 第 15 章 | 标准库精讲（一）：时间、数学与排序 | [正文](docs/guide/go15-stdlib-time-sort.md) | `internal/chapter/go15_stdlib_time_sort/` |
| 第 16 章 | 标准库精讲（二）：正则、文本与模板 | [go16-stdlib-text](docs/guide/go16-stdlib-text.md) | `internal/chapter/go16_stdlib_text/` |
| 第 17 章 | 文件、路径与 IO | [正文](docs/guide/go17-files-io.md) | `internal/chapter/go17_files_io/` |
| 第 18 章 | 序列化与配置 | [go18-serde-config](docs/guide/go18-serde-config.md) | `internal/chapter/go18_serde_config/` |
| 第 19 章 | 测试、基准与代码质量 | [go19-testing](docs/guide/go19-testing.md) | `internal/chapter/go19_testing/` |
| 第 20 章 | 并发基础 | [go20-concurrency](docs/guide/go20-concurrency.md) | `internal/chapter/go20_concurrency/` |
| 第 21 章 | 并发模式与陷阱 | [go21-concurrency-patterns](docs/guide/go21-concurrency-patterns.md) | `internal/chapter/go21_concurrency_patterns/` |
| 第 22 章 | context 与生命周期管理 | [go22-context](docs/guide/go22-context.md) | `internal/chapter/go22_context/` |
| 第 23 章 | 运行时、调度与内存模型 | [go23-runtime](docs/guide/go23-runtime.md) | `internal/chapter/go23_runtime/` |
| 第 24 章 | 性能分析与优化 | [go24-performance](docs/guide/go24-performance.md) | `internal/chapter/go24_performance/` |
| 第 25 章 | 网络编程基础与 TCP/UDP | [go25-net](docs/guide/go25-net.md) | `internal/chapter/go25_net/` |
| 第 26 章 | HTTP 服务端与 REST API | [go26-http-server](docs/guide/go26-http-server.md) | `internal/chapter/go26_http_server/` |
| 第 27 章 | HTTP 客户端与外部服务 | [go27-http-client](docs/guide/go27-http-client.md) | `internal/chapter/go27_http_client/` |
| 第 28 章 | 数据库编程 | [go28-database](docs/guide/go28-database.md) | `internal/chapter/go28_database/` |
| 第 29 章 | 命令行工具、日志与配置 | [go29-cli-logging](docs/guide/go29-cli-logging.md) | `internal/chapter/go29_cli_logging/` |
| 第 30 章 | 常用第三方生态与选型 | [go30-ecosystem](docs/guide/go30-ecosystem.md) | `internal/chapter/go30_ecosystem/` |
| 第 31 章 | unsafe、cgo 与代码生成 | [go31-unsafe-cgo](docs/guide/go31-unsafe-cgo.md) | `internal/chapter/go31_unsafe_cgo/` |
| 第 32 章 | 工程实践与项目结构 | [go32-engineering](docs/guide/go32-engineering.md) | `internal/chapter/go32_engineering/` |
| 第 33 章 | 综合实战：Go 服务端项目 | [go33-app](docs/guide/go33-app.md) | `internal/chapter/go33_app/` |
| 第 34 章 | 部署、可观测性与运维 | 规划中 | `internal/chapter/go34_ops/` |

## 文档站部署

## 附录速查

- [Go 版本特性对照](docs/appendix/versions.md)
- [常见编译错误与运行时报错](docs/appendix/errors.md)
- [标准库常用包速查](docs/appendix/stdlib.md)

推送到 `main` 分支后，GitHub Actions 会自动跑 Go 检查并构建 VitePress 发布到 GitHub Pages：

- CI：`.github/workflows/ci.yml`——`gofmt`、`go vet`、`go build`、`go test -race`、`go run ./cmd/go-learn`，外加文档站的 `pnpm docs:build`
- 文档部署：`.github/workflows/deploy.yml`——构建 VitePress 并发布到 GitHub Pages

首次部署前需要手动开启一次 Pages：仓库 **Settings → Pages → Build and deployment → Source** 选 **GitHub Actions**。这一步必须手动做——工作流里的 `GITHUB_TOKEN` 没有创建 Pages 站点的权限，`configure-pages` 的自动启用会报 `Resource not accessible by integration`。开启之后再跑一次工作流即可。

CI 里的 pnpm 固定为 8.15.9，和仓库里 `pnpm-lock.yaml` 的 `lockfileVersion 6.0` 对齐。升级 pnpm 大版本时要先重新生成锁文件，否则 `pnpm install --frozen-lockfile` 会失败。

仓库名如果不是 `go-learn`，需要同步修改 `docs/.vitepress/config.mts` 里的 `base`（用户主页仓库要写成 `base: '/'`）。
