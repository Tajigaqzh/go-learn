package go33_app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// Demo 是第 33 章的统一入口。
func Demo() {
	fmt.Println("========== go33_app: 综合实战：Go 服务端项目 ==========")

	demo33_1()
	demo33_2()
	demo33_3()
	demo33_4()
	demo33_5()
	demo33_6()
	demo33_7()
	demo33_8()

	fmt.Println("========== 综合实战：Go 服务端项目演示结束 ==========")
}

// newSlog 构造一个去掉时间戳、写到 w 的文本 logger，让演示输出每次一致。
func newSlog(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

// openDemoDB 打开内存 SQLite。:memory: 每个连接各有独立库，所以限定单连接共享数据。
func openDemoDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	return db
}

// assembleDemo 装配一个完整的演示应用：迁移 -> 仓储 -> service -> App。
func assembleDemo(logger *slog.Logger, idGen func() string) *App {
	db := openDemoDB()
	if err := Migrate(context.Background(), db); err != nil {
		panic(err)
	}
	svc := NewUserService(NewSQLiteUserRepo(db))
	return NewApp(svc, logger, WithRequestIDGen(idGen))
}

// counter 返回一个从 req-1 开始自增的请求 ID 生成器。
func counter() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("req-%d", n)
	}
}

// doRequest 发一个请求，返回状态码和去掉首尾空白后的响应体。
func doRequest(client *http.Client, method, url, body string) (int, string) {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return 0, err.Error()
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(b))
}

// mustPost 建一个用户，失败直接 panic（仅演示辅助）。
func mustPost(client *http.Client, base, body string) {
	status, resp := doRequest(client, http.MethodPost, base+"/api/v1/users", body)
	if status != http.StatusCreated {
		panic(fmt.Sprintf("创建失败: %d %s", status, resp))
	}
}

// netListenDemo 在 127.0.0.1 的随机端口上监听，供优雅关闭演示拿到真实地址。
func netListenDemo() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

// serveOn 在既有 listener 上跑一个带超时的 server，ctx 取消后优雅关闭再返回。
// 与 server.go 的 Run 同构，区别在调用方先拿到 listener，方便打印端口。
func serveOn(ctx context.Context, ln net.Listener, handler http.Handler, logger *slog.Logger) error {
	srv := NewHTTPServer(handler)
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("服务器已优雅关闭")
	return nil
}

// demo33_1 展示项目骨架与依赖装配。
func demo33_1() {
	fmt.Println("\n--- 33.1 项目骨架与依赖装配 ---")

	fmt.Println("本项目把一个「用户服务」拆成一包、多文件，每份职责单一：")
	fmt.Println()
	fmt.Println("  internal/chapter/go33_app/")
	fmt.Println("  ├─ app.go        # App 装配 + 路由 + handler")
	fmt.Println("  ├─ apierror.go   # 错误码 + JSON 写回")
	fmt.Println("  ├─ user.go       # 领域对象 + service（业务层）")
	fmt.Println("  ├─ sqlite.go     # SQLite 仓储（数据层） + 迁移")
	fmt.Println("  ├─ middleware.go # 请求 ID / 访问日志 / panic 恢复")
	fmt.Println("  ├─ server.go     # http.Server 超时 + 优雅关闭")
	fmt.Println("  ├─ config.go     # 配置加载")
	fmt.Println("  ├─ demo.go       # 本章演示入口")
	fmt.Println("  ├─ app_test.go   # 单元 + 集成测试")
	fmt.Println("  └─ bench_test.go # 基准测试")
	fmt.Println()
	fmt.Println("装配顺序（main 里就三行）：")
	fmt.Println("  db := sql.Open(sqlite3, cfg.DSN) -> Migrate -> NewSQLiteUserRepo(db)")
	fmt.Println("  svc := NewUserService(repo)      -> 业务依赖接口")
	fmt.Println("  app := NewApp(svc, logger)       -> 挂路由 + 中间件")
}

// demo33_2 演示配置加载的优先级与校验。
func demo33_2() {
	fmt.Println("\n--- 33.2 配置加载：默认值 < 环境变量 < 校验 ---")

	cfg, _ := LoadConfigFrom(func(string) (string, bool) { return "", false })
	fmt.Printf("无环境变量：   %+v\n", cfg)

	env := map[string]string{"APP_ADDR": "127.0.0.1:9090", "APP_DSN": "./data.db", "APP_LOG_LEVEL": "debug"}
	cfg, _ = LoadConfigFrom(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	fmt.Printf("覆盖后：       %+v\n", cfg)

	_, err := LoadConfigFrom(func(k string) (string, bool) { return "verbose", k == "APP_LOG_LEVEL" })
	fmt.Printf("非法日志级别： %v\n", err)
}

// demo33_3 演示数据库迁移，并打印实际建出的表结构。
func demo33_3() {
	fmt.Println("\n--- 33.3 数据库迁移 ---")

	db := openDemoDB()
	defer db.Close()
	if err := Migrate(context.Background(), db); err != nil {
		panic(err)
	}

	var schema string
	if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&schema); err != nil {
		panic(err)
	}
	fmt.Println("Migrate 后 users 表结构：")
	fmt.Println(schema)
	fmt.Println("要点：IF NOT EXISTS 让迁移幂等；真实项目把迁移拆成带版本的文件按序应用。")
}

