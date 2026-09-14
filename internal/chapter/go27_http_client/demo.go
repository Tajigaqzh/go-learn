// Package go27_http_client 演示 Go 的 HTTP 客户端与调用外部服务。
//
// 本章覆盖：
//   - http.Client 超时与 context 的关系
//   - Transport 与连接复用
//   - 构造请求：JSON、表单、自定义请求头
//   - 处理响应：Body 的正确关闭方式、状态码、gzip、限流读取
//   - 重试与退避（含 Retry-After 与上下文取消）
//   - 文件上传与流式下载
//   - 用 httptest 测外部依赖、TLS 与重定向策略
//
// 所有演示都打一个用 httptest 搭起来的假上游，不依赖外网。时钟、退避和
// 随机抖动都可以注入，因此除了临时端口，输出是可复现的。
package go27_http_client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// Demo 运行第 27 章所有演示。
func Demo() {
	fmt.Println("========== go27_http_client: HTTP 客户端与外部服务 ==========")

	fmt.Println("\n--- 27.1 http.Client 超时与 context ---")
	demoTimeouts()

	fmt.Println("\n--- 27.2 Transport 与连接复用 ---")
	demoConnectionReuse()

	fmt.Println("\n--- 27.3 构造请求：JSON、表单与请求头 ---")
	demoBuildRequest()

	fmt.Println("\n--- 27.4 处理响应：状态码、gzip 与 Body 关闭 ---")
	demoResponseHandling()

	fmt.Println("\n--- 27.5 重试与退避 ---")
	demoRetry()

	fmt.Println("\n--- 27.6 上传与流式下载 ---")
	demoTransfer()

	fmt.Println("\n--- 27.7 测试外部依赖、TLS 与重定向 ---")
	demoTestingAndTLS()

	fmt.Println("\n========== HTTP 客户端与外部服务演示结束 ==========")
}

// --- 27.1 http.Client 超时与 context ---

