// Package go26_http_server 演示用标准库写 HTTP 服务端与 REST API。
//
// 本章覆盖：
//   - http.Handler 与 http.HandlerFunc
//   - ServeMux 与 Go 1.22 的方法 + 通配符路由
//   - 请求解析（路径参数、查询参数、JSON body）与 JSON 响应
//   - 统一错误响应与状态码
//   - 中间件链：请求 ID、访问日志、panic 恢复、CORS、限流
//   - http.Server 超时与优雅关闭
//   - httptest 与可测试性
//
// 演示全部在 127.0.0.1 的临时端口上运行，不依赖外部网络；时钟、请求 ID 都做了
// 注入，所以除了监听端口本身，输出都是可复现的。
package go26_http_server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// demoTime 是演示用的固定时刻，让响应体里的时间戳稳定可复现。
func demoTime() time.Time {
	return time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC)
}

// Demo 运行第 26 章所有演示。
func Demo() {
	fmt.Println("========== go26_http_server: HTTP 服务端与 REST API ==========")

	fmt.Println("\n--- 26.1 Handler 与 HandlerFunc ---")
	demoHandler()

	fmt.Println("\n--- 26.2 ServeMux 与 Go 1.22 路由增强 ---")
	demoRouting()

	fmt.Println("\n--- 26.3 请求解析与 JSON 响应 ---")
	demoRequestParsing()

	fmt.Println("\n--- 26.4 统一错误响应 ---")
	demoErrorResponse()

	fmt.Println("\n--- 26.5 中间件链 ---")
	demoMiddleware()

	fmt.Println("\n--- 26.6 服务器超时与优雅关闭 ---")
	demoGracefulShutdown()

	fmt.Println("\n--- 26.7 httptest 与生产化细节 ---")
	demoHttptest()

	fmt.Println("\n========== HTTP 服务端与 REST API 演示结束 ==========")
}

// --- 26.1 Handler 与 HandlerFunc ---

// greetingHandler 用结构体实现 http.Handler。
type greetingHandler struct {
	from string
}

// ServeHTTP 让 greetingHandler 满足 http.Handler 接口。
func (h greetingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "hello, %s", h.from)
}

// demoHandler 对比「结构体实现接口」和「函数适配成接口」两种写法。
func demoHandler() {
	rec := httptest.NewRecorder()
	greetingHandler{from: "结构体"}.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))
	fmt.Printf("1) 结构体实现 ServeHTTP: %d %s\n", rec.Code, rec.Body.String())

	rec = httptest.NewRecorder()
	http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello, 函数")
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))
	fmt.Printf("2) http.HandlerFunc 适配函数: %d %s\n", rec.Code, rec.Body.String())

	var handler http.Handler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	fmt.Printf("3) http.HandlerFunc 本身就是 http.Handler: %t\n", handler != nil)
	fmt.Println("   一个 handler 只做三件事：读请求、写响应；状态码由 WriteHeader 决定")
}

// --- 26.2 ServeMux 与 Go 1.22 路由增强 ---

// demoRouting 展示方法模式、路径通配符，以及 ServeMux 自动给出的 404/405。
func demoRouting() {
	client, base, closeServer := newDemoServer(demoOptions()...)
	defer closeServer()

	mustCreateTask(client, base, "写第 26 章")

	fmt.Println("路由注册（Go 1.22 起支持「方法 + 路径」模式）:")
	for _, pattern := range []string{
		"GET /healthz",
		"GET /api/tasks",
		"POST /api/tasks",
		"GET /api/tasks/{id}",
		"PATCH /api/tasks/{id}",
		"DELETE /api/tasks/{id}",
	} {
		fmt.Printf("  %s\n", pattern)
	}
	fmt.Println("实际请求:")
	show(client, base, http.MethodGet, "/api/tasks/1", "")
	show(client, base, http.MethodPost, "/api/tasks/1", "")
	show(client, base, http.MethodGet, "/api/absent", "")
	show(client, base, http.MethodGet, "/api/tasks/abc", "")
}

// --- 26.3 请求解析与 JSON 响应 ---

// demoRequestParsing 演示路径参数、查询参数和 JSON 请求体的解析。
func demoRequestParsing() {
	client, base, closeServer := newDemoServer(demoOptions()...)
	defer closeServer()

	mustCreateTask(client, base, "写第 26 章")
	mustCreateTask(client, base, "复习 pprof")
	show(client, base, http.MethodPatch, "/api/tasks/2", `{"done":true}`)

	fmt.Println("查询参数过滤与截断:")
	show(client, base, http.MethodGet, "/api/tasks", "")
	show(client, base, http.MethodGet, "/api/tasks?done=true", "")
	show(client, base, http.MethodGet, "/api/tasks?limit=1", "")

	fmt.Println("失败路径:")
	show(client, base, http.MethodPost, "/api/tasks", `{"title":""}`)
	show(client, base, http.MethodPost, "/api/tasks", `{"title":123}`)
	show(client, base, http.MethodPost, "/api/tasks", `{"titel":"拼错的字段"}`)
	show(client, base, http.MethodGet, "/api/tasks?limit=abc", "")
}