// demo33_4 展示一条请求如何穿过 handler -> service -> repository 三层。
func demo33_4() {
	fmt.Println("\n--- 33.4 分层调用链 ---")

	app := assembleDemo(newSlog(io.Discard), counter())
	server := httptest.NewServer(app.Handler())
	defer server.Close()
	client := &http.Client{Timeout: 2 * time.Second}

	mustPost(client, server.URL, `{"name":"张三","email":"zhang@example.com"}`)
	mustPost(client, server.URL, `{"name":"李四","email":"li@example.com"}`)

	status, body := doRequest(client, http.MethodGet, server.URL+"/api/v1/users/1", "")
	fmt.Printf("GET  /api/v1/users/1  -> %d %s\n", status, body)

	status, body = doRequest(client, http.MethodGet, server.URL+"/api/v1/users", "")
	fmt.Printf("GET  /api/v1/users    -> %d %s\n", status, body)

	fmt.Println("调用链：handler 解析参数 -> service 校验 -> repo 执行 SQL -> 逐层返回")
}

// demo33_5 展示 REST 接口与统一错误码。
func demo33_5() {
	fmt.Println("\n--- 33.5 REST 接口与统一错误码 ---")

	app := assembleDemo(newSlog(io.Discard), counter())
	server := httptest.NewServer(app.Handler())
	defer server.Close()
	client := &http.Client{Timeout: 2 * time.Second}

	show := func(method, path, body string) {
		status, resp := doRequest(client, method, server.URL+path, body)
		fmt.Printf("%-4s %-22s -> %d %s\n", method, path, status, resp)
	}

	show(http.MethodPost, "/api/v1/users", `{"name":"","email":"a@b.com"}`)
	show(http.MethodPost, "/api/v1/users", `{"name":"张三","email":"bad-email"}`)
	show(http.MethodPost, "/api/v1/users", `{"name":"张三","email":"zhang@example.com"}`)
	show(http.MethodPost, "/api/v1/users", `{"name":"李四","email":"zhang@example.com"}`)
	show(http.MethodGet, "/api/v1/users/999", "")
	show(http.MethodGet, "/api/v1/users/abc", "")

	fmt.Println("每个失败响应统一是 {\"code\":..., \"message\":...}，code 给程序判断、message 给人看：")
	fmt.Println("  invalid_argument -> 400；not_found -> 404；already_exists -> 409；internal -> 500")
}

// demo33_6 展示中间件链：请求 ID、访问日志、panic 恢复。
func demo33_6() {
	fmt.Println("\n--- 33.6 中间件：请求 ID / 访问日志 / panic 恢复 ---")

	var logs bytes.Buffer
	app := assembleDemo(newSlog(&logs), counter())
	// 临时挂一条必定 panic 的路由，演示 recover 中间件。
	app.Mux().HandleFunc("GET /panic", func(http.ResponseWriter, *http.Request) {
		panic("故意触发的 panic")
	})

	server := httptest.NewServer(app.Handler())
	defer server.Close()
	client := &http.Client{Timeout: 2 * time.Second}

	status, body := doRequest(client, http.MethodGet, server.URL+"/healthz", "")
	fmt.Printf("GET /healthz -> %d %s（响应头带 X-Request-ID）\n", status, body)
	status, body = doRequest(client, http.MethodGet, server.URL+"/panic", "")
	fmt.Printf("GET /panic  -> %d %s\n", status, body)

	fmt.Println("写出的访问日志（logger 经 With 去掉了时间戳）：")
	fmt.Print(logs.String())
}

// demo33_7 演示在真实端口上的优雅关闭。
func demo33_7() {
	fmt.Println("\n--- 33.7 优雅关闭 ---")

	var logs bytes.Buffer
	logger := newSlog(&logs)
	app := assembleDemo(logger, counter())

	ln, err := netListenDemo()
	if err != nil {
		fmt.Printf("监听失败: %v\n", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveOn(ctx, ln, app.Handler(), logger) }()

	addr := ln.Addr().String()
	fmt.Printf("监听地址（端口每次不同）: %s\n", addr)
	client := &http.Client{Timeout: 2 * time.Second}
	status, body := doRequest(client, http.MethodGet, "http://"+addr+"/healthz", "")
	fmt.Printf("GET /healthz -> %d %s\n", status, body)

	cancel()
	fmt.Printf("取消 ctx 后 serveOn 返回: %v\n", <-done)
	fmt.Print("服务器日志：")
	fmt.Print(logs.String())
}

// demo33_8 简述测试、基准与 Docker 打包（详细输出见正文）。
func demo33_8() {
	fmt.Println("\n--- 33.8 测试、基准与 Docker 打包 ---")

	fmt.Println("本项目的质量保障分三层：")
	fmt.Println("  - 单元测试：service 逻辑用 fakeRepo 隔离（app_test.go）")
	fmt.Println("  - 集成测试：httptest + 内存 SQLite 走完整 HTTP 链路（app_test.go）")
	fmt.Println("  - 基准测试：BenchmarkCreateUser 度量 handler 单次创建开销（bench_test.go）")
	fmt.Println()
	fmt.Println("具体命令与真实输出：")
	fmt.Println("  go test ./internal/chapter/go33_app/ -v      # 跑测试")
	fmt.Println("  go test ./internal/chapter/go33_app/ -bench . -benchmem   # 跑基准")
	fmt.Println()
	fmt.Println("Docker 打包用多阶段构建（golang 编译 -> distroless 运行），见正文 33.9。")
}