// demoTimeouts 对比 Client.Timeout、context 超时与主动取消三种情况。
func demoTimeouts() {
	up := newUpstream()
	defer up.Close()
	base := up.URL()

	fmt.Println("① Client.Timeout=100ms，上游需要 300ms:")
	quickClient := NewClient()
	quickClient.Timeout = 100 * time.Millisecond
	quick := NewAPI(base, quickClient)

	err := quick.Ping(context.Background(), 300)
	fmt.Printf("   %s\n", normalize(err, base))
	fmt.Printf("   *url.Error.Timeout() = %t，errors.Is(err, context.DeadlineExceeded) = %t\n",
		isTimeout(err), errors.Is(err, context.DeadlineExceeded))

	fmt.Println("② 用 context.WithTimeout(50ms) 控制同一次调用:")
	api := NewAPI(base, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = api.Ping(ctx, 300)
	fmt.Printf("   %s\n", normalize(err, base))
	fmt.Printf("   errors.Is(err, context.DeadlineExceeded) = %t，Retryable(err) = %t\n",
		errors.Is(err, context.DeadlineExceeded), Retryable(err))

	fmt.Println("③ 调用方主动取消，请求立刻结束:")
	cancelled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	err = api.Ping(cancelled, 0)
	fmt.Printf("   %s\n", normalize(err, base))
	fmt.Printf("   errors.Is(err, context.Canceled) = %t，Retryable(err) = %t\n",
		errors.Is(err, context.Canceled), Retryable(err))

	fmt.Println("④ 预算充足时正常返回:")
	fmt.Printf("   Ping(0ms) = %v\n", api.Ping(context.Background(), 0))
}

// --- 27.2 Transport 与连接复用 ---

// demoConnectionReuse 用 httptrace 统计连接是新建的还是复用的。
func demoConnectionReuse() {
	up := newUpstream()
	defer up.Close()

	keepAlive := &countingTransport{base: NewTransport()}
	api := NewAPI(up.URL(), &http.Client{Timeout: 3 * time.Second, Transport: keepAlive})
	for i := 0; i < 3; i++ {
		if _, err := api.User(context.Background(), "1"); err != nil {
			fmt.Printf("请求失败: %v\n", err)
			return
		}
	}
	newConns, reused := keepAlive.stats()
	fmt.Printf("保留空闲连接: 3 次请求新建 %d 条连接，复用 %d 次\n", newConns, reused)

	noKeepAlive := NewTransport()
	noKeepAlive.DisableKeepAlives = true
	counter := &countingTransport{base: noKeepAlive}
	api = NewAPI(up.URL(), &http.Client{Timeout: 3 * time.Second, Transport: counter})
	for i := 0; i < 3; i++ {
		if _, err := api.User(context.Background(), "1"); err != nil {
			fmt.Printf("请求失败: %v\n", err)
			return
		}
	}
	newConns, reused = counter.stats()
	fmt.Printf("DisableKeepAlives: 3 次请求新建 %d 条连接，复用 %d 次\n", newConns, reused)

	transport := NewTransport()
	fmt.Printf("连接池配置: MaxIdleConns=%d MaxIdleConnsPerHost=%d IdleConnTimeout=%v ResponseHeaderTimeout=%v\n",
		transport.MaxIdleConns, transport.MaxIdleConnsPerHost,
		transport.IdleConnTimeout, transport.ResponseHeaderTimeout)
}

// --- 27.3 构造请求：JSON、表单与请求头 ---

// demoBuildRequest 演示 JSON、表单与自定义请求头。
func demoBuildRequest() {
	up := newUpstream()
	defer up.Close()
	api := NewAPI(up.URL(), nil)
	ctx := context.Background()

	echo, err := api.Echo(ctx, EchoRequest{Message: "hello", Count: 2}, map[string]string{
		"X-Request-ID": "req-1",
		"User-Agent":   "go-learn/1.0",
	})
	if err != nil {
		fmt.Printf("JSON 请求失败: %v\n", err)
		return
	}
	fmt.Printf("JSON 回显: message=%s count=%d\n", echo.Message, echo.Count)
	fmt.Printf("上游看到的头: Content-Type=%s User-Agent=%s X-Request-ID=%s\n",
		echo.Headers["Content-Type"], echo.Headers["User-Agent"], echo.Headers["X-Request-ID"])

	values, err := api.SubmitForm(ctx, url.Values{
		"name": {"gopher"},
		"tag":  {"go", "http"},
	})
	if err != nil {
		fmt.Printf("表单请求失败: %v\n", err)
		return
	}
	fmt.Printf("表单回显: %s\n", formatValues(values))
	fmt.Printf("表单请求次数（上游侧统计）: /api/form = %d\n", up.hitsFor("/api/form"))
}

// --- 27.4 处理响应：状态码、gzip 与 Body 关闭 ---

// demoResponseHandling 覆盖状态码判定、gzip 与响应体关闭对连接复用的影响。
func demoResponseHandling() {
	up := newUpstream()
	defer up.Close()
	base := up.URL()
	api := NewAPI(base, nil)
	ctx := context.Background()

	err := api.ProbeStatus(ctx, 404)
	var statusErr *StatusError
	fmt.Printf("404 响应: %s\n", normalize(err, base))
	if errors.As(err, &statusErr) {
		fmt.Printf("errors.As 拿到 *StatusError: 状态码=%d 可重试=%t\n",
			statusErr.StatusCode, statusErr.Retryable())
	}

	up.setFlakyAfter(1, "1")
	err = api.Flaky(ctx)
	fmt.Printf("503 响应: %s\n", normalize(err, base))
	if errors.As(err, &statusErr) {
		fmt.Printf("解析 Retry-After: 等待 %v，可重试=%t\n", statusErr.RetryAfter, statusErr.Retryable())
	}

	body, header, err := api.RawGet(ctx, "/api/gzip", nil)
	if err != nil {
		fmt.Printf("gzip 请求失败: %v\n", err)
		return
	}
	fmt.Printf("自动解压: Content-Encoding=%q 响应体=%s\n",
		header.Get("Content-Encoding"), strings.TrimSpace(string(body)))

	body, header, err = api.RawGet(ctx, "/api/gzip", map[string]string{"Accept-Encoding": "gzip"})
	if err != nil {
		fmt.Printf("gzip 请求失败: %v\n", err)
		return
	}
	fmt.Printf("手动指定 Accept-Encoding: Content-Encoding=%q 前两个字节=%#x %#x（gzip 魔数，未解压）\n",
		header.Get("Content-Encoding"), body[0], body[1])

	fmt.Println("响应体关闭方式对连接复用的影响:")
	newConns, reused := measureReuse(base, 64, 64) // 读完 64 字节
	fmt.Printf("  读完再关闭: 2 次请求新建 %d 条连接，复用 %d 次\n", newConns, reused)

	newConns, reused = measureReuse(base, 1<<20, 1) // 大响应体只读 1 字节
	fmt.Printf("  只读 1 字节就关闭（1 MB）: 2 次请求新建 %d 条连接，复用 %d 次\n", newConns, reused)

	newConns, reused = measureReuse(base, 64<<10, 1) // 小响应体只读 1 字节
	fmt.Printf("  只读 1 字节就关闭（64 KB）: 2 次请求新建 %d 条连接，复用 %d 次（小响应体会被排空）\n",
		newConns, reused)
}

// --- 27.5 重试与退避 ---

// demoRetry 演示可重试判定、指数退避、Retry-After 与退避期间取消。
func demoRetry() {
	up := newUpstream()
	defer up.Close()
	api := NewAPI(up.URL(), nil)

	var delays []time.Duration
	record := func(_ context.Context, d time.Duration) error {
		delays = append(delays, d)
		return nil
	}
	policy := NewRetryPolicy().WithSleep(record).WithJitter(nil)

	up.setFlaky(2)
	delays = nil
	attempts, err := policy.Do(context.Background(), func(ctx context.Context) error {
		return api.Flaky(ctx)
	})
	fmt.Printf("上游先失败 2 次: 尝试 %d 次，退避 %v，最终错误=%v\n", attempts, delays, err)

	delays = nil
	attempts, err = policy.Do(context.Background(), func(ctx context.Context) error {
		return api.ProbeStatus(ctx, 400)
	})
	fmt.Printf("400 不可重试: 尝试 %d 次，退避 %v，最终错误=%s\n", attempts, delays, normalize(err, up.URL()))

	up.setFlakyAfter(1, "1")
	delays = nil
	attempts, err = policy.Do(context.Background(), func(ctx context.Context) error {
		return api.Flaky(ctx)
	})
	fmt.Printf("503 带 Retry-After=1 秒: 尝试 %d 次，退避 %v，最终错误=%v\n", attempts, delays, err)

	// 退避期间上下文被取消：重试立即停止，返回取消原因。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	up.setFlaky(3)
	delays = nil
	attempts, err = policy.WithSleep(func(ctx context.Context, d time.Duration) error {
		delays = append(delays, d)
		cancel()
		return ctx.Err()
	}).Do(ctx, func(ctx context.Context) error {
		return api.Flaky(ctx)
	})
	fmt.Printf("退避期间取消: 尝试 %d 次，最终错误=%v（errors.Is(context.Canceled)=%t）\n",
		attempts, err, errors.Is(err, context.Canceled))
}

// --- 27.6 上传与流式下载 ---

// demoTransfer 演示 multipart 上传与流式下载。
func demoTransfer() {
	up := newUpstream()
	defer up.Close()
	base := up.URL()
	api := NewAPI(base, nil)
	ctx := context.Background()

	content := bytes.Repeat([]byte("go-learn\n"), 128)
	result, err := api.UploadFile(ctx, "file", "notes.txt", content)
	if err != nil {
		fmt.Printf("上传失败: %v\n", err)
		return
	}
	local := sha256.Sum256(content)
	fmt.Printf("上传 %d 字节: 文件名=%s 大小=%d 上游算出 sha256=%s…\n",
		len(content), result.Name, result.Size, result.SHA256[:12])
	fmt.Printf("本地 sha256 与上游一致: %t\n",
		hex.EncodeToString(local[:])[:12] == result.SHA256[:12])

	var sink bytes.Buffer
	written, sum, err := api.DownloadTo(ctx, &sink, "/api/download?bytes=8192")
	if err != nil {
		fmt.Printf("下载失败: %v\n", err)
		return
	}
	fmt.Printf("流式下载: 写入 %d 字节，sha256=%s…（缓冲区里就是最终内容）\n", written, sum[:12])

	resp, err := api.HTTPClient().Get(base + "/api/download?bytes=8192")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	head, err := io.ReadAll(io.LimitReader(resp.Body, 100))
	resp.Body.Close()
	if err != nil {
		fmt.Printf("读取失败: %v\n", err)
		return
	}
	fmt.Printf("只要前 100 字节: 读到 %d 字节就关闭，剩下的不回内存\n", len(head))
}

// --- 27.7 测试外部依赖、TLS 与重定向 ---

// demoTestingAndTLS 演示自签 TLS、重定向策略与请求次数断言。
func demoTestingAndTLS() {
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"scheme": "https"})
	}))
	defer tlsServer.Close()
	// 默认客户端连它会握手失败，把服务端日志丢掉，避免演示输出里混进
	// 带时间戳和端口的 TLS 报错。
	tlsServer.Config.ErrorLog = log.New(io.Discard, "", 0)

	resp, err := tlsServer.Client().Get(tlsServer.URL)
	if err != nil {
		fmt.Printf("TLS 请求失败: %v\n", err)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	resp.Body.Close()
	fmt.Printf("用 httptest 的客户端访问自签 HTTPS: %d %s", resp.StatusCode, body)

	_, err = NewClient().Get(tlsServer.URL)
	var certErr *tls.CertificateVerificationError
	fmt.Printf("换成默认客户端: 失败=%t，证书校验错误=%t\n",
		err != nil, errors.As(err, &certErr))

	up := newUpstream()
	defer up.Close()
	base := up.URL()
	api := NewAPI(base, nil)

	resp, err = api.HTTPClient().Get(base + "/api/redirect")
	if err != nil {
		fmt.Printf("重定向请求失败: %v\n", err)
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	fmt.Printf("默认跟随重定向: 状态=%d 最终路径=%s\n", resp.StatusCode, resp.Request.URL.Path)

	noFollow := NewClient()
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	api = NewAPI(base, noFollow)
	resp, err = api.HTTPClient().Get(base + "/api/redirect")
	if err != nil {
		fmt.Printf("重定向请求失败: %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Printf("CheckRedirect=ErrUseLastResponse: 状态=%d Location=%s\n",
		resp.StatusCode, resp.Header.Get("Location"))

	fmt.Printf("上游侧统计: /api/users/1 被请求 %d 次（测试里就能这样断言调用次数）\n",
		up.hitsFor("/api/users/1"))
}

// --- 演示辅助 ---

// countingTransport 用 httptrace 统计新建连接与复用连接次数。
type countingTransport struct {
	base http.RoundTripper

	mu       sync.Mutex
	newConns int
	reused   int
}

// RoundTrip 在请求上挂一个 httptrace 钩子，再交给底层 Transport。
func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			t.mu.Lock()
			defer t.mu.Unlock()
			if info.Reused {
				t.reused++
				return
			}
			t.newConns++
		},
	}
	return t.base.RoundTrip(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
}