// --- 26.4 统一错误响应 ---

// demoErrorResponse 对比统一错误体与 http.Error 的默认行为。
func demoErrorResponse() {
	client, base, closeServer := newDemoServer(demoOptions()...)
	defer closeServer()

	fmt.Println("统一错误体（code 给程序判断，message 给人看）:")
	show(client, base, http.MethodGet, "/api/tasks/404", "")

	rec := httptest.NewRecorder()
	http.Error(rec, "任务不存在", http.StatusNotFound)
	fmt.Printf("http.Error 的默认行为: %d %q Content-Type=%s\n",
		rec.Code, rec.Body.String(), rec.Header().Get("Content-Type"))

	rec = httptest.NewRecorder()
	writeJSON(rec, http.StatusBadRequest, APIError{Code: "invalid_argument", Message: "title 不能为空"})
	fmt.Printf("writeJSON 的结果: %d %s", rec.Code, rec.Body.String())
	fmt.Println("状态码只能写一次：先 WriteHeader 再写 body，顺序颠倒会触发 superfluous 警告")
}

// --- 26.5 中间件链 ---

// demoMiddleware 把中间件串起来，并展示 panic 恢复与 CORS 预检。
func demoMiddleware() {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)

	requestCount := 0
	nextID := func() string {
		requestCount++
		return fmt.Sprintf("demo-%d", requestCount)
	}

	client, base, closeServer := newDemoServerWithLogger(logger, WithRequestIDGenerator(nextID))
	defer closeServer()

	mustCreateTask(client, base, "写第 26 章")
	show(client, base, http.MethodGet, "/api/tasks", "")
	show(client, base, http.MethodGet, "/api/tasks/1", "")

	// 带 X-Request-ID 的请求会沿用调用方的 ID，便于跨服务串联日志。
	status, _, _ := doRequest(client, http.MethodGet, base+"/healthz", "", map[string]string{
		"X-Request-ID": "from-client",
	})
	fmt.Printf("调用方自带 X-Request-ID 时沿用: %d\n", status)

	// 预检请求由 CORS 中间件直接回答，不进入业务逻辑。
	status, header, _ := doRequest(client, http.MethodOptions, base+"/api/tasks", "", map[string]string{
		"Origin":                        "https://example.com",
		"Access-Control-Request-Method": "POST",
	})
	fmt.Printf("预检: OPTIONS /api/tasks -> %d Allow-Origin=%s Allow-Methods=%s\n",
		status,
		header.Get("Access-Control-Allow-Origin"),
		header.Get("Access-Control-Allow-Methods"))

	fmt.Println("panic 恢复（未捕获的 panic 会掐断连接，Recover 把它变成 500）:")
	show(client, base, http.MethodGet, "/panic", "")

	fmt.Println("中间件写出的访问日志:")
	fmt.Print(logs.String())
}

// --- 26.6 服务器超时与优雅关闭 ---

// demoGracefulShutdown 在真实端口上跑一遍「启动 -> 服务 -> 关闭」。
func demoGracefulShutdown() {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)

	store := newStoreWithClock(demoTime)
	api := NewAPI(store, logger, WithRequestIDGenerator(func() string { return "demo-1" }))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("监听失败: %v\n", err)
		return
	}

	srv := NewServer(api.Handler(), logger)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ServeUntilCanceled(ctx, srv, ln, logger)
	}()

	addr := ln.Addr().String()
	base := "http://" + addr
	fmt.Printf("监听地址: %s（端口每次运行都不同）\n", addr)
	fmt.Printf("超时配置: ReadHeaderTimeout=%v ReadTimeout=%v WriteTimeout=%v IdleTimeout=%v\n",
		readHeaderTimeout, readTimeout, writeTimeout, idleTimeout)

	client := &http.Client{Timeout: 2 * time.Second}
	show(client, base, http.MethodGet, "/healthz", "")

	cancel()
	fmt.Printf("取消 ctx 后 ServeUntilCanceled 返回: %v\n", <-done)
	fmt.Print("服务器日志: ")
	fmt.Print(logs.String())

	_, err = client.Get(base + "/healthz")
	fmt.Printf("关闭后再次请求失败: %t\n", err != nil)
}

