// Package go32_engineering 演示 Go 工程实践与项目结构。
//
// 本章聚焦「把零散的代码组织成可维护、可测试的工程」：标准项目布局、
// 分层与依赖方向、手动依赖注入、Functional Options 惯用法、配置与环境区分、
// 错误与日志规范，以及 go generate、版本注入、Makefile、静态检查与依赖安全。
//
// 为让每个主题都能直接运行并对照输出，本章用一个迷你「用户服务」作为贯穿示例：
// handler → service → store 三层，依赖只朝内（接口）流动，方便替换与测试。
package go32_engineering

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"time"
)

// 章节级哨兵错误：包内共享，第 32.6 节用它们演示 errors.Is 的穿透能力。
var (
	// ErrNotFound 表示记录不存在，是「数据层」对外暴露的稳定错误契约。
	ErrNotFound = errors.New("user not found")
	// ErrConflict 表示主键冲突（例如重复创建同名用户）。
	ErrConflict = errors.New("user already exists")
)

// User 是贯穿全章的领域对象，ID 为唯一标识。
type User struct {
	ID   int
	Name string
}

// UserStore 是数据访问接口，由 service 层依赖；实现方可能是内存、SQL 或测试替身。
type UserStore interface {
	Get(id int) (User, error)
	List() []User
}

// memoryStore 是 UserStore 的内存实现，仅用于演示分层与依赖注入。
type memoryStore struct {
	users map[int]User
}

// newMemoryStore 构造一个含两条初始数据的 memoryStore。
func newMemoryStore() *memoryStore {
	return &memoryStore{users: map[int]User{
		1: {ID: 1, Name: "张三"},
		2: {ID: 2, Name: "李四"},
	}}
}

// Get 按 ID 返回用户，不存在时返回 ErrNotFound。
func (s *memoryStore) Get(id int) (User, error) {
	u, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

// List 返回全部用户，按 ID 升序保证输出稳定。
func (s *memoryStore) List() []User {
	ids := make([]int, 0, len(s.users))
	for id := range s.users {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]User, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.users[id])
	}
	return out
}

// fakeStore 是测试替身：行为可控，还能记录调用次数，用于验证 service 逻辑而不依赖真实存储。
type fakeStore struct {
	users    map[int]User
	getCalls int
}

// Get 记录一次调用；命中则返回用户，否则返回 ErrNotFound。
func (f *fakeStore) Get(id int) (User, error) {
	f.getCalls++
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return User{}, ErrNotFound
}

// List 返回 fakeStore 中的用户（演示用，顺序不作保证）。
func (f *fakeStore) List() []User {
	out := make([]User, 0, len(f.users))
	for _, u := range f.users {
		out = append(out, u)
	}
	return out
}

// UserService 是业务层：只依赖 UserStore 接口，不关心具体实现。
type UserService struct {
	store UserStore
}

// NewUserService 通过构造器注入 UserStore，这是 Go 里最常用的依赖注入方式。
func NewUserService(store UserStore) *UserService {
	return &UserService{store: store}
}

// GetByID 按 ID 取用户；参数非法直接返回错误，命中的错误用 %w 补一层上下文。
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

// Server 演示 Functional Options：可选配置项里既有零值派生默认值，也有指向可替换组件的字段。
type Server struct {
	addr     string
	timeout  time.Duration
	maxConns int
	logger   *slog.Logger
}

// Option 是修改 Server 的选项函数，用户通过可变参数传入。
type Option func(*Server)

// WithTimeout 设置请求超时。
func WithTimeout(d time.Duration) Option {
	return func(s *Server) { s.timeout = d }
}

// WithMaxConns 设置最大并发连接数。
func WithMaxConns(n int) Option {
	return func(s *Server) { s.maxConns = n }
}

// WithLogger 注入自定义 logger。
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) { s.logger = l }
}

