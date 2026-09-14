package go27_http_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxResponseBytes 限制单次响应读取量，避免异常大的响应把内存吃满。
const maxResponseBytes = 1 << 20

// StatusError 表示上游返回了非 2xx 状态码。
type StatusError struct {
	StatusCode int
	Body       string
	RetryAfter time.Duration
}

// Error 实现 error 接口。
func (e *StatusError) Error() string {
	return fmt.Sprintf("上游返回 %d: %s", e.StatusCode, strings.TrimSpace(e.Body))
}

// Retryable 判断这个状态码是否值得重试：429 与 5xx 属于「可能恢复」。
func (e *StatusError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// NewClient 返回配置好超时与连接池的 http.Client。
//
// 直接用 http.DefaultClient 的两个常见问题：没有超时（请求可能永远挂住）、
// MaxIdleConnsPerHost 只有 2（并发一高就频繁重建连接）。
func NewClient() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: NewTransport(),
	}
}

// NewTransport 返回带连接池参数的 Transport。
func NewTransport() *http.Transport {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{}
	}

	transport := base.Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 10                // 默认 2
	transport.IdleConnTimeout = 90 * time.Second      // 空闲连接多久回收
	transport.ResponseHeaderTimeout = 5 * time.Second // 只等响应头
	transport.TLSHandshakeTimeout = 5 * time.Second   // TLS 握手超时
	return transport
}

// API 是访问上游的客户端，携带基地址与 http.Client。
type API struct {
	base   string
	client *http.Client
}

// NewAPI 创建客户端；client 为 nil 时使用 NewClient()。
func NewAPI(base string, client *http.Client) *API {
	if client == nil {
		client = NewClient()
	}
	return &API{base: strings.TrimRight(base, "/"), client: client}
}

// HTTPClient 暴露底层 client，便于测试替换 Transport。
func (a *API) HTTPClient() *http.Client {
	return a.client
}

// User 拉取用户信息。
func (a *API) User(ctx context.Context, id string) (User, error) {
	resp, err := a.do(ctx, http.MethodGet, "/api/users/"+url.PathEscape(id), nil, "", nil)
	if err != nil {
		return User{}, err
	}
	if err := checkStatus(resp); err != nil {
		return User{}, err
	}

	var user User
	if err := decodeJSON(resp, &user); err != nil {
		return User{}, fmt.Errorf("解析用户响应: %w", err)
	}
	return user, nil
}

// Echo 发送 JSON 并回读上游看到的请求信息。
func (a *API) Echo(ctx context.Context, in EchoRequest, headers map[string]string) (EchoResponse, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return EchoResponse{}, fmt.Errorf("编码请求体: %w", err)
	}

	resp, err := a.do(ctx, http.MethodPost, "/api/echo", bytes.NewReader(payload), "application/json", headers)
	if err != nil {
		return EchoResponse{}, err
	}
	if err := checkStatus(resp); err != nil {
		return EchoResponse{}, err
	}

	var out EchoResponse
	if err := decodeJSON(resp, &out); err != nil {
		return EchoResponse{}, fmt.Errorf("解析回显响应: %w", err)
	}
	return out, nil
}

// SubmitForm 以 application/x-www-form-urlencoded 提交表单。
func (a *API) SubmitForm(ctx context.Context, values url.Values) (map[string][]string, error) {
	resp, err := a.do(ctx, http.MethodPost, "/api/form",
		strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var payload struct {
		Values map[string][]string `json:"values"`
	}
	if err := decodeJSON(resp, &payload); err != nil {
		return nil, fmt.Errorf("解析表单响应: %w", err)
	}
	return payload.Values, nil
}

// ProbeStatus 请求指定状态码，返回对应的错误（2xx 时为 nil）。
func (a *API) ProbeStatus(ctx context.Context, code int) error {
	resp, err := a.do(ctx, http.MethodGet, "/api/status/"+strconv.Itoa(code), nil, "", nil)
	if err != nil {
		return err
	}
	if err := checkStatus(resp); err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Ping 请求 /api/slow?ms=，用来演示超时与取消。
func (a *API) Ping(ctx context.Context, ms int) error {
	resp, err := a.do(ctx, http.MethodGet, "/api/slow?ms="+strconv.Itoa(ms), nil, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// Flaky 请求 /api/flaky，用来演示重试。
func (a *API) Flaky(ctx context.Context) error {
	resp, err := a.do(ctx, http.MethodGet, "/api/flaky", nil, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// RawGet 返回响应体与响应头，用于演示 gzip 解压、Body 关闭这类底层行为。
func (a *API) RawGet(ctx context.Context, path string, headers map[string]string) ([]byte, http.Header, error) {
	resp, err := a.do(ctx, http.MethodGet, path, nil, "", headers)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("读取响应体: %w", err)
	}
	return body, resp.Header, nil
}

// do 构造并发送请求，调用方负责关闭响应体。
func (a *API) do(ctx context.Context, method, path string, body io.Reader, contentType string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, body)
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return a.client.Do(req)
}

// checkStatus 把非 2xx 转成 *StatusError，并关闭响应体。
func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	defer resp.Body.Close()

	// 只读一小段错误信息：错误页可能很大，没必要全部读进内存。
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return &StatusError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
}

// decodeJSON 解析 JSON 响应，并保证响应体被关闭。
func decodeJSON(resp *http.Response, dst any) error {
	defer resp.Body.Close()
	return json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(dst)
}

// parseRetryAfter 解析 Retry-After：既支持秒数，也支持 HTTP 日期。
func parseRetryAfter(raw string) time.Duration {
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(raw); err == nil {
		if wait := time.Until(at); wait > 0 {
			return wait
		}
	}
	return 0
}

// Retryable 判断错误是否值得重试。
//
// 规则：上下文取消和超时是调用方自己的决定，不重试；HTTP 状态错误按状态码判断；
// 其余 url.Error（连接被拒、连接被重置、EOF 等）属于网络抖动，可以重试。
func Retryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.Retryable()
	}

	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

// RetryAfter 取出上游要求的等待时间，没有则返回 0。
func RetryAfter(err error) time.Duration {
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.RetryAfter
	}
	return 0
}
