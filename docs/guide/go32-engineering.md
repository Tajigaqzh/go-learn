# 第 32 章 工程实践与项目结构

前面三十一章讲的是「怎么写对一段 Go」。但从「能跑的脚本」到「能长期维护、能安全迭代的工程」，中间还隔着一层组织能力：目录怎么摆、代码怎么分层、依赖怎么注入、配置和日志怎么规范、构建与检查怎么自动化。这一章就补上这块短板。

本章不引入新的语法，所有结论都围绕一个迷你「用户服务」展开：`handler → service → store` 三层，依赖只朝内（接口）流动。你可以跟着代码把每一层替换掉，感受「可测试」和「难测试」的分界线到底在哪。

**本章配套代码**在 `internal/chapter/go32_engineering/`，执行 `go run ./cmd/go-learn` 可以看到全部输出；单独验证这一章用 `go test ./internal/chapter/go32_engineering/`。

---

## 32.1 项目布局：cmd / internal / pkg

### 结论

Go 官方没有强制目录结构，但社区沉淀出一套「职责分层」的布局。核心只有一条：**可执行入口和可复用代码分开，可复用代码再用 `internal/` 划出私有边界**。

```text
go-learn/
├─ cmd/          # 可执行程序入口，每个子目录一个 main
├─ internal/     # 私有代码，只能被本模块内部 import
├─ pkg/          # 可被外部项目 import 的公共库
├─ test/         # 跨包集成测试
├─ docs/         # 文档
└─ go.mod        # 模块定义
```

### 三个目录各自的职责

| 目录 | 谁放这里 | 谁不能动 | 边界由谁保证 |
| --- | --- | --- | --- |
| `cmd/` | `main` 包，只做参数解析、依赖装配、启动/退出 | 不放业务逻辑 | 约定 |
| `internal/` | 本份代码库私有的包 | 外部模块 | **编译器强制** |
| `pkg/` | 真正要公开给别人 import 的库 | —— | 约定 + 谨慎 |