// stats 返回新建与复用的连接数。
func (t *countingTransport) stats() (int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.newConns, t.reused
}

// measureReuse 发两次请求测量连接复用：第一次从 /api/download 取 size 字节的
// 响应体、只读 readBytes 字节就关闭，第二次请求另一个接口，统计连接的新建与复用。
func measureReuse(base string, size, readBytes int) (int, int) {
	counter := &countingTransport{base: NewTransport()}
	client := &http.Client{Timeout: 5 * time.Second, Transport: counter}

	resp, err := client.Get(fmt.Sprintf("%s/api/download?bytes=%d", base, size))
	if err == nil {
		if readBytes > 0 {
			_, _ = io.ReadFull(resp.Body, make([]byte, readBytes))
		}
		resp.Body.Close()
	}

	resp, err = client.Get(base + "/api/users/1")
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return counter.stats()
}

// normalize 把临时端口替换成占位符，让正文里的报错输出可以逐字复现。
func normalize(err error, base string) string {
	if err == nil {
		return "<nil>"
	}
	return strings.ReplaceAll(err.Error(), base, "<upstream>")
}

// isTimeout 判断错误是否属于「客户端超时」。
func isTimeout(err error) bool {
	var urlErr *url.Error
	return errors.As(err, &urlErr) && urlErr.Timeout()
}

// formatValues 把表单值按 key 排序后拼成一行，保证输出稳定。
func formatValues(values map[string][]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.Join(values[key], ","))
	}
	return strings.Join(parts, " ")
}
