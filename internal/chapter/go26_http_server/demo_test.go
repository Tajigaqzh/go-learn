package go26_http_server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestAPI 构造一个日志丢弃、时钟固定的 API，供测试使用。
func newTestAPI(options ...Option) *API {
	return NewAPI(newStoreWithClock(demoTime), log.New(io.Discard, "", 0), options...)
}

// serve 直接调用 handler，不经过真实网络。
func serve(handler http.Handler, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// jsonHeaders 是发 JSON 请求时需要的头。
func jsonHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json"}
}

// decodeError 把响应体解析成统一错误结构。
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) APIError {
	t.Helper()

	var apiErr APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("响应体不是合法的 APIError: %v（原文 %q）", err, rec.Body.String())
	}
	return apiErr
}

// TestRouting 覆盖路由匹配、方法不匹配和路径参数校验。
func TestRouting(t *testing.T) {
	api := newTestAPI()
	if rec := serve(api.Handler(), http.MethodPost, "/api/tasks", `{"title":"写第 26 章"}`, jsonHeaders()); rec.Code != http.StatusCreated {
		t.Fatalf("创建任务失败: %d %s", rec.Code, rec.Body.String())
	}

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
	}{
		{"按 id 查询", http.MethodGet, "/api/tasks/1", http.StatusOK, `"title":"写第 26 章"`},
		{"方法不匹配", http.MethodPost, "/api/tasks/1", http.StatusMethodNotAllowed, "Method Not Allowed"},
		{"路径不存在", http.MethodGet, "/api/absent", http.StatusNotFound, "404 page not found"},
		{"id 不是整数", http.MethodGet, "/api/tasks/abc", http.StatusBadRequest, `"code":"invalid_argument"`},
		{"id 不是正数", http.MethodGet, "/api/tasks/0", http.StatusBadRequest, "id 必须大于 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(api.Handler(), tt.method, tt.target, "", nil)
			if rec.Code != tt.wantStatus {
				t.Errorf("状态码：期望 %d，实际 %d", tt.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("响应体 %q 里找不到 %q", rec.Body.String(), tt.wantBody)
			}
		})
	}

	t.Run("405 带 Allow 头", func(t *testing.T) {
		rec := serve(api.Handler(), http.MethodPost, "/api/tasks/1", "", nil)
		allow := rec.Header().Get("Allow")
		for _, method := range []string{"GET", "PATCH", "DELETE"} {
			if !strings.Contains(allow, method) {
				t.Errorf("Allow 头 %q 里应当包含 %s", allow, method)
			}
		}
	})
}

// TestCreateTaskValidation 覆盖请求体校验的各种失败路径。
func TestCreateTaskValidation(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
		wantCode    string
	}{
		{"缺少 Content-Type", `{"title":"x"}`, "", http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"JSON 不合法", `{"title":123}`, "application/json", http.StatusBadRequest, "invalid_json"},
		{"字段拼错", `{"titel":"x"}`, "application/json", http.StatusBadRequest, "invalid_json"},
		{"标题为空", `{"title":"   "}`, "application/json", http.StatusBadRequest, "invalid_argument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.contentType != "" {
				headers["Content-Type"] = tt.contentType
			}
			api := newTestAPI()
			rec := serve(api.Handler(), http.MethodPost, "/api/tasks", tt.body, headers)

			if rec.Code != tt.wantStatus {
				t.Fatalf("状态码：期望 %d，实际 %d（%s）", tt.wantStatus, rec.Code, rec.Body.String())
			}
			if got := decodeError(t, rec).Code; got != tt.wantCode {
				t.Errorf("错误码：期望 %q，实际 %q", tt.wantCode, got)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("错误响应应当是 JSON，实际 Content-Type=%q", ct)
			}
		})
	}
}

// TestCreateTaskResponse 验证 201、Location 头和响应体。
func TestCreateTaskResponse(t *testing.T) {
	api := newTestAPI()
	rec := serve(api.Handler(), http.MethodPost, "/api/tasks", `{"title":"写第 26 章"}`, jsonHeaders())

	if rec.Code != http.StatusCreated {
		t.Fatalf("状态码：期望 201，实际 %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/api/tasks/1" {
		t.Errorf("Location：期望 /api/tasks/1，实际 %q", got)
	}

	var task Task
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatalf("响应体解析失败: %v", err)
	}
	if task.ID != 1 || task.Title != "写第 26 章" || task.Done {
		t.Errorf("任务内容不对: %+v", task)
	}
	if !task.CreatedAt.Equal(demoTime()) {
		t.Errorf("创建时间应当来自注入的时钟，实际 %v", task.CreatedAt)
	}
}

// TestListFilters 验证 done 过滤和 limit 截断。
func TestListFilters(t *testing.T) {
	api := newTestAPI()
	handler := api.Handler()
	serve(handler, http.MethodPost, "/api/tasks", `{"title":"任务一"}`, jsonHeaders())
	serve(handler, http.MethodPost, "/api/tasks", `{"title":"任务二"}`, jsonHeaders())
	serve(handler, http.MethodPatch, "/api/tasks/2", `{"done":true}`, jsonHeaders())

	tests := []struct {
		target string
		want   int
	}{
		{"/api/tasks", 2},
		{"/api/tasks?done=true", 1},
		{"/api/tasks?done=false", 1},
		{"/api/tasks?limit=1", 1},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			rec := serve(handler, http.MethodGet, tt.target, "", nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("状态码：期望 200，实际 %d", rec.Code)
			}
			var payload struct {
				Count int `json:"count"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("响应体解析失败: %v", err)
			}
			if payload.Count != tt.want {
				t.Errorf("count：期望 %d，实际 %d", tt.want, payload.Count)
			}
		})
	}

	t.Run("limit 不是整数", func(t *testing.T) {
		rec := serve(handler, http.MethodGet, "/api/tasks?limit=abc", "", nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("状态码：期望 400，实际 %d", rec.Code)
		}
	})
}

// TestUpdateAndDelete 覆盖 PATCH 的局部更新与 DELETE 的语义。
func TestUpdateAndDelete(t *testing.T) {
	api := newTestAPI()
	handler := api.Handler()
	serve(handler, http.MethodPost, "/api/tasks", `{"title":"任务一"}`, jsonHeaders())

	t.Run("只改 done", func(t *testing.T) {
		rec := serve(handler, http.MethodPatch, "/api/tasks/1", `{"done":true}`, jsonHeaders())
		if rec.Code != http.StatusOK {
			t.Fatalf("状态码：期望 200，实际 %d", rec.Code)
		}
		var task Task
		if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
			t.Fatalf("响应体解析失败: %v", err)
		}
		if !task.Done || task.Title != "任务一" {
			t.Errorf("局部更新应当保留未提供的字段，实际 %+v", task)
		}
	})

	t.Run("什么都没传", func(t *testing.T) {
		rec := serve(handler, http.MethodPatch, "/api/tasks/1", `{}`, jsonHeaders())
		if rec.Code != http.StatusBadRequest {
			t.Errorf("状态码：期望 400，实际 %d", rec.Code)
		}
	})

	t.Run("删除后查不到", func(t *testing.T) {
		if rec := serve(handler, http.MethodDelete, "/api/tasks/1", "", nil); rec.Code != http.StatusNoContent {
			t.Fatalf("状态码：期望 204，实际 %d", rec.Code)
		}
		if rec := serve(handler, http.MethodDelete, "/api/tasks/1", "", nil); rec.Code != http.StatusNotFound {
			t.Errorf("重复删除应当返回 404，实际 %d", rec.Code)
		}
		if rec := serve(handler, http.MethodGet, "/api/tasks/1", "", nil); rec.Code != http.StatusNotFound {
			t.Errorf("删除后查询应当返回 404，实际 %d", rec.Code)
		}
	})
}

// TestBodyLimit 验证超过上限的请求体被拒绝。
func TestBodyLimit(t *testing.T) {
	api := newTestAPI()
	body := `{"title":"` + strings.Repeat("a", maxBodyBytes*2) + `"}`
	rec := serve(api.Handler(), http.MethodPost, "/api/tasks", body, jsonHeaders())

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码：期望 413，实际 %d（%s）", rec.Code, rec.Body.String())
	}
	if got := decodeError(t, rec).Code; got != "request_too_large" {
		t.Errorf("错误码：期望 request_too_large，实际 %q", got)
	}
}

// TestRequestIDMiddleware 验证请求 ID 的沿用与生成规则。
func TestRequestIDMiddleware(t *testing.T) {
	api := newTestAPI(WithRequestIDGenerator(func() string { return "generated-1" }))
	handler := api.Handler()

	t.Run("沿用调用方的 ID", func(t *testing.T) {
		rec := serve(handler, http.MethodGet, "/healthz", "", map[string]string{"X-Request-ID": "from-client"})
		if got := rec.Header().Get("X-Request-ID"); got != "from-client" {
			t.Errorf("响应头里的请求 ID：期望 from-client，实际 %q", got)
		}
	})

	t.Run("没有就生成", func(t *testing.T) {
		rec := serve(handler, http.MethodGet, "/healthz", "", nil)
		if got := rec.Header().Get("X-Request-ID"); got != "generated-1" {
			t.Errorf("响应头里的请求 ID：期望 generated-1，实际 %q", got)
		}
	})
}

// TestLoggingMiddleware 验证访问日志的字段。
func TestLoggingMiddleware(t *testing.T) {
	var logs bytes.Buffer
	api := NewAPI(newStoreWithClock(demoTime), log.New(&logs, "", 0),
		WithRequestIDGenerator(func() string { return "log-1" }))

	serve(api.Handler(), http.MethodGet, "/healthz", "", nil)

	line := logs.String()
	for _, want := range []string{"request_id=log-1", "method=GET", "path=/healthz", "status=200"} {
		if !strings.Contains(line, want) {
			t.Errorf("日志 %q 里找不到 %q", line, want)
		}
	}
}

// TestRecoverMiddleware 验证 panic 被转成 500 而不是掐断连接。
func TestRecoverMiddleware(t *testing.T) {
	var logs bytes.Buffer
	api := NewAPI(newStoreWithClock(demoTime), log.New(&logs, "", 0))
	api.Mux().HandleFunc("GET /panic", func(http.ResponseWriter, *http.Request) {
		panic("故意触发的 panic")
	})

	rec := serve(api.Handler(), http.MethodGet, "/panic", "", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("状态码：期望 500，实际 %d", rec.Code)
	}
	if got := decodeError(t, rec).Code; got != "internal" {
		t.Errorf("错误码：期望 internal，实际 %q", got)
	}
	if !strings.Contains(logs.String(), "panic recovered") {
		t.Errorf("应当记录 panic，实际日志 %q", logs.String())
	}
}

// TestCORSMiddleware 验证跨域响应头和预检。
func TestCORSMiddleware(t *testing.T) {
	api := newTestAPI(WithAllowedOrigin("https://example.com"))
	handler := api.Handler()

	t.Run("预检直接返回 204", func(t *testing.T) {
		rec := serve(handler, http.MethodOptions, "/api/tasks", "", map[string]string{
			"Origin":                        "https://example.com",
			"Access-Control-Request-Method": "POST",
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("状态码：期望 204，实际 %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
			t.Errorf("Allow-Origin：期望 https://example.com，实际 %q", got)
		}
		if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "POST") {
			t.Errorf("Allow-Methods 里应当包含 POST，实际 %q", rec.Header().Get("Access-Control-Allow-Methods"))
		}
	})

	t.Run("不匹配的来源不回显", func(t *testing.T) {
		rec := serve(handler, http.MethodGet, "/healthz", "", map[string]string{
			"Origin": "https://evil.example",
		})
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("不匹配的来源不应回显 Allow-Origin，实际 %q", got)
		}
	})
}

// TestRateLimitMiddleware 用注入的时钟验证令牌桶行为。
func TestRateLimitMiddleware(t *testing.T) {
	now := demoTime()
	limiter := newLimiterWithClock(1, 2, func() time.Time { return now })
	api := newTestAPI(WithLimiter(limiter))
	handler := api.Handler()

	for i := 1; i <= 2; i++ {
		if rec := serve(handler, http.MethodGet, "/healthz", "", nil); rec.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求应当通过，实际 %d", i, rec.Code)
		}
	}

	rec := serve(handler, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("第 3 次请求应当被限流，实际 %d", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Errorf("Retry-After：期望 1，实际 %q", got)
	}

	now = now.Add(time.Second)
	if rec := serve(handler, http.MethodGet, "/healthz", "", nil); rec.Code != http.StatusOK {
		t.Errorf("前进 1 秒后应当重新放行，实际 %d", rec.Code)
	}
}

// TestServerTimeouts 守住超时配置，避免有人把防护删掉。
func TestServerTimeouts(t *testing.T) {
	srv := NewServer(newTestAPI().Handler(), log.New(io.Discard, "", 0))

	timeouts := map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	}
	for name, value := range timeouts {
		if value <= 0 {
			t.Errorf("%s 必须大于 0，实际 %v", name, value)
		}
	}
}

// TestServeUntilCanceled 验证优雅关闭：请求能正常送达，取消后干净退出。
func TestServeUntilCanceled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}

	logger := log.New(io.Discard, "", 0)
	srv := NewServer(newTestAPI(WithRequestIDGenerator(func() string { return "shutdown-1" })).Handler(), logger)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ServeUntilCanceled(ctx, srv, ln, logger)
	}()

	base := "http://" + ln.Addr().String()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/healthz")
	if err != nil {
		t.Fatalf("关闭前请求失败: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("状态码：期望 200，实际 %d", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("优雅关闭应当成功，实际 %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("优雅关闭超时")
	}

	if _, err := client.Get(base + "/healthz"); err == nil {
		t.Error("关闭后不应再接受请求")
	}
}

// TestStoreConcurrentAccess 交给 -race 检查存储的并发安全。
func TestStoreConcurrentAccess(t *testing.T) {
	store := newStoreWithClock(demoTime)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				task := store.Create("并发任务")
				store.List(nil, 0)
				store.Get(task.ID)
				store.Update(task.ID, nil, nil)
				store.Delete(task.ID)
			}
		}(i)
	}
	wg.Wait()

	if got := len(store.List(nil, 0)); got != 0 {
		t.Errorf("所有任务都应被删除，实际剩 %d 个", got)
	}
}

// TestAPIErrorMessage 验证 APIError 作为 error 时的输出。
func TestAPIErrorMessage(t *testing.T) {
	err := APIError{Code: "not_found", Message: "任务不存在"}
	if got := err.Error(); got != "not_found: 任务不存在" {
		t.Errorf("Error()：期望 %q，实际 %q", "not_found: 任务不存在", got)
	}
}
