# 第 1 章 · 环境搭建与第一个 Go 程序

学一门语言最容易卡住的地方不是语法，而是第一步：装完之后不知道 `GOPATH` 和 `go mod` 是什么关系，代码该放在哪个目录，`go run` 和 `go build` 差在哪，报错了也不知道先看哪一行。

这一章不碰复杂语法，只把「Go 程序是怎么被组织、编译和运行的」讲清楚：模块、包、可见性、初始化顺序，以及 `cmd/`、`internal/` 这类目录约定。后面每一章都建立在这些约定之上。

本章配套代码在 `internal/chapter/go01_hello/`，执行 `go run ./cmd/go-learn` 可以看到全部输出；只想验证这一章，用 `go test ./internal/chapter/go01_hello/`。

## 1.1 三条命令确认环境没问题

装上 Go 之后先做一次体检，前两条看版本和路径，第三条看关键环境变量：

```bash
go version
go env GOROOT GOPATH GOMODCACHE GOPROXY GOTOOLCHAIN GOOS GOARCH
```

实测输出（本机 Windows / amd64）：

```text
go version go1.26.3 windows/amd64
D:\program\go
D:\program\gopath
D:\program\gopath\pkg\mod
https://goproxy.cn,direct
auto
windows
amd64
```

这几个变量先记住含义，后面都绕不开：

| 变量 | 含义 | 什么时候会关心 |
| --- | --- | --- |
| `GOROOT` | Go 工具链自己的安装目录 | 极少手动改；多版本共存时才动 |
| `GOPATH` | 工作区，`go install` 的产物默认落到 `GOPATH/bin` | 想把工具装到固定位置时 |
| `GOMODCACHE` | 下载下来的依赖缓存目录 | 清缓存、排查依赖问题时 |
| `GOPROXY` | 依赖下载源，`direct` 表示直连 | 拉不动依赖时（国内一般配代理） |
| `GOTOOLCHAIN` | 工具链选择策略，`auto` 表示按 `go.mod` 自动切换 | 老机器要跑新版本项目时 |
| `GOOS` / `GOARCH` | 目标操作系统与架构 | 交叉编译、写平台相关代码时 |

`GOTOOLCHAIN=auto` 是 Go 1.21 之后的能力：如果项目 `go.mod` 里写的版本比本机新，`go` 命令会自动下载对应工具链，而不是直接报错。这也是本书把 `go 1.26` 写进 `go.mod` 后仍能在旧版本上跑的原因。

## 1.2 go.mod：module 时代的起点

本仓库的 `go.mod` 只有两行：

```text
module go-learn

go 1.26
```

`module go-learn` 这一行决定了所有导入路径的前缀。本仓库里命令行的导入路径就是 `go-learn/cmd/go-learn`，用 `go list` 可以验证：

```bash
go list ./...
```

```text
go-learn/cmd/go-learn
go-learn/internal/chapter/go01_hello
go-learn/test
```

`go list ./...` 里的 `...` 是通配符，表示「当前模块下所有包」，这条命令是排查「我的包到底有没有被认出来」的第一选择。

### 从 GOPATH 到 module

Go 1.11 之前没有 module，所有代码必须放在 `GOPATH/src` 下面，导入路径 = 目录相对 `GOPATH/src` 的路径；项目依赖直接放在同一个工作区里，无法表达版本。module 解决的就是这个问题：**依赖版本写进 `go.mod`，代码可以放在任何目录**。

判断当前是不是在模块里，看这两条命令：

```bash
go list -m
go mod graph
```

本仓库实测（还没有任何第三方依赖）：

```text
go-learn
```

```text
go-learn go@1.26
go@1.26 toolchain@go1.26
```

`go mod graph` 输出的每一行是「A 依赖 B」。现在只有主模块和工具链两条自带关系；等第 12 章引入真实依赖后，这张图会变长，那时再回头看它最有价值。

## 1.3 第一个程序：最小需要什么