`internal/` 是三者里最硬的一道边界：从模块外 import `go-learn/internal/...` 会直接编译报错，报错见 [32.10 报错与反例](#3210-报错与反例) 第二例。

### 实测输出

```text
--- 32.1 项目布局：cmd / internal / pkg ---
一个可长期维护的 Go 工程，目录按职责划分：

  go-learn/
  ├─ cmd/          # 可执行程序入口，每个子目录一个 main
  ├─ internal/     # 私有代码，只能被本模块内部 import
  ├─ pkg/          # 可被外部项目 import 的公共库
  ├─ test/         # 跨包集成测试
  ├─ docs/         # 文档
  └─ go.mod        # 模块定义

要点：
  - cmd/ 只放薄薄的 main，真正的逻辑下沉到 internal/
  - internal/ 是编译器强制「私有」的边界，外部模块 import 会报错
  - pkg/ 只放确实要公开的库，不要一上来就建 pkg/
```

### 边界与坑

- **不要一上来就建 `pkg/`**：多数项目的代码根本不打算被别人 import，过早公开意味着要长期承诺 API 稳定性。
- `cmd/` 里可以有多个子目录（`cmd/server`、`cmd/migrate`），它们共享 `internal/` 里的逻辑，这是单一二进制拆分多命令的标准做法。
- 本仓库本身就是这个布局的样本：入口在 `cmd/go-learn`，章节代码全部收在 `internal/chapter`，跨包集成测试单独放在 `test/`。

---

## 32.2 分层与依赖方向：接口朝内依赖

### 结论

把一条请求的处理拆成三层，**依赖方向只能朝一个方向指**：

```text
handler ──> service ──> UserStore(接口)
                            ↑ 实现
                       memoryStore
```

关键不是「分了几层」，而是两层之间用什么类型衔接：上层依赖**接口**，具体实现从下层掉头注入进来——这就是依赖倒置。接口定义在使用方（service）这一侧，而不是实现方（memoryStore）那一侧。

### 最小可运行示例

```go
// UserStore 是数据访问接口，由 service 层依赖；实现方可能是内存、SQL 或测试替身。
type UserStore interface {
	Get(id int) (User, error)
	List() []User
}

// memoryStore 是 UserStore 的内存实现。
type memoryStore struct {
	users map[int]User
}

func (s *memoryStore) Get(id int) (User, error) {
	u, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

// UserService 只依赖 UserStore 接口，不关心具体实现。
type UserService struct {
	store UserStore
}

func (svc *UserService) GetByID(id int) (User, error) {
	if id <= 0 {
		return User{}, fmt.Errorf("invalid id %d", id)
	}
	u, err := svc.store.Get(id)
	if err != nil {
		return User{}, fmt.Errorf("service: get user %d: %w", id, err)
	}
	return u, nil
}
```

### 实测输出

```text
--- 32.2 分层与依赖方向：接口朝内依赖 ---
依赖方向（只能朝一个方向引用）：
  handler ──> service ──> UserStore(接口)
                               ↑ 实现
                          memoryStore

关键点：service 只 import 接口，memoryStore 是被注入进来的实现。
接口由「使用方」定义，而不是由实现方定义——依赖倒置。

GetByID(1) = {ID:1 Name:张三}, err = <nil>
GetByID(99) = {ID:0 Name:}, err = service: get user 99: user not found
  -> 错误里带着 service 与 store 两层上下文，便于定位
```

### 边界与坑

- **依赖方向错了性价比极低**：如果 `service` 直接 `import` 一个具体的 `memoryStore` 包，后面想换 SQL、想打桩测试都得改 service。
- 接口别设计得太大：`UserStore` 只暴露 service 真正需要的方法（`Get`、`List`），这叫接口最小化。
- 错误要一层层往上包：`store` 返回 `ErrNotFound`，`service` 用 `%w` 补上 `service: get user 99:`，定位问题时能一眼看出是哪层。

---

## 32.3 手动依赖注入：换一个实现就能换一套行为

### 结论

Go 生态最主流的依赖注入方式是**构造器注入**：`NewXxx(dep)` 把依赖当参数传进去，而不是在包内用全局变量或单例。好处是一句话能说清——**要换行为，换注入的东西即可，业务代码一行不改**。

### 最小可运行示例

```go
// 同一套 service 逻辑，注入真实实现：
realSvc := NewUserService(newMemoryStore())

// 换注入一个测试替身，行为立刻不同：
fake := &fakeStore{users: map[int]User{2: {ID: 2, Name: "测试替身"}}}
fakeSvc := NewUserService(fake)
```

`fakeStore` 还能记录调用次数，用来验证「service 到底调了几次 store」，这是可测试性的关键一步：

```go
type fakeStore struct {
	users    map[int]User
	getCalls int
}

func (f *fakeStore) Get(id int) (User, error) {
	f.getCalls++
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return User{}, ErrNotFound
}
```

### 实测输出

```text
--- 32.3 手动依赖注入：换一个实现就能换一套行为 ---
注入 memoryStore：      GetByID(2) = 李四
注入 fakeStore：        GetByID(2) = 测试替身
fakeStore.Get 被调用 2 次（可验证交互）
结论：构造器注入（NewXxx(dep)）比包级全局变量/单例更可控、更好测。
```

### 边界与坑

- **反模式：包级 `var db *sql.DB`**。它是隐式全局状态，测试要换实现得动用「全局变量替换」这种脆弱手段，还可能被并行测试互相污染。
- 依赖一多，`NewServer(a, b, c, d, e)` 会变成「参数地狱」，此时有两种出路：把关联依赖收进一个结构体一起注入，或对可选配置改用下一节的 Functional Options。
- 需要框架级 DI（wire、fx）通常要等到依赖图足够大再说；中小项目手动注入反而最透明。

---

## 32.4 Functional Options：优雅的可选配置

### 结论

当一个类型的构造函数有**一堆可选配置**时，用「选项函数」代替多个 `bool`/`int` 参数：默认值集中在一处，新增选项不破坏已有调用方，选项可自由组合。

### 最小可运行示例

```go
type Server struct {
	addr     string
	timeout  time.Duration
	maxConns int
	logger   *slog.Logger
}

type Option func(*Server)

func WithTimeout(d time.Duration) Option {
	return func(s *Server) { s.timeout = d }
}
func WithMaxConns(n int) Option {
	return func(s *Server) { s.maxConns = n }
}

// NewServer 先给出默认配置，再逐个应用 Option。
func NewServer(addr string, opts ...Option) *Server {
	s := &Server{addr: addr, timeout: 3 * time.Second, maxConns: 100, logger: slog.Default()}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
```

### 实测输出

```text
--- 32.4 Functional Options：优雅的可选配置 ---
默认服务：        addr=:8080 timeout=3s maxConns=100
定制服务：        addr=:9090 timeout=500ms maxConns=10
优点：新增选项不破坏已有调用；默认值集中在一处；选项可自由组合。
对比：固定结构体字面量在字段变多后难以区分「未设置」和「设为零值」。
```

### 边界与坑

- **零值陷阱**：用普通结构体字面量 `cfg := &Server{maxConns: 0}` 时，你无法区分「用户显式设成 0」和「忘了设」。Functional Options 把「默认值」和「覆盖」分开，解决了这个歧义。
- 别把**必填**参数也做成 Option——`addr` 这种必须给的仍然走普通参数，Option 只承接「可选」。
- Option 函数里不要让某个 option panic；保持纯赋值，校验可以放在 `NewServer` 末尾统一做。

---

## 32.5 配置与环境：默认值 < 环境变量 < 显式校验

### 结论

配置加载遵循一条清晰优先级：**默认值 < 环境变量**（更上层还有配置文件、命令行参数，本章从简）。同时，环境变量永远只是字符串，解析与范围校验必须集中在一处、**尽早失败**。

### 最小可运行示例

为了可测试，把「读环境变量」抽象成一个 `lookup` 函数注入，生产环境传 `os.LookupEnv`，测试里传一个 map：

```go
// LoadConfig 用 os.LookupEnv 加载配置，等价于 LoadConfigFrom(os.LookupEnv)。
func LoadConfig() (*Config, error) {
	return LoadConfigFrom(os.LookupEnv)
}

// LoadConfigFrom 按「默认值 < 环境变量」的优先级构造配置；lookup 注入是为了可测试。
func LoadConfigFrom(lookup func(string) (string, bool)) (*Config, error) {
	cfg := &Config{Addr: "0.0.0.0", Port: 8080, LogLevel: "info"}
	if v, ok := lookup("APP_ADDR"); ok && v != "" {
		cfg.Addr = v
	}
	if v, ok := lookup("APP_PORT"); ok && v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("parse APP_PORT %q: %w", v, err)
		}
		if p < 1 || p > 65535 {
			return nil, fmt.Errorf("APP_PORT %d out of range [1, 65535]", p)
		}
		cfg.Port = p
	}
	// ...
	return cfg, nil
}
```

### 实测输出

```text
--- 32.5 配置与环境：默认值 < 环境变量 < 显式校验 ---
无环境变量：      &{Addr:0.0.0.0 Port:8080 LogLevel:info}
覆盖后：          &{Addr:127.0.0.1 Port:9090 LogLevel:debug}
非法 APP_PORT：   parse APP_PORT "not-a-number": strconv.Atoi: parsing "not-a-number": invalid syntax
原则：环境变量只做「字符串覆盖」，解析与范围校验集中在 LoadConfig，尽早失败。
```

### 边界与坑

- **越早失败越好**：端口配错了应该启动时就报错退出，而不是等到某条请求命中了才崩。
- 校验要做**范围**检查，不只是能解析：`APP_PORT=99999` 能被 `Atoi` 解析，但它不是合法端口，仍要拒掉。
- 把 `lookup` 注入进来后，测试可以完全不碰真实环境变量，也就不会被并行测试的 `t.Setenv` 相互干扰。

---

## 32.6 错误规范：哨兵错误、%w 包装、errors.Is

### 结论

错误处理要回答两个问题：**调用方怎么稳定地识别一类错误**（哨兵错误 + `errors.Is`），以及**错误从哪层抛出来**（`%w` 逐层包上下文）。

### 最小可运行示例

```go
var (
	ErrNotFound = errors.New("user not found")
	ErrConflict = errors.New("user already exists")
)

// 在 service 层把 store 的错误再包一层，errors.Is 仍能穿透到 ErrNotFound。
u, err := svc.store.Get(id)
if err != nil {
	return User{}, fmt.Errorf("service: get user %d: %w", id, err)
}
```

### 实测输出

```text
--- 32.6 错误规范：哨兵错误、%w 包装、errors.Is ---
完整错误： service: get user 99: user not found
errors.Is(err, ErrNotFound) = true
errors.Is(err, ErrConflict) = false

规范：
  - 用哨兵错误给调用方一个稳定的判断锚点（errors.Is）
  - 逐层用 %w 补上下文，不吞错、不把错误变成字符串
  - 库函数只 return err，不打日志；最上层（main/handler）统一记录
  反例：log.Fatalf 会 os.Exit(1)，库代码里调用会杀死宿主进程且无法 defer
```

### 边界与坑

- `%w` 和 `%v` 一字之差，效果天壤之别：`%v` 只是把错误内容拼进字符串，`errors.Is` 从此**穿透不了**，调用方就丢了判断锚点。
- **除非有意吞掉恢复，否则不要写 `_, _ = ...` 忽略错误**——至少有注释说明为什么。
- 日志和错误的边界：库函数和中间层只 `return err`，由最外层统一记日志；否则同一条错误会沿调用链被重复打印 N 遍。

---

## 32.7 结构化日志：log/slog

### 结论

日志要能被程序解析、按字段检索，而不是靠人眼扫拼出来的字符串。Go 1.21 起的 `log/slog` 用**键值对**承载上下文，并用 `With` 派生子 logger 把固定字段挂在上下文里。

### 最小可运行示例

演示里为了输出稳定，用 `ReplaceAttr` 去掉了每次都不一样的时间戳：

```go
func newDemoLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // 去掉时间戳，让演示输出每次一致
			}
			return a
		},
	}))
}

logger.Info("server starting", "addr", "0.0.0.0:8080")
userLogger := logger.With("service", "user")
userLogger.Debug("store connected", "impl", "memory")
userLogger.Error("get user failed", "user_id", 42, "err", ErrNotFound)
```

### 实测输出

```text
--- 32.7 结构化日志：log/slog ---
level=INFO msg="server starting" addr=0.0.0.0:8080
level=DEBUG msg="store connected" service=user impl=memory
level=ERROR msg="get user failed" service=user user_id=42 err="user not found"
level=WARN msg="slow query" service=user duration_ms=350
```

### 边界与坑

- 默认的 text handler 会输出时间戳（`time=2026-09-15T...`），版本和时区每次运行都变，所以演示代码里显式去掉它；生产环境默认保留即可。
- **错误作为 value 传给 slog**（`"err", ErrNotFound`），别把整段错误 `fmt.Errorf` 拼成一句再当纯文本 `msg` 打出来。
- 固定字段（服务名、模块名）用 `With` 挂一次，别在每个调用点手写重复。

---

## 32.8 版本注入（-ldflags）与构建信息

### 结论

一个二进制的「我是谁」至少要回答两件事：**发布版本号**和**对应哪次提交**。前者用 `-ldflags -X` 注入，后者靠 Go 工具链自动嵌入的 VCS 指纹，二者互补。

### 最小可运行示例

```go
package main

import "fmt"

// version 默认是 "dev"，真正发布时用 -ldflags -X 注入版本号。
var version = "dev"

func main() {
	fmt.Println("version =", version)
}
```

```bash
go build -ldflags "-X main.version=v1.2.3" -o app main.go
```

读构建信息用 `runtime/debug.ReadBuildInfo()`，或者更直接地对已有二进制用 `go version -m`。

### 实测输出

`go run` 下（不落地二进制，所以没有 git 指纹）：

```text
--- 32.8 版本注入（-ldflags）与构建信息 ---
Go 版本：  go1.27.0
模块路径： go-learn
模块版本： (devel)
关键构建设置：
  CGO_ENABLED = 1
  GOARCH = amd64
  GOOS = windows
```

`-X` 注入前后：

```text
$ go run main.go                         # version = dev
$ go build -ldflags "-X main.version=v1.2.3" -o app main.go && ./app
version = v1.2.3
```

`go build` 落地二进制后，`go version -m app` 会多出 VCS 指纹：

```text
mod	go-learn	v0.0.0-20260914100730-a9dc111610e7+dirty
build	vcs=git
build	vcs.revision=a9dc111610e75f83fa9fb06419915451386ec6a6
build	vcs.time=2026-09-14T10:07:30Z
build	vcs.modified=true
```

### 边界与坑

- `vcs.revision`（指向 HEAD 提交）在 `go build` 时都会嵌入；如果工作区有未提交改动，会额外标记 `vcs.modified=true`，且版本号尾部带 `+dirty`。上面抓取时本章代码尚未提交，所以是 `+dirty`——提交后 clean 状态下会变成 `false` 并去掉 `+dirty`。
- `go run` 不写磁盘二进制，所以版本是 `(devel)`、且没有 vcs 设置；要拿到完整版本信息，必须 `go build` / `go install`。
- `-X` 只对包级**变量**生效（字符串或整数），常量改不了；符号名要带完整包路径（`main.version`、`go-learn/internal/...`）。

---

## 32.9 Makefile、静态检查与依赖安全

### 结论

人记不住一堆命令，就把它收进 `Makefile` 或脚本。检查按「格式 → 易错点 → 更严规则 → 依赖漏洞」递进，逐级兜底。

### 命令清单

```makefile
fmt:      gofmt -l -w .
vet:      go vet ./...
test:     go test -race ./...
build:    go build -ldflags "-s -w" -o bin/app ./cmd/go-learn
vuln:     govulncheck ./...
```

每个 target 都等价于一条 `go` 命令，不依赖 `make` 是否安装。

### 实测输出

```text
--- 32.9 Makefile、静态检查与依赖安全 ---
把常用命令收进 Makefile，降低「记命令」成本：

  fmt:      gofmt -l -w .
  vet:      go vet ./...
  test:     go test -race ./...
  build:    go build -ldflags "-s -w" -o bin/app ./cmd/go-learn
  vuln:     govulncheck ./...

等价于直接执行上面的 go 命令，不依赖 make 是否安装。
静态检查分级：gofmt（格式） -> go vet（易错点） -> staticcheck/golangci-lint（更严）
依赖安全：govulncheck 基于漏洞库扫描依赖；go list -m -u 查看可升级版本。
注：本机未安装 make/staticcheck/golangci-lint/govulncheck，命令以 go 原生命令为准。
```

### 边界与坑

- **分级含义**：`gofmt` 保证格式统一（CI 里用 `gofmt -l` 查漏），`go vet` 抓常见的错误用法，`staticcheck`/`golangci-lint` 抓更深的代码质量与风格，`govulncheck` 才管「依赖有没有已知漏洞」。
- `-ldflags "-s -w"`（去掉符号表和 DWARF）能让发布二进制明显变小，但会损失崩溃时的可读堆栈，通常只在最终发布层级用。
- 本机本次未安装 `make`/`staticcheck`/`golangci-lint`/`govulncheck`，所以上面只承诺 `go` 原生命令的结果；这些二进制的具体输出留待你在装好工具后用 `go version`/各自 `--version` 自行核对。

---

## 32.10 报错与反例

### 报错 1：import cycle not allowed

分层方向一旦画反，最典型的报错是循环依赖：

```text
package cycle.example/x/a
	imports cycle.example/x/b from a.go
	imports cycle.example/x/a from b.go: import cycle not allowed
```

**原因**：`a` import `b`，`b` 又 import `a`，形成环。

**修复线索**：看最后 `import cycle not allowed` 前面的两条 `imports`，把其中一边依赖的「具体实现」下沉成接口，或者把被两头共用的东西抽到第三个包。

### 报错 2：use of internal package not allowed

从模块外 import 本仓库的 `internal` 包：

```text
package example.com/other
	main.go:3:8: use of internal package go-learn/internal/chapter/go32_engineering not allowed
```

**原因**：`internal/` 是编译器强制私有边界，只有 `go-learn/...` 路径下的包才能 import。

**修复线索**：如果确实要共享，把包从 `internal/` 挪到 `pkg/`；多数情况下只是改成 import 未被 `internal` 包裹的公开入口。

### 报错 3：log.Fatalf 跳过 defer

```text
$ go run main.go
2026/09/15 10:31:44 加载配置失败: file does not exist
exit status 1
```

注意：代码里 `defer fmt.Println("cleanup: 关闭数据库连接")` 那行**根本没打印**。

**原因**：`log.Fatalf` 内部调用 `os.Exit(1)`，进程立刻退出，跳过所有 `defer`。

**修复线索**：库代码/中间层一律 `return err`，只有 `main` 在拿到致命错误后做必要的显式清理、再调用一个带清理的 `run()` + 统一退出。

### 反例：%v 让 errors.Is 失效

```go
_, err := svc.GetByID(99)
wrapped := fmt.Errorf("wrap: %v", err) // 用了 %v
fmt.Println(errors.Is(wrapped, ErrNotFound)) // false
```

对比 `%w` 版本：

```go
wrapped := fmt.Errorf("wrap: %w", err)
fmt.Println(errors.Is(wrapped, ErrNotFound)) // true
```

**原因**：`%v` 只是把错误的字面内容拼进新字符串，`Unwrap` 链断了；`%w` 才会把原始错误挂进新错误的 `Unwrap` 链里。这是错误处理里最容易踩、又最隐蔽的坑。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `import cycle not allowed` | 两个包互相 import | 依赖倒置：下层实现上提，上层只依赖接口 |
| `use of internal package ... not allowed` | 从模块外 import `internal` 包 | 要公开就挪 `pkg/`，否则改 import 公开入口 |
| service 换实现要改一堆代码 | service 直接依赖具体 store 包 | 依赖 `UserStore` 接口，构造器注入实现 |
| 测试里全局单例互相污染 | 包级 `var` 保存状态 | 构造器注入，测试构造独立实例 |
| 分不清「忘设」和「设了零值」 | 结构体字面量零值歧义 | Functional Options 统一默认值 |
| 端口配错等到请求时才崩 | 只解析不校验、晚失败 | `LoadConfig` 里解析 + 范围校验，启动即报错 |
| `errors.Is` 判断不到底层错误 | 中间用了 `%v` 而非 `%w` | 逐层 `fmt.Errorf("...: %w", err)` |
| 同一条错误打印 N 遍 | 每层都 `log` 一次 | 库函数只 `return err`，最外层统一记日志 |
| `defer` 清理没执行 | 某处调用了 `log.Fatal`/`os.Exit` | 中间层只返回错误，退出统一放 `main` |
| 发布二进制带不了 git 指纹 | 用 `go run` / 环境不含 git | 用 `go build`，`go version -m` 查 VCS 指纹 |
| 时间戳让日志测试不稳定 | 默认 text handler 带时间 | `ReplaceAttr` 去掉时间，或注入固定时钟 |

---

## 练习

### 第 1 题

模仿正文的 `Server`，给 `NewServer` 增加一个 `WithReadTimeout` 选项，并说明为什么用选项而不是给构造函数再加一个 `time.Duration` 参数。

::: details 第 1 题参考答案

```go
func WithReadTimeout(d time.Duration) Option {
	return func(s *Server) { s.readTimeout = d }
}
```

`Server` 里新增字段 `readTimeout time.Duration`，`NewServer` 默认值里给一个合理值（如 `10 * time.Second`），其余不变。

**为什么这样写更好**：

- 已有调用方（`NewServer(":8080")`、`NewServer(":9090", WithTimeout(...))`）一行不用改，向后兼容。
- 如果把可选配置做成普通参数，`NewServer(addr, timeout, maxConns, readTimeout)` 会越滚越长，调用方还要被迫为不关心的项填空值——正是 Functional Options 要消灭的问题。

:::

### 第 2 题

给 `UserService` 实现一个 `Create(u User) error` 方法，要求：`id <= 0` 返回参数错误；`id` 已存在时返回可被 `errors.Is(err, ErrConflict)` 命中的错误。用注入 `fakeStore` 的方式写出测试。

::: details 第 2 题参考答案

```go
func (svc *UserService) Create(u User) error {
	if u.ID <= 0 {
		return fmt.Errorf("invalid id %d", u.ID)
	}
	if _, err := svc.store.Get(u.ID); err == nil {
		return fmt.Errorf("service: create user %d: %w", u.ID, ErrConflict)
	}
	return svc.store.Create(u) // 需要 UserStore 接口补一个 Create
}
```

测试：

```go
func TestUserServiceCreateConflict(t *testing.T) {
	store := &fakeStore{}  // fakeStore 增加 Create 实现
	svc := NewUserService(store)
	// 先 Put 一条 ID=1，再 Create 同 ID
	if err := svc.Create(User{ID: 1, Name: "旧"}); err != nil { t.Fatal(err) }
	if err := svc.Create(User{ID: 1, Name: "新"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("期望 ErrConflict，得到 %v", err)
	}
}
```

**为什么这样写更好**：

- 冲突用 `%w` 包 `ErrConflict`，调用方 `errors.Is` 就能稳定识别，而不是去比对错误字符串。
- `Create` 只依赖 `UserStore` 接口，测试里用 `fakeStore` 完全脱离存储，不碰 IO、跑得快、可控。

:::

### 第 3 题

给 `Config` 增加一个 `Timeout time.Duration` 字段，从环境变量 `APP_TIMEOUT` 解析（默认 `3s`），非法值返回错误。列出 `LoadConfigFrom` 需要改动的地方。

::: details 第 3 题参考答案

```go
type Config struct {
	Addr     string
	Port     int
	LogLevel string
	Timeout  time.Duration // 新增
}

// LoadConfigFrom 里新增分支：
cfg := &Config{Addr: "0.0.0.0", Port: 8080, LogLevel: "info", Timeout: 3 * time.Second}
if v, ok := lookup("APP_TIMEOUT"); ok && v != "" {
	d, err := time.ParseDuration(v)
	if err != nil {
		return nil, fmt.Errorf("parse APP_TIMEOUT %q: %w", v, err)
	}
	if d <= 0 {
		return nil, fmt.Errorf("APP_TIMEOUT %s must be positive", v)
	}
	cfg.Timeout = d
}
```

边界情况：`APP_TIMEOUT=abc` 报 `time: invalid duration "abc"`；`APP_TIMEOUT=0s` 或负数走 `d <= 0` 拒绝。

**为什么这样写更好**：

- 解析用 `time.ParseDuration` 而非手写数字提取，直接复用标准库、支持 `500ms`/`1m30s` 等写法。
- 校验「必须为正」单独兜一层，避免 `0` 超时导致下游行为异常。

:::

### 第 4 题

解释为什么「库代码里不要 `log.Fatal` / `os.Exit`」，并给出一段可测试的替代写法。

::: details 第 4 题参考答案

**为什么不行**：

1. `log.Fatal` 调 `os.Exit(1)`，进程立刻退出，所有 `defer`（关连接、刷日志、清理临时文件）都不执行。
2. 进程直接死掉，调用方没法拿到 `error` 去重试、降级或打点。
3. 单元测试里一旦走到这条路径，整个测试进程被杀，其他测试全跟着挂。

**替代写法**：把「会失败的工作」放进一个返回 `error` 的 `run()` 函数，`main` 只负责调用并统一退出：

```go
func run() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err // 只返回，不退出
	}
	s, err := buildServer(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Serve()
}

func main() {
	if err := run(); err != nil {
		log.Printf("启动失败: %v", err)
		os.Exit(1) // 唯一允许退出的一处
	}
}
```

**为什么这样写更好**：可测试（直接测 `run()`）、可清理（`defer` 生效）、错误路径统一。

:::

### 第 5 题

写一个三层错误包装链，然后用 `errors.Is` 证明它能穿透到最底层哨兵错误，并说明中间某一层若误用 `%v` 会发生什么。

::: details 第 5 题参考答案

```go
var ErrDB = errors.New("db down")

func store() error     { return fmt.Errorf("store: %w", ErrDB) }
func service() error   { return fmt.Errorf("service: %w", store()) }
func handler() error   { return fmt.Errorf("handler: %w", service()) }

func main() {
	err := handler()
	fmt.Println(err)                        // handler: service: store: db down
	fmt.Println(errors.Is(err, ErrDB))      // true
}
```

如果把 `service()` 里的 `%w` 换成 `%v`：

```go
func service() error { return fmt.Errorf("service: %v", store()) }
```

此时 `handler()` 的错误字符串看起来没变，但 `errors.Is(err, ErrDB)` 变成 `false`——`%v` 切断了 `Unwrap` 链。

**为什么这样写更好**：`%w` 让每一层既是「给人类看的完整上下文」，又是「给机器判断的 `Unwrap` 链」，两者兼得；`%v` 只保留前者，牺牲了程序化判断能力。

:::

---

## 小结

- **项目布局**：`cmd/` 只放 thin main，逻辑下沉 `internal/`，真正要公开的才放 `pkg/`；`internal/` 由编译器强制私有。
- **分层**：`handler → service → store`，依赖只朝内，层与层之间用**接口**衔接，接口由使用方定义（依赖倒置）。
- **依赖注入**：构造器注入 `NewXxx(dep)` 是主流；换实现 = 换注入对象，业务代码不改，可测试性由此而来。
- **Functional Options**：可选配置用 `Option func(*T)`，默认值集中、新增选项不破坏调用、解决零值歧义。
- **配置**：默认值 < 环境变量，解析与范围校验集中在 `LoadConfig`、尽早失败；注入 `lookup` 让配置可测试。
- **错误**：哨兵错误 + `%w` 逐层包装 + `errors.Is` 判断；`%v` 会切断 Unwrap 链。
- **日志**：`log/slog` 键值对结构化输出，`With` 挂固定字段，错误作为 value 传入。
- **版本**：`-ldflags -X` 注入发布版本，`go build` 自动嵌入 `vcs.revision`/`vcs.time`，`go version -m` 只读核对。
- **工程化**：`gofmt -l` → `go vet` → `staticcheck`/`golangci-lint` → `govulncheck`，分级兜底，收进 Makefile。

下一章（第 33 章）会把本章这些工程手段层层串起来，落成一个完整的分层、可测试、可观测的服务端项目。