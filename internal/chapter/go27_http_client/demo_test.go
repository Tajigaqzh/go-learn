package go27_http_client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newTestAPI 启动假上游并返回客户端与清理函数。
func newTestAPI(t *testing.T) (*API, *upstream) {
	t.Helper()

	up := newUpstream()
	t.Cleanup(up.Close)
	return NewAPI(up.URL(), NewClient()), up
}

// TestUser 覆盖成功与 404 两条路径。
func TestUser(t *testing.T) {
	api, _ := newTestAPI(t)

	user, err := api.User(context.Background(), "7")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if user.ID != "7" || user.Name != "user-7" {
		t.Errorf("用户内容不对: %+v", user)
	}

	_, err = api.User(context.Background(), "404")
	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("404 应当返回 *StatusError，实际 %v", err)
	}
	if statusErr.StatusCode != http.StatusNotFound {
		t.Errorf("状态码：期望 404，实际 %d", statusErr.StatusCode)
	}
	if statusErr.Retryable() {
		t.Error("404 不应该被判为可重试")
	}
}

// TestStatusErrorRetryable 验证状态码与可重试性的对应关系。
func TestStatusErrorRetryable(t *testing.T) {
	cases := map[int]bool{
		400: false,
		404: false,
		409: false,
		429: true,
		500: true,
		502: true,
		503: true,
	}
	for code, want := range cases {
		err := &StatusError{StatusCode: code}
		if got := err.Retryable(); got != want {
			t.Errorf("状态码 %d：期望 Retryable=%t，实际 %t", code, want, got)
		}
	}
}

// TestRetryAfterParsing 覆盖 Retry-After 的两种写法。
func TestRetryAfterParsing(t *testing.T) {
	if got := parseRetryAfter("3"); got != 3*time.Second {
		t.Errorf("秒数写法：期望 3s，实际 %v", got)
	}
	if got := parseRetryAfter(""); got != 0 {
		t.Errorf("空值：期望 0，实际 %v", got)
	}
	if got := parseRetryAfter("不是数字也不是时间"); got != 0 {
		t.Errorf("非法值：期望 0，实际 %v", got)
	}

	at := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(at); got <= 0 || got > 3*time.Second {
		t.Errorf("HTTP 日期写法：期望 0 < d <= 3s，实际 %v", got)
	}
}

// TestEcho 验证 JSON 请求体与自定义请求头都被发出去了。
func TestEcho(t *testing.T) {
	api, _ := newTestAPI(t)

	echo, err := api.Echo(context.Background(), EchoRequest{Message: "hi", Count: 3}, map[string]string{
		"X-Request-ID": "req-9",
		"User-Agent":   "go-learn-test/1.0",
	})
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if echo.Message != "hi" || echo.Count != 3 {
		t.Errorf("回显内容不对: %+v", echo)
	}
	if echo.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type：期望 application/json，实际 %q", echo.Headers["Content-Type"])
	}
	if echo.Headers["X-Request-ID"] != "req-9" || echo.Headers["User-Agent"] != "go-learn-test/1.0" {
		t.Errorf("自定义请求头没有传到上游: %+v", echo.Headers)
	}
}

// TestSubmitForm 验证表单编码与 Content-Type。
func TestSubmitForm(t *testing.T) {
	api, up := newTestAPI(t)

	values, err := api.SubmitForm(context.Background(), url.Values{
		"name": {"gopher"},
		"tag":  {"go", "http"},
	})
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if got := strings.Join(values["tag"], ","); got != "go,http" {
		t.Errorf("表单字段 tag：期望 go,http，实际 %q", got)
	}
	if got := up.hitsFor("/api/form"); got != 1 {
		t.Errorf("上游应当只收到 1 次表单请求，实际 %d", got)
	}
}

// TestGzipBehaviour 验证自动解压与手动 Accept-Encoding 的区别。
func TestGzipBehaviour(t *testing.T) {
	api, _ := newTestAPI(t)

	t.Run("自动解压", func(t *testing.T) {
		body, header, err := api.RawGet(context.Background(), "/api/gzip", nil)
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		if header.Get("Content-Encoding") != "" {
			t.Errorf("自动解压时 Content-Encoding 应当被移除，实际 %q", header.Get("Content-Encoding"))
		}
		if !strings.Contains(string(body), `"status":"ok"`) {
			t.Errorf("应当拿到解压后的 JSON，实际 %q", body)
		}
	})

	t.Run("手动指定后不自动解压", func(t *testing.T) {
		body, header, err := api.RawGet(context.Background(), "/api/gzip",
			map[string]string{"Accept-Encoding": "gzip"})
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		if header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Content-Encoding 应当是 gzip，实际 %q", header.Get("Content-Encoding"))
		}
		if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
			t.Errorf("响应体应当是 gzip 字节流，实际前两字节 %x", body[:min(2, len(body))])
		}
	})
}

// TestConnectionReuse 验证连接池确实复用连接，关掉 keep-alive 后不再复用。
func TestConnectionReuse(t *testing.T) {
	up := newUpstream()
	defer up.Close()

	keepAlive := &countingTransport{base: NewTransport()}
	api := NewAPI(up.URL(), &http.Client{Timeout: 3 * time.Second, Transport: keepAlive})
	for i := 0; i < 3; i++ {
		if _, err := api.User(context.Background(), "1"); err != nil {
			t.Fatalf("请求失败: %v", err)
		}
	}
	newConns, reused := keepAlive.stats()
	if newConns != 1 || reused != 2 {
		t.Errorf("期望 1 条连接复用 2 次，实际新建 %d 条、复用 %d 次", newConns, reused)
	}

	transport := NewTransport()
	transport.DisableKeepAlives = true
	noKeepAlive := &countingTransport{base: transport}
	api = NewAPI(up.URL(), &http.Client{Timeout: 3 * time.Second, Transport: noKeepAlive})
	for i := 0; i < 3; i++ {
		if _, err := api.User(context.Background(), "1"); err != nil {
			t.Fatalf("请求失败: %v", err)
		}
	}
	newConns, reused = noKeepAlive.stats()
	if newConns != 3 || reused != 0 {
		t.Errorf("关掉 keep-alive 后期望 3 条新连接，实际新建 %d 条、复用 %d 次", newConns, reused)
	}
}

// TestBodyCloseAffectsReuse 验证大响应体不读完会丢连接，小响应体被排空后仍可复用。
func TestBodyCloseAffectsReuse(t *testing.T) {
	up := newUpstream()
	defer up.Close()

	t.Run("读完再关闭", func(t *testing.T) {
		newConns, reused := measureReuse(up.URL(), 64, 64)
		if newConns != 1 || reused != 1 {
			t.Errorf("期望新建 1 条、复用 1 次，实际新建 %d 条、复用 %d 次", newConns, reused)
		}
	})

	t.Run("大响应体只读 1 字节", func(t *testing.T) {
		newConns, reused := measureReuse(up.URL(), 1<<20, 1)
		if newConns != 2 || reused != 0 {
			t.Errorf("期望新建 2 条、复用 0 次，实际新建 %d 条、复用 %d 次", newConns, reused)
		}
	})

	t.Run("小响应体只读 1 字节", func(t *testing.T) {
		newConns, reused := measureReuse(up.URL(), 64<<10, 1)
		if newConns != 1 || reused != 1 {
			t.Errorf("小响应体会被排空，期望新建 1 条、复用 1 次，实际新建 %d 条、复用 %d 次", newConns, reused)
		}
	})
}

// TestTimeoutClassification 验证三种超时/取消的错误特征。
func TestTimeoutClassification(t *testing.T) {
	up := newUpstream()
	defer up.Close()

	t.Run("Client.Timeout", func(t *testing.T) {
		client := NewClient()
		client.Timeout = 100 * time.Millisecond
		api := NewAPI(up.URL(), client)

		err := api.Ping(context.Background(), 300)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("期望包装 context.DeadlineExceeded，实际 %v", err)
		}
		if !isTimeout(err) {
			t.Error("期望 *url.Error.Timeout() 为 true")
		}
	})

	t.Run("context 超时", func(t *testing.T) {
		api := NewAPI(up.URL(), NewClient())
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := api.Ping(ctx, 300)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("期望 context.DeadlineExceeded，实际 %v", err)
		}
		if Retryable(err) {
			t.Error("上下文超时不该被重试")
		}
	})

	t.Run("主动取消", func(t *testing.T) {
		api := NewAPI(up.URL(), NewClient())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := api.Ping(ctx, 0)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("期望 context.Canceled，实际 %v", err)
		}
		if Retryable(err) {
			t.Error("主动取消不该被重试")
		}
	})
}

// TestRetryPolicy 覆盖重试次数、退避节奏、Retry-After 与取消。
func TestRetryPolicy(t *testing.T) {
	api, up := newTestAPI(t)

	var delays []time.Duration
	policy := NewRetryPolicy().WithSleep(func(_ context.Context, d time.Duration) error {
		delays = append(delays, d)
		return nil
	}).WithJitter(nil)

	t.Run("指数退避", func(t *testing.T) {
		delays = nil
		up.setFlaky(2)
		attempts, err := policy.Do(context.Background(), api.Flaky)
		if err != nil {
			t.Fatalf("重试后应当成功，实际 %v", err)
		}
		if attempts != 3 {
			t.Errorf("尝试次数：期望 3，实际 %d", attempts)
		}
		if len(delays) != 2 || delays[0] != 10*time.Millisecond || delays[1] != 20*time.Millisecond {
			t.Errorf("退避节奏不对: %v", delays)
		}
	})

	t.Run("不可重试的状态码不重试", func(t *testing.T) {
		delays = nil
		attempts, err := policy.Do(context.Background(), func(ctx context.Context) error {
			return api.ProbeStatus(ctx, http.StatusBadRequest)
		})
		if attempts != 1 {
			t.Errorf("尝试次数：期望 1，实际 %d", attempts)
		}
		if len(delays) != 0 {
			t.Errorf("不该有退避，实际 %v", delays)
		}
		var statusErr *StatusError
		if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadRequest {
			t.Errorf("期望拿到 400 的 StatusError，实际 %v", err)
		}
	})

	t.Run("尊重 Retry-After", func(t *testing.T) {
		delays = nil
		up.setFlakyAfter(1, "1")
		attempts, err := policy.Do(context.Background(), api.Flaky)
		if err != nil {
			t.Fatalf("重试后应当成功，实际 %v", err)
		}
		if attempts != 2 {
			t.Errorf("尝试次数：期望 2，实际 %d", attempts)
		}
		if len(delays) != 1 || delays[0] != time.Second {
			t.Errorf("应当按 Retry-After 等 1s，实际 %v", delays)
		}
	})

	t.Run("退避期间取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		up.setFlaky(3)

		attempts, err := policy.WithSleep(func(ctx context.Context, _ time.Duration) error {
			cancel()
			return ctx.Err()
		}).Do(ctx, api.Flaky)
		if attempts != 1 {
			t.Errorf("尝试次数：期望 1，实际 %d", attempts)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("期望 context.Canceled，实际 %v", err)
		}
	})
}

// TestRetryDelayCap 验证退避不会超过 MaxDelay。
func TestRetryDelayCap(t *testing.T) {
	policy := NewRetryPolicy().WithJitter(nil)

	if got := policy.delay(1, 0); got != 10*time.Millisecond {
		t.Errorf("第 1 次退避：期望 10ms，实际 %v", got)
	}
	if got := policy.delay(3, 0); got != 40*time.Millisecond {
		t.Errorf("第 3 次退避：期望 40ms，实际 %v", got)
	}
	if got := policy.delay(20, 0); got != policy.MaxDelay {
		t.Errorf("退避应当被封顶到 %v，实际 %v", policy.MaxDelay, got)
	}
	if got := policy.delay(1, time.Hour); got != policy.MaxDelay {
		t.Errorf("Retry-After 超过上限时应当封顶到 %v，实际 %v", policy.MaxDelay, got)
	}
}