一个能跑起来的 Go 程序，骨架只有三段：

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
```

- `package main`：声明这个文件属于哪个包。叫 `main` 的包会被编译成可执行文件，**并且必须**有一个 `main` 函数；叫别的名字就是库包。
- `import "fmt"`：引入标准库的格式化输入输出包。导入但没有用到会直接编译失败（1.10 有真实报错）。
- `func main()`：程序入口，无参数、无返回值。

本章的代码没有写在 `main` 包里，而是写在 `internal/chapter/go01_hello` 这个普通库里，只对外提供一个 `Demo()` 函数；真正调用它的是 `cmd/go-learn` 里的 `main` 包。这样做的好处是每章代码都能被单独测试，而不是全挤在 `main` 里。

第 1 节的实际输出：

```text
--- 1. 最简单的 Go 程序 ---
Hello, World!
Hello, Go!
欢迎来到 Go 的世界，这是第 1 章
```

三行分别来自 `fmt.Println`、`fmt.Print` 和 `fmt.Printf`：

| 函数 | 行为 |
| --- | --- |
| `fmt.Println(a, b)` | 打印后自动换行，多个参数之间加空格 |
| `fmt.Print(a, b)` | 不自动换行，多个参数之间只在两边都不是字符串时加空格 |
| `fmt.Printf(格式, 参数...)` | 按格式串输出，不自动换行 |

格式串里的 `%s` 是字符串、`%d` 是十进制整数、`%T` 打印类型。这三个函数和全部动词表放在第 3 章，这里先认识形状就行。

## 1.4 包、导入与可见性

Go 用**目录**划分包：同一个目录下的所有 `.go` 文件属于同一个包（`_test.go` 可以是同包，也可以是 `包名_test`）。本章的包里有 `demo.go`、`version.go` 和 `demo_test.go` 三个文件，包名都是 `go01_hello`。

包名和导入路径不是一回事：

- 导入路径是「模块路径 + 目录路径」，例如 `go-learn/internal/chapter/go01_hello`；
- 包名是文件里 `package` 后面那个标识符，这里是 `go01_hello`；
- 调用方用包名访问导出成员：`go01_hello.GetGreeting("Go")`。

**可见性只看首字母大小写**，没有 `public` / `private` 关键字：

| 写法 | 可见范围 |
| --- | --- |
| `func GetGreeting(...)` | 导出，包外可以调用 |
| `func internalHelper()` | 未导出，只有本包（含同包测试文件）能调用 |
| `type User struct{ Name string }` | 类型导出，但只有 `Name` 字段被导出 |

同包测试文件能测到未导出函数，这正是 `internalHelper` 只被 `demo_test.go` 使用的原因——它是演示「包内私有」的靶子，不是给外部用的 API。

用 `go doc` 看一个包的公开面貌（实测输出）：

```bash
go doc go-learn/internal/chapter/go01_hello
```

```text
package go01_hello // import "go-learn/internal/chapter/go01_hello"

Package go01_hello 演示环境搭建与第一个 Go 程序的核心概念。

本章覆盖 go 工具链基础、module 初始化、包与 main、初始化顺序、 以及项目目录布局的最佳实践。

运行方式：

    go run ./cmd/go-learn

只想验证这一章：

    go test ./internal/chapter/go01_hello/

const Chapter = 1 ...
func Demo()
func GetGreeting(name string) string
```

注意 `go doc` 只列出**导出**成员：`internalHelper` 不在里面，包注释里写的运行方式却会原样展示。所以包注释要当成文档来写——它会出现在 `go doc` 和 pkg.go.dev 上。

## 1.5 go run、go build、go install

三个命令都编译，区别在产物放哪、留不留：

```bash
go run ./cmd/go-learn        # 编译到临时目录并立即执行，不留产物
go build -o bin/go-learn.exe ./cmd/go-learn   # 编译到指定路径
go install ./cmd/go-learn    # 编译并安装到 GOBIN（默认 GOPATH/bin）
```

`go run` 的「临时目录」不是比喻，用 `-work` 可以让它把工作目录打印出来并保留（实测输出）：

```text
WORK=C:\Users\123\AppData\Local\Temp\go-build463307534
```

所以 `go run` 每次都要重新走到编译这一步（有构建缓存，改动过的包才重编）。它适合跑脚本、验证示例；要交付或部署，用 `go build`。

编译出来的二进制里嵌着模块和构建信息，能直接读出来（实测输出，构建产物 2 493 440 字节）：

```bash
go version -m <二进制路径>
```

```text
C:\Users\123\AppData\Local\Temp\go-learn-demo.exe: go1.26.3
	path	go-learn/cmd/go-learn
	mod	go-learn	(devel)
	build	-buildmode=exe
	build	-compiler=gc
	build	CGO_ENABLED=1
	build	GOARCH=amd64
	build	GOOS=windows
	build	GOAMD64=v1
