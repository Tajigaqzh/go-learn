package go27_http_client

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"time"
)

// upstream 是用 httptest 搭出来的假外部服务。
//
// 本章所有演示和测试都打它：端口由内核分配，不依赖外网，也不会因为第三方
// 服务抖动而让结果变得不可复现。
type upstream struct {
	server          *httptest.Server
	mu              sync.Mutex
	flaky           int
	flakyRetryAfter string
	hits            map[string]int
}

// newUpstream 启动假上游。
func newUpstream() *upstream {
	u := &upstream{hits: make(map[string]int)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users/{id}", u.handleUser)
	mux.HandleFunc("POST /api/echo", u.handleEcho)
	mux.HandleFunc("POST /api/form", u.handleForm)
	mux.HandleFunc("POST /api/upload", u.handleUpload)
	mux.HandleFunc("GET /api/download", u.handleDownload)
	mux.HandleFunc("GET /api/flaky", u.handleFlaky)
	mux.HandleFunc("GET /api/gzip", u.handleGzip)
	mux.HandleFunc("GET /api/slow", u.handleSlow)
	mux.HandleFunc("GET /api/redirect", u.handleRedirect)
	mux.HandleFunc("GET /api/loop", u.handleLoop)
	mux.HandleFunc("GET /api/status/{code}", u.handleStatus)

	u.server = httptest.NewServer(mux)
	return u
}

// URL 返回假上游的基地址（端口每次运行都不同）。
func (u *upstream) URL() string {
	return u.server.URL
}

// Close 关闭假上游。
func (u *upstream) Close() {
	u.server.Close()
}

// setFlaky 设置 /api/flaky 还要失败几次。
func (u *upstream) setFlaky(failures int) {
	u.setFlakyAfter(failures, "")
}

// setFlakyAfter 设置 /api/flaky 的失败次数与 Retry-After 响应头。
func (u *upstream) setFlakyAfter(failures int, retryAfter string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.flaky = failures
	u.flakyRetryAfter = retryAfter
}

// hitsFor 返回某个路径被请求的次数。
func (u *upstream) hitsFor(path string) int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.hits[path]
}

// count 记录一次命中。
func (u *upstream) count(r *http.Request) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.hits[r.URL.Path]++
}

// User 是 GET /api/users/{id} 的响应体。
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// handleUser 返回固定格式的用户信息；id 传 "404" 时返回 404。
func (u *upstream) handleUser(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	id := r.PathValue("id")
	if id == "404" {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"code":    "not_found",
			"message": "用户不存在",
		})
		return
	}
	writeJSON(w, http.StatusOK, User{ID: id, Name: "user-" + id})
}

// EchoRequest 是 POST /api/echo 的请求体。
type EchoRequest struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// EchoResponse 是 POST /api/echo 的响应体：回显内容，并附上几个请求头。
type EchoResponse struct {
	Message string            `json:"message"`
	Count   int               `json:"count"`
	Headers map[string]string `json:"headers"`
}

// handleEcho 回显 JSON 请求体，并把上游看到的请求头一起返回。
func (u *upstream) handleEcho(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	var req EchoRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "invalid_json",
			"message": "请求体不是合法 JSON",
		})
		return
	}

	writeJSON(w, http.StatusOK, EchoResponse{
		Message: req.Message,
		Count:   req.Count,
		Headers: map[string]string{
			"User-Agent":   r.Header.Get("User-Agent"),
			"Content-Type": r.Header.Get("Content-Type"),
			"X-Request-ID": r.Header.Get("X-Request-ID"),
		},
	})
}

// handleForm 解析表单并回显。
func (u *upstream) handleForm(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "invalid_form",
			"message": "表单解析失败",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"values":       r.PostForm,
		"content_type": r.Header.Get("Content-Type"),
	})
}

// handleUpload 接收 multipart 文件并返回大小与摘要。
func (u *upstream) handleUpload(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "invalid_multipart",
			"message": "multipart 解析失败",
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "missing_file",
			"message": "缺少 file 字段",
		})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "read_failed",
			"message": "读取上传内容失败",
		})
		return
	}
	sum := sha256.Sum256(content)

	writeJSON(w, http.StatusOK, UploadResult{
		Name:   header.Filename,
		Size:   len(content),
		SHA256: hex.EncodeToString(sum[:]),
	})
}

// handleDownload 输出一段确定性的二进制内容，长度由 ?bytes= 决定。
func (u *upstream) handleDownload(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	size := 4096
	if raw := r.URL.Query().Get("bytes"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 && n <= 1<<20 {
			size = n
		}
	}

	pattern := []byte("0123456789abcdef")
	payload := bytes.Repeat(pattern, (size+len(pattern)-1)/len(pattern))[:size]

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(size))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

// handleFlaky 先失败若干次再成功，用来演示重试。
func (u *upstream) handleFlaky(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	u.mu.Lock()
	remaining := u.flaky
	retryAfter := u.flakyRetryAfter
	if remaining > 0 {
		u.flaky--
	}
	u.mu.Unlock()

	if remaining > 0 {
		if retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"code":    "unavailable",
			"message": "上游暂时不可用",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleGzip 返回 gzip 压缩过的响应体。
func (u *upstream) handleGzip(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")

	writer := gzip.NewWriter(w)
	defer writer.Close()
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"status": "ok",
		"note":   "响应体是 gzip 压缩的",
	})
}

// handleSlow 在 ms 毫秒后返回，用来演示超时。
func (u *upstream) handleSlow(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	delay := 300 * time.Millisecond
	if raw := r.URL.Query().Get("ms"); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms >= 0 {
			delay = time.Duration(ms) * time.Millisecond
		}
	}

	select {
	case <-time.After(delay):
	case <-r.Context().Done():
		return // 客户端提前放弃
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleRedirect 302 跳到用户接口。
func (u *upstream) handleRedirect(w http.ResponseWriter, r *http.Request) {
	u.count(r)
	http.Redirect(w, r, "/api/users/1", http.StatusFound)
}

// handleLoop 永远跳回自己，用来演示重定向上限。
func (u *upstream) handleLoop(w http.ResponseWriter, r *http.Request) {
	u.count(r)
	http.Redirect(w, r, "/api/loop", http.StatusFound)
}

// handleStatus 返回指定状态码和一段 JSON。
func (u *upstream) handleStatus(w http.ResponseWriter, r *http.Request) {
	u.count(r)

	code, err := strconv.Atoi(r.PathValue("code"))
	if err != nil || code < 400 || code > 599 {
		code = http.StatusBadRequest
	}
	writeJSON(w, code, map[string]string{
		"code":    http.StatusText(code),
		"message": "由 /api/status 生成",
	})
}

// writeJSON 是本章统一的小响应写法。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