// TestUploadFile 验证 multipart 上传的内容与摘要。
func TestUploadFile(t *testing.T) {
	api, _ := newTestAPI(t)
	content := bytes.Repeat([]byte("go-learn\n"), 64)

	result, err := api.UploadFile(context.Background(), "file", "notes.txt", content)
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if result.Name != "notes.txt" || result.Size != len(content) {
		t.Errorf("上传结果不对: %+v", result)
	}
	sum := sha256.Sum256(content)
	if result.SHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("上游算出的摘要与本地不一致: %s", result.SHA256)
	}
}

// TestDownloadTo 验证流式下载的字节数与摘要。
func TestDownloadTo(t *testing.T) {
	api, _ := newTestAPI(t)

	var sink bytes.Buffer
	written, sum, err := api.DownloadTo(context.Background(), &sink, "/api/download?bytes=8192")
	if err != nil {
		t.Fatalf("下载失败: %v", err)
	}
	if written != 8192 || sink.Len() != 8192 {
		t.Errorf("字节数不对：写入 %d，缓冲区 %d", written, sink.Len())
	}
	expected := sha256.Sum256(sink.Bytes())
	if sum != hex.EncodeToString(expected[:]) {
		t.Errorf("摘要对不上: %s", sum)
	}
}

// TestTLSClient 验证用 httptest 的客户端访问自签 HTTPS，而默认客户端会被拒绝。
func TestTLSClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"scheme": "https"})
	}))
	defer server.Close()
	server.Config.ErrorLog = log.New(io.Discard, "", 0)

	resp, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("用自带客户端请求应当成功: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("状态码：期望 200，实际 %d", resp.StatusCode)
	}

	_, err = NewClient().Get(server.URL)
	var certErr *tls.CertificateVerificationError
	if !errors.As(err, &certErr) {
		t.Errorf("默认客户端应当因证书校验失败，实际 %v", err)
	}
}

// TestRedirectPolicy 验证默认跟随重定向与 CheckRedirect 停止跟随。
func TestRedirectPolicy(t *testing.T) {
	up := newUpstream()
	defer up.Close()

	t.Run("默认跟随", func(t *testing.T) {
		api := NewAPI(up.URL(), NewClient())
		resp, err := api.HTTPClient().Get(up.URL() + "/api/redirect")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("状态码：期望 200，实际 %d", resp.StatusCode)
		}
		if resp.Request.URL.Path != "/api/users/1" {
			t.Errorf("最终路径：期望 /api/users/1，实际 %s", resp.Request.URL.Path)
		}
	})

	t.Run("用 ErrUseLastResponse 停止", func(t *testing.T) {
		client := NewClient()
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
		api := NewAPI(up.URL(), client)

		resp, err := api.HTTPClient().Get(up.URL() + "/api/redirect")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusFound {
			t.Errorf("状态码：期望 302，实际 %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/api/users/1" {
			t.Errorf("Location：期望 /api/users/1，实际 %q", got)
		}
	})
}

// TestParseStatusErrorBody 验证错误体被截断保留，且不会把整个错误页读进内存。
func TestParseStatusErrorBody(t *testing.T) {
	api, _ := newTestAPI(t)

	err := api.ProbeStatus(context.Background(), http.StatusInternalServerError)
	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("期望 *StatusError，实际 %v", err)
	}
	if !statusErr.Retryable() {
		t.Error("500 应当可重试")
	}
	if len(statusErr.Body) > 512 {
		t.Errorf("错误体应当被截断，实际长度 %d", len(statusErr.Body))
	}
}