```

`path` 是被编译的包，`mod` 是主模块（`(devel)` 表示还没打版本标签），`build` 那几行是编译期设置。排查「线上跑的是哪个版本」时，这条命令比重装一遍环境快得多。

| 命令 | 产物位置 | 典型用途 |
| --- | --- | --- |
| `go run` | 临时目录，执行完即弃 | 跑示例、临时脚本 |
| `go build` | 当前目录或 `-o` 指定路径 | 本地调试、交付二进制 |
| `go install` | `GOBIN`（默认 `GOPATH/bin`） | 安装 `golangci-lint` 这类工具 |

## 1.6 初始化顺序：包级变量 → init → main

运行 `go run ./cmd/go-learn`，在本章标题**之前**会先出现三行（实测输出）：

```text
[启动阶段] 步骤 1：包级变量初始化（按依赖关系与声明顺序）
[启动阶段] 步骤 2：demo.go 的 init() 执行（同文件内按声明顺序）
[启动阶段] 步骤 2：version.go 的 init() 执行（文件名排序在 demo.go 之后）
```

这三行就是本章第 5 节要解释的现象。完整的初始化顺序是：

1. **被导入的包先初始化**，而且是递归的：`main` → `go01_hello` → `fmt`、`os`、`runtime`，最底层的包最先完成。
2. **当前包的包级变量**按依赖关系初始化（有依赖的先算，无依赖的按声明顺序）。
3. **当前包的 `init` 函数**执行：多个文件按**文件名排序**，同一文件内按**声明顺序**。
4. 最后才执行 `main`。

第 2 步是本章唯一「看得见」的演示代码：

```go
// packageVar 的初始化表达式是一次函数调用，所以它打印的内容落在 main 之前
var packageVar = initPackageVar()

func initPackageVar() string {
	fmt.Println("[启动阶段] 步骤 1：包级变量初始化（按依赖关系与声明顺序）")
	return "package-level"
}
```

第 3 步在 `demo.go` 和 `version.go` 里各有一个 `init`：

```go
// demo.go
func init() {
	fmt.Println("[启动阶段] 步骤 2：demo.go 的 init() 执行（同文件内按声明顺序）")
}

// version.go
func init() {
	fmt.Println("[启动阶段] 步骤 2：version.go 的 init() 执行（文件名排序在 demo.go 之后）")
}
```

输出顺序验证了「按文件名排序」这条规则：`demo.go` 排在 `version.go` 前面。

关于 `init` 需要注意几点：

- 不能有参数和返回值，也不能被显式调用，只能由运行时自动触发；
- 一个包可以有任意多个 `init`，一个文件里也可以有多个；
- 常见用途是注册驱动（如 `database/sql` 驱动）和初始化包级缓存；
- 反例是把重活放在 `init` 里：它不能返回错误，出事只能 `panic`，而且时机早于 `main`，会让启动逻辑难以测试。第 12 章会讲更可控的替代方案。

## 1.7 变量与零值（第 2 章预告）

第 3 节提前展示了三种变量声明和零值，第 2 章会展开讲，这里先看形状：

```go
var message string = "Go 是静态类型语言" // 完整声明：名字 + 类型 + 值
var count = 42                          // 省略类型，由右侧推导
language := "Go"                        // 短变量声明，只能在函数内用