// --- 26.7 httptest 与生产化细节 ---

// demoHttptest 演示不依赖真实端口的测试写法，以及限流和请求体上限。
func demoHttptest() {
	store := newStoreWithClock(demoTime)
	api := NewAPI(store, log.New(io.Discard, "", 0))

	// httptest.NewRequest + NewRecorder：把 handler 当普通函数调用，毫秒级完成。
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	fmt.Printf("httptest.NewRecorder: GET /healthz -> %d %s", rec.Code, rec.Body.String())
	fmt.Printf("响应头里的请求 ID 自动生成: %t\n", rec.Header().Get("X-Request-ID") != "")

	// 注入时钟的令牌桶：桶容量 3，每秒补 1 个。
	now := demoTime()
	limiter := newLimiterWithClock(1, 3, func() time.Time { return now })
	client, base, closeServer := newDemoServer(WithLimiter(limiter))
	defer closeServer()

	fmt.Print("令牌桶（容量 3，1 个/秒）连打 5 次: ")
	for i := 0; i < 5; i++ {
		status, _, _ := doRequest(client, http.MethodGet, base+"/healthz", "", nil)
		fmt.Printf("%d ", status)
	}
	now = now.Add(time.Second)
	status, _, _ := doRequest(client, http.MethodGet, base+"/healthz", "", nil)
	fmt.Printf("；前进 1 秒后: %d\n", status)

	// 请求体上限 1 KB：单独开一个不限流的服务，避免上面的令牌桶干扰结果。
	plainClient, plainBase, closePlain := newDemoServer()
	defer closePlain()
	big := `{"title":"` + strings.Repeat("a", 2<<10) + `"}`
	status, header, body := doRequest(plainClient, http.MethodPost, plainBase+"/api/tasks", big,
		map[string]string{"Content-Type": "application/json"})
	fmt.Printf("2 KB 请求体: %d %s（Content-Type=%s）\n",
		status, strings.TrimSpace(body), header.Get("Content-Type"))
}

// --- 演示辅助 ---

// demoOptions 返回演示用的固定请求 ID 生成器。
func demoOptions() []Option {
	requestCount := 0
	return []Option{
		WithRequestIDGenerator(func() string {
			requestCount++
			return fmt.Sprintf("demo-%d", requestCount)
		}),
	}
}

// newDemoServer 用一个丢弃日志的 logger 启动临时服务。
func newDemoServer(options ...Option) (*http.Client, string, func()) {
	return newDemoServerWithLogger(log.New(io.Discard, "", 0), options...)
}

// newDemoServerWithLogger 启动临时 HTTP 服务，返回客户端、基地址和关闭函数。
func newDemoServerWithLogger(logger *log.Logger, options ...Option) (*http.Client, string, func()) {
	store := newStoreWithClock(demoTime)
	api := NewAPI(store, logger, options...)

	// 额外挂一个必定 panic 的路由，用来演示 Recover 中间件。
	api.Mux().HandleFunc("GET /panic", func(http.ResponseWriter, *http.Request) {
		panic("故意触发的 panic")
	})

	server := httptest.NewServer(api.Handler())
	client := &http.Client{Timeout: 2 * time.Second}
	return client, server.URL, server.Close
}

// mustCreateTask 建一条固定标题的任务，失败直接 panic（仅演示使用）。
func mustCreateTask(client *http.Client, base, title string) {
	status, _, body := doRequest(client, http.MethodPost, base+"/api/tasks",
		fmt.Sprintf(`{"title":%q}`, title),
		map[string]string{"Content-Type": "application/json"})
	if status != http.StatusCreated {
		panic(fmt.Sprintf("创建任务失败: %d %s", status, body))
	}
}

// show 发一个请求并打印「方法 路径 -> 状态码 响应体」。
func show(client *http.Client, base, method, path, body string) {
	headers := map[string]string{}
	if body != "" {
		headers["Content-Type"] = "application/json"
	}
	status, header, respBody := doRequest(client, method, base+path, body, headers)

	line := fmt.Sprintf("%-6s %-34s -> %d %s", method, path, status, strings.TrimSpace(respBody))
	if allow := header.Get("Allow"); allow != "" {
		line += "  Allow: " + allow
	}
	if location := header.Get("Location"); location != "" {
		line += "  Location: " + location
	}
	fmt.Println(strings.TrimRight(line, " "))
}

// doRequest 发一个请求，返回状态码、响应头和响应体。
func doRequest(client *http.Client, method, url, body string, headers map[string]string) (int, http.Header, string) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, http.Header{}, err.Error()
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, http.Header{}, err.Error()
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Header, err.Error()
	}
	return resp.StatusCode, resp.Header, string(payload)
}