// NewServer 先给出默认配置，再逐个应用 Option；新增配置项无需改动已有调用方。
func NewServer(addr string, opts ...Option) *Server {
	s := &Server{
		addr:     addr,
		timeout:  3 * time.Second,
		maxConns: 100,
		logger:   slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Config 是加载环境变量后得到的运行配置。
type Config struct {
	Addr     string
	Port     int
	LogLevel string
}

// LoadConfig 用 os.LookupEnv 加载配置，等价于 LoadConfigFrom(os.LookupEnv)。
func LoadConfig() (*Config, error) {
	return LoadConfigFrom(os.LookupEnv)
}

// LoadConfigFrom 按「默认值 < 环境变量」的优先级构造配置；lookup 注入是为了可测试。
func LoadConfigFrom(lookup func(string) (string, bool)) (*Config, error) {
	cfg := &Config{
		Addr:     "0.0.0.0",
		Port:     8080,
		LogLevel: "info",
	}
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
	if v, ok := lookup("APP_LOG_LEVEL"); ok && v != "" {
		cfg.LogLevel = v
	}
	return cfg, nil
}

// newDemoLogger 构造一个去掉时间戳的文本 logger，让演示输出每次一致。
func newDemoLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

// Demo 是第 32 章的统一入口。
func Demo() {
	fmt.Println("========== go32_engineering: 工程实践与项目结构 ==========")

	demo32_1()
	demo32_2()
	demo32_3()
	demo32_4()
	demo32_5()
	demo32_6()
	demo32_7()
	demo32_8()
	demo32_9()

	fmt.Println("========== 工程实践与项目结构演示结束 ==========")
}

// demo32_1 讲解 cmd / internal / pkg 项目布局。
func demo32_1() {
	fmt.Println("\n--- 32.1 项目布局：cmd / internal / pkg ---")

	fmt.Println("一个可长期维护的 Go 工程，目录按职责划分：")
	fmt.Println()
	fmt.Println("  go-learn/")
	fmt.Println("  ├─ cmd/          # 可执行程序入口，每个子目录一个 main")
	fmt.Println("  ├─ internal/     # 私有代码，只能被本模块内部 import")
	fmt.Println("  ├─ pkg/          # 可被外部项目 import 的公共库")
	fmt.Println("  ├─ test/         # 跨包集成测试")
	fmt.Println("  ├─ docs/         # 文档")
	fmt.Println("  └─ go.mod        # 模块定义")
	fmt.Println()
	fmt.Println("要点：")
	fmt.Println("  - cmd/ 只放薄薄的 main，真正的逻辑下沉到 internal/")
	fmt.Println("  - internal/ 是编译器强制「私有」的边界，外部模块 import 会报错")
	fmt.Println("  - pkg/ 只放确实要公开的库，不要一上来就建 pkg/")
	fmt.Println("  - 本仓库就遵循这个布局：入口在 cmd/go-learn，各章代码在 internal/chapter")
}

// demo32_2 讲解分层与依赖方向（依赖倒置）。
func demo32_2() {
	fmt.Println("\n--- 32.2 分层与依赖方向：接口朝内依赖 ---")

	store := newMemoryStore()
	svc := NewUserService(store)

	fmt.Println("依赖方向（只能朝一个方向引用）：")
	fmt.Println("  handler ──> service ──> UserStore(接口)")
	fmt.Println("                               ↑ 实现")
	fmt.Println("                          memoryStore")
	fmt.Println()
	fmt.Println("关键点：service 只 import 接口，memoryStore 是被注入进来的实现。")
	fmt.Println("接口由「使用方」定义，而不是由实现方定义——依赖倒置。")
	fmt.Println()

	u, err := svc.GetByID(1)
	fmt.Printf("GetByID(1) = %+v, err = %v\n", u, err)

	u, err = svc.GetByID(99)
	fmt.Printf("GetByID(99) = %+v, err = %v\n", u, err)
	fmt.Println("  -> 错误里带着 service 与 store 两层上下文，便于定位")
}

// demo32_3 讲解手动依赖注入与可测试性。
func demo32_3() {
	fmt.Println("\n--- 32.3 手动依赖注入：换一个实现就能换一套行为 ---")

	// 同一套 service 逻辑，注入真实实现：
	realSvc := NewUserService(newMemoryStore())
	u, _ := realSvc.GetByID(2)
	fmt.Printf("注入 memoryStore：      GetByID(2) = %s\n", u.Name)

	// 换注入一个测试替身，行为立刻不同，service 代码一行不用改：
	fake := &fakeStore{users: map[int]User{2: {ID: 2, Name: "测试替身"}}}
	fakeSvc := NewUserService(fake)
	u, _ = fakeSvc.GetByID(2)
	fmt.Printf("注入 fakeStore：        GetByID(2) = %s\n", u.Name)

	_, _ = fakeSvc.GetByID(2)
	fmt.Printf("fakeStore.Get 被调用 %d 次（可验证交互）\n", fake.getCalls)

	fmt.Println("结论：构造器注入（NewXxx(dep)）比包级全局变量/单例更可控、更好测。")
}

// demo32_4 讲解 Functional Options 惯用法。
func demo32_4() {
	fmt.Println("\n--- 32.4 Functional Options：优雅的可选配置 ---")

	def := NewServer(":8080")
	fmt.Printf("默认服务：        addr=%s timeout=%s maxConns=%d\n",
		def.addr, def.timeout, def.maxConns)

	custom := NewServer(":9090",
		WithTimeout(500*time.Millisecond),
		WithMaxConns(10),
		WithLogger(newDemoLogger()),
	)
	fmt.Printf("定制服务：        addr=%s timeout=%s maxConns=%d\n",
		custom.addr, custom.timeout, custom.maxConns)

	fmt.Println("优点：新增选项不破坏已有调用；默认值集中在一处；选项可自由组合。")
	fmt.Println("对比：固定结构体字面量在字段变多后难以区分「未设置」和「设为零值」。")
}

// demo32_5 讲解配置与环境区分，并演示优先级与校验。
func demo32_5() {
	fmt.Println("\n--- 32.5 配置与环境：默认值 < 环境变量 < 显式校验 ---")

	none := func(string) (string, bool) { return "", false }
	cfg, _ := LoadConfigFrom(none)
	fmt.Printf("无环境变量：      %+v\n", cfg)

	environ := func(k string) (string, bool) {
		m := map[string]string{
			"APP_ADDR":      "127.0.0.1",
			"APP_PORT":      "9090",
			"APP_LOG_LEVEL": "debug",
		}
		v, ok := m[k]
		return v, ok
	}
	cfg, _ = LoadConfigFrom(environ)
	fmt.Printf("覆盖后：          %+v\n", cfg)

	bad := func(k string) (string, bool) { return "not-a-number", k == "APP_PORT" }
	if _, err := LoadConfigFrom(bad); err != nil {
		fmt.Printf("非法 APP_PORT：   %v\n", err)
	}

	fmt.Println("原则：环境变量只做「字符串覆盖」，解析与范围校验集中在 LoadConfig，尽早失败。")
}

// demo32_6 讲解错误处理规范：哨兵错误 + %w 包装 + errors.Is。
func demo32_6() {
	fmt.Println("\n--- 32.6 错误规范：哨兵错误、%w 包装、errors.Is ---")

	svc := NewUserService(newMemoryStore())

	_, err := svc.GetByID(99)
	fmt.Println("完整错误：", err)
	fmt.Println("errors.Is(err, ErrNotFound) =", errors.Is(err, ErrNotFound))
	fmt.Println("errors.Is(err, ErrConflict) =", errors.Is(err, ErrConflict))

	fmt.Println()
	fmt.Println("规范：")
	fmt.Println("  - 用哨兵错误给调用方一个稳定的判断锚点（errors.Is）")
	fmt.Println("  - 逐层用 %w 补上下文，不吞错、不把错误变成字符串")
	fmt.Println("  - 库函数只 return err，不打日志；最上层（main/handler）统一记录")
	fmt.Println("  反例：log.Fatalf 会 os.Exit(1)，库代码里调用会杀死宿主进程且无法 defer")
}

// demo32_7 讲解结构化日志规范（log/slog）。
func demo32_7() {
	fmt.Println("\n--- 32.7 结构化日志：log/slog ---")

	logger := newDemoLogger()

	logger.Info("server starting", "addr", "0.0.0.0:8080")
	userLogger := logger.With("service", "user")
	userLogger.Debug("store connected", "impl", "memory")
	userLogger.Error("get user failed", "user_id", 42, "err", ErrNotFound)
	userLogger.Warn("slow query", "duration_ms", int64(350))

	fmt.Println("要点：")
	fmt.Println("  - 用键值对字段承载上下文，机器可解析；不要靠拼字符串日志")
	fmt.Println("  - slog.With 派生子 logger，把固定字段（service 等）挂在上下文里")
	fmt.Println("  - 错误作为 value 传给 slog，不要 fmt.Errorf 里夹带日志文案")
}

// demo32_8 讲解版本注入（-ldflags）与构建信息（debug.ReadBuildInfo）。
func demo32_8() {
	fmt.Println("\n--- 32.8 版本注入（-ldflags）与构建信息 ---")

	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("未拿到构建信息")
		return
	}
	fmt.Printf("Go 版本：  %s\n", info.GoVersion)
	fmt.Printf("模块路径： %s\n", info.Main.Path)
	fmt.Printf("模块版本： %s\n", info.Main.Version)
	fmt.Println("关键构建设置：")
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision", "vcs.time", "vcs.modified", "CGO_ENABLED", "GOOS", "GOARCH":
			fmt.Printf("  %s = %s\n", s.Key, s.Value)
		}
	}
	fmt.Println()
	fmt.Println("注意：go run 不落地二进制，所以上面看不到 git 指纹，版本是 (devel)。")
	fmt.Println("用 go build / go install 时才嵌入 vcs.revision / vcs.time / vcs.modified（见正文）。")
	fmt.Println()
	fmt.Println("版本注入惯用法：在 main 包声明 var version = \"dev\"，构建时用")
	fmt.Println("  go build -ldflags \"-X main.version=v1.2.3\"")
	fmt.Println("覆盖它；vcs.revision 是 git 自动嵌入的提交指纹，二者互补。")
	fmt.Println("go generate 则用于在源文件里声明 //go:generate 指令，由 go generate 触发代码生成。")
}

// demo32_9 讲解 Makefile、静态检查与依赖安全。
func demo32_9() {
	fmt.Println("\n--- 32.9 Makefile、静态检查与依赖安全 ---")

	fmt.Println("把常用命令收进 Makefile，降低「记命令」成本：")
	fmt.Println()
	fmt.Println("  fmt:      gofmt -l -w .")
	fmt.Println("  vet:      go vet ./...")
	fmt.Println("  test:     go test -race ./...")
	fmt.Println("  build:    go build -ldflags \"-s -w\" -o bin/app ./cmd/go-learn")
	fmt.Println("  vuln:     govulncheck ./...")
	fmt.Println()
	fmt.Println("等价于直接执行上面的 go 命令，不依赖 make 是否安装。")
	fmt.Println("静态检查分级：gofmt（格式） -> go vet（易错点） -> staticcheck/golangci-lint（更严）")
	fmt.Println("依赖安全：govulncheck 基于漏洞库扫描依赖；go list -m -u 查看可升级版本。")
	fmt.Println("注：本机未安装 make/staticcheck/golangci-lint/govulncheck，命令以 go 原生命令为准。")
}