var num int     // 零值 0
var text string // 零值 ""
var flag bool   // 零值 false
```

实测输出：

```text
--- 3. 变量声明与零值（第 2 章预告）---
Go 是静态类型语言
count 的类型是 int，值是 42
Go 诞生于 2009 年
零值：int=0, string="", bool=false
```

值得先记住的一条：**Go 没有未初始化变量**。声明了但没赋值的变量拿到的是类型的零值，不是随机内存。这条规则让「变量忘了初始化」这类 bug 在 Go 里基本不存在。

## 1.8 运行环境：runtime 与 go env

程序可以在运行期读到自己的版本和平台信息，第 4 节的输出就是这么来的（实测）：

```text
--- 4. go env 与环境信息 ---
Go 版本：go1.26.3
操作系统：windows
架构：amd64
CPU 核心数：28
GOPATH：D:\program\gopath
GOPROXY：https://goproxy.cn,direct
```

前四行来自 `runtime` 包，属于**编译期就确定**的信息：

| 调用 | 来源 | 交叉编译时的值 |
| --- | --- | --- |
| `runtime.Version()` | 编译这个二进制的工具链版本 | 不变 |
| `runtime.GOOS` / `runtime.GOARCH` | 目标平台 | 变成目标平台，不是本机 |
| `runtime.NumCPU()` | 运行期读到的逻辑核心数 | 运行的那台机器 |

最后两行来自 `os.Getenv`，读的是**当前进程的环境变量**。这里有个容易搞混的点：环境变量为空并不代表默认值是空，只代表没人显式设置过。想知道实际生效值，要问工具链自己：

```bash
go env GOPATH
go env GOPROXY
```

所以本章代码里对空值的处理是「提示去看 `go env`」，而不是硬写一个默认值——写在代码里的默认值迟早会跟工具链的真实默认值不一致。

## 1.9 目录布局：cmd、internal、pkg、test

本仓库的布局：

```text
go-learn/
├── cmd/go-learn/           程序入口，只放 main 包
├── internal/chapter/       每章一个包，形如 goNN_主题
├── test/                   跨包的集成测试
├── docs/                   VitePress 文档
├── AGENTS.md / CLAUDE.md   协作规范
└── go.mod
```

常见目录名的约定：

| 目录 | 放什么 | 为什么 |
| --- | --- | --- |
| `cmd/<名字>/` | 只放 `main` 包，`main` 函数尽量薄 | 一个仓库可以有多个可执行程序，互不干扰 |
| `internal/` | 内部实现包 | **编译器强制**：模块外的代码无法导入 `internal` 下的包 |
| `pkg/` | 允许外部导入的公开库 | 本仓库没有发布库的计划，所以先不用它 |
| `test/` | 跨包集成测试 | 与单元测试分开，跑起来代价更大 |

`internal` 是两个里唯一由语言规则保证的：把包放进 `internal`，就等于宣布「这是实现细节，外部别依赖」。`pkg` 没有任何语法含义，纯粹是社区约定——本仓库用不上就不建，**不要为了凑结构建空目录**。

## 1.10 五个真实报错怎么读

下面每条都来自实际运行，行号对应各自的最小示例。

**报错 1：不在模块里运行**

```text
go: go.mod file not found in current directory or any parent directory; see 'go help modules'
```

在没有任何 `go.mod` 的目录里执行 `go run .` 就会看到它。`go` 命令会从当前目录向上找 `go.mod`，一直找到磁盘根目录为止。修法是 `go mod init <模块路径>`，或者回到模块内部执行。

**报错 2：导入但没使用**

```go
import (
	"fmt"
	"os" // 整个文件没用到 os
)
```

```text
# go-learn/.scratch/unused
.scratch\unused\main.go:5:2: "os" imported and not used
```

`imported and not used` 是**编译错误**，不是警告：Go 不给「先留着以后用」的机会。同理，函数里声明了不用的局部变量也会报 `declared and not used`。这和 Go 的一个取舍有关——宁可在编译期拦住，也不留下需要人工清理的噪音（未使用的**包级**变量和函数则是允许的）。

**报错 3：`go vet` 抓出格式化错误**

```go
name := "Go"
fmt.Printf("count = %d\n", name) // %d 收到的是 string
```

```text
.scratch\vetprintf\main.go:7:22: fmt.Printf format %d has arg name of wrong type string
```

这条代码**能编译通过**，但运行结果是：

```text
count = %!d(string=Go)
```

`%!d(string=Go)` 是 `fmt` 在运行期发现类型不匹配时的兜底输出。它不会崩，只会安静地把数据打错——所以 `go vet`（以及第 19 章的 `golangci-lint`）必须进 CI，光靠编译器拦不住这类问题。

**报错 4：依赖没有声明就 import**

```text
.scratch\badmodule\main.go:6:2: no required module provides package github.com/google/uuid; to add it:
	go get github.com/google/uuid
```

Go 不会像某些语言那样「看到 import 就自动下载」。报错里已经给了修复命令；如果网络或代理不通，会看到更底层的一条：

```text
go: github.com/google/uuid: module lookup disabled by GOPROXY=off
```

`GOPROXY=off` 表示禁止任何网络查找，这条消息说明问题出在「拿不到模块」，不是代码写错了。国内环境通常把 `GOPROXY` 设成 `https://goproxy.cn,direct`（1.1 的实测输出就是这个值）。

**报错 5：同一个目录放了两套包声明**

```text
found packages a (a.go) and b (b.go) in C:\...\.scratch\mixedpkgs
```

一个目录只能有一个包名（测试文件除外）。想拆包就拆目录，不能靠文件名区分。

## 1.11 常用命令速查

| 命令 | 作用 | 备注 |
| --- | --- | --- |
| `go version` | 查看工具链版本 | 排查「我装的到底是哪个版本」 |
| `go env <变量>` | 查看生效的环境变量 | 比 `echo $GOPATH` 可靠 |
| `go mod init <路径>` | 初始化模块 | 只在新建项目时执行一次 |
| `go mod tidy` | 增删 `go.mod` / `go.sum` 里的依赖 | 提交前跑一遍 |
| `go mod why <包>` | 解释某个依赖为什么被引入 | 清理依赖时用 |
| `go list ./...` | 列出当前模块的所有包 | 确认包有没有被识别 |
| `go run ./cmd/...` | 编译并运行 | 不留产物 |
| `go build -o <路径> ./cmd/...` | 编译到指定位置 | 交付用 |
| `go install ./cmd/...` | 安装到 `GOBIN` | 装工具用 |
| `gofmt -l .` | 列出没格式化的文件 | 空输出才是通过 |
| `go vet ./...` | 静态检查 | 能抓出 `Printf` 格式错误 |
| `go test ./...` | 跑全部测试 | 第 19 章展开 |
| `go doc <包>` | 看公开 API 与包注释 | 不用离开终端 |
| `go version -m <二进制>` | 读二进制里的模块信息 | 定位线上版本 |
| `go clean -cache` | 清构建缓存 | 排查「改了没生效」时 |

## 1.12 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 在某个目录里 `go run` 报 `go.mod file not found` | 当前目录不在任何模块内 | `go mod init`，或回到模块根目录执行 |
| `imported and not used` / `declared and not used` | Go 把未使用的导入、局部变量当作编译错误，不是警告 | 删掉，或用 `_ = x` 显式丢弃 |
| `go vet` 报 `Printf` 格式错误，但 `go run` 照样跑 | 这类错误到运行期才显形，编译器拦不住 | 按 vet 提示改；把 `go vet` 接进 CI |
| 改了源码，反复 `go run` 输出还是旧的 | 命中构建/测试缓存 | `go clean -cache`（测试则 `go clean -testcache`） |
| 多个文件里 `init` 的执行顺序和文件摆放顺序不一致 | 编译按文件名字母序、再按出现顺序执行 | 别依赖隐式顺序，需要先后就用依赖关系显式控制 |

## 1.13 练习

1. 在本章包下新建 `echo.go`，实现 `func Echo(name string) string`：名字为空时返回 `"Hello, World!"`，否则返回 `"Hello, <名字>!"`；再用表驱动测试覆盖空名字、英文名和中文名三种情况。
2. 在 `go01_hello` 包里再加一个文件 `extra.go`，写一个 `init` 打印任意标记，然后运行 `go test ./internal/chapter/go01_hello/`，观察它与 `demo.go`、`version.go` 里两个 `init` 的先后顺序，说出排序规则。
3. 执行 `go build -o bin/go-learn.exe ./cmd/go-learn`，然后分别用 `go version -m` 和 `go run -work` 验证：编译出来的二进制里有什么信息？`go run` 的工作目录在哪？
4. 故意在某个 `.go` 文件里导入 `"strconv"` 却不用它，运行 `go run`，把报错原文抄下来，并解释 Go 为什么不把「未使用的导入」降级成警告。
5. 把 `go.mod` 里的 `module go-learn` 改成 `module example.com/go-learn`，执行 `go build ./...`，把报错抄下来，想清楚：为什么改模块路径会牵动所有 import？

### 参考答案

::: details 第 1 题

```go
// internal/chapter/go01_hello/echo.go
package go01_hello

import "fmt"

// Echo 返回一句问候；名字为空时退化成默认问候语。
func Echo(name string) string {
	if name == "" {
		return "Hello, World!"
	}
	return fmt.Sprintf("Hello, %s!", name)
}
```

```go
// internal/chapter/go01_hello/echo_test.go
package go01_hello

import "testing"

func TestEcho(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"空名字", "", "Hello, World!"},
		{"英文名", "Go", "Hello, Go!"},
		{"中文名", "世界", "Hello, 世界!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Echo(tt.in); got != tt.want {
				t.Errorf("Echo(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
```

表驱动测试是 Go 里最主流的写法：用例写成数据，循环里用 `t.Run` 生成子测试，失败时能直接从子测试名字定位到是哪条用例。

:::

::: details 第 2 题

新增文件后输出顺序由**文件名字典序**决定，所以 `demo.go` → `extra.go` → `version.go`，同一文件内再按声明顺序。用 `go test -v` 就能看到这些 `init` 打印的标记排在所有测试之前。

规则可以这样记：**文件名排序在前、文件内声明顺序在后**。想让注册逻辑先跑，就给它取一个靠前的文件名，或者放进同一个文件的顶部。

:::

::: details 第 3 题

`go version -m bin/go-learn.exe` 会打印 `go1.26.3`、`path go-learn/cmd/go-learn`、`mod go-learn (devel)` 以及一串 `build` 行（`GOOS`、`GOARCH`、`CGO_ENABLED` 等）。这些信息是编译期写进二进制的。

`go run -work ./cmd/go-learn` 会打印 `WORK=<临时目录>`，说明 `go run` 先把程序编译到临时目录再执行；`-work` 的作用是把目录保留下来，方便查看中间产物。

:::

::: details 第 4 题

```text
# go-learn/...
main.go:5:2: "strconv" imported and not used
```

Go 的设计者把「未使用的导入/局部变量」定为编译错误，理由是：这类残留会让代码读者以为它有用（假信息），而且随着重构反复出现，交给人工清理不如让工具在编译期挡住。它的代价是写代码时偶尔要多删一行——换来的是仓库里不会长期堆积死代码。

:::

::: details 第 5 题

报错形如：

```text
no required module provides package go-learn/internal/chapter/go01_hello; to add it:
	go get go-learn/internal/chapter/go01_hello
```

因为**导入路径的前缀就是模块路径**：`cmd/go-learn/main.go` 里写的是 `import "go-learn/internal/chapter/go01_hello"`。改了 `module` 之后，所有以旧模块路径开头的 import 都不再指向本模块，编译器只能把 `go-learn/...` 当成外部模块去找。

正确做法是改 `module` 的同时，把仓库里所有 import 的前缀一起替换（IDE 的「重命名模块路径」或 `gofmt -r` 都能做），这也是很多项目在迁移仓库地址时一次改一大批文件的原因。

:::

## 1.14 小结

- Go 的环境体检就三条：`go version`、`go env`、`go list ./...`；环境变量为空 ≠ 默认值为空，实际取值要问 `go env`。

- `go.mod` 的 `module` 行决定了所有导入路径的前缀；module 取代 GOPATH 之后，代码可以放在任意目录，依赖版本写进 `go.mod`。

- 可执行程序必须属于 `main` 包且有一个 `main` 函数；其他包是库包。本章的代码放在库里，由 `cmd/go-learn` 统一调用，方便单章测试。

- 可见性只看首字母大小写：大写导出、小写包内可见；包注释和导出成员会出现在 `go doc` 里，要当成文档写。

- `go run` 编译到临时目录立即执行，`go build -o` 产出可交付的二进制，`go install` 装到 `GOBIN`；二进制的模块信息可以用 `go version -m` 读出来。

- 初始化顺序是「导入的包 → 包级变量 → `init` → `main`」，多个 `init` 按文件名排序、文件内按声明顺序；`init` 不能返回错误，所以别把重活放进去。

- Go 没有未初始化变量，声明后拿到的是类型的零值（`0`、`""`、`false`）。

- 目录约定：`cmd/` 放入口，`internal/` 由编译器强制禁止外部导入，`test/` 放集成测试；`pkg/` 是约定而非语法，用不上就不建。

- 报错速记：`go.mod file not found` 是没进模块；`imported and not used` 是编译错误；`%!d(string=...)` 说明格式串和参数类型不匹配，靠 `go vet` 提前发现；`module lookup disabled` 是代理/网络问题，不是代码问题。

下一章讲**变量、常量与基本类型**：把这一章里只用了一次的 `var`、`:=` 和零值规则展开，讲清楚 Go 的类型系统在数值、字符和字符串上到底怎么划界的。
