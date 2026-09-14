package go26_http_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// maxBodyBytes 是请求体上限：超过就回 413，避免恶意的超大 body 吃满内存。
const maxBodyBytes = 1 << 10

// APIError 是统一的错误响应体。
//
// 统一的形状让客户端只需要写一次错误处理；Details 只在需要给排查线索时出现。
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error 让 APIError 也能当 error 使用。
func (e APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Option 是构造 API 时的可选配置。
type Option func(*API)

// WithAllowedOrigin 设置 CORS 允许的来源，"*" 表示不限制。
func WithAllowedOrigin(origin string) Option {
	return func(a *API) { a.allowedOrigin = origin }
}

// WithLimiter 打开限流。
func WithLimiter(limiter *Limiter) Option {
	return func(a *API) { a.limiter = limiter }
}

// WithRequestIDGenerator 替换请求 ID 生成器。
func WithRequestIDGenerator(gen func() string) Option {
	return func(a *API) { a.requestIDGen = gen }
}

// API 把存储、路由和中间件组装成一个 http.Handler。
type API struct {
	store         *Store
	logger        *log.Logger
	allowedOrigin string
	limiter       *Limiter
	requestIDGen  func() string
	mux           *http.ServeMux
	handler       http.Handler
}

// NewAPI 按给定配置组装 API。
func NewAPI(store *Store, logger *log.Logger, options ...Option) *API {
	api := &API{
		store:         store,
		logger:        logger,
		allowedOrigin: "https://example.com",
		requestIDGen:  defaultRequestID,
	}
	for _, option := range options {
		option(api)
	}

	api.mux = http.NewServeMux()
	api.routes()

	middlewares := []Middleware{
		RequestIDWithGenerator(api.requestIDGen),
		Logging(logger),
		Recover(logger),
		CORS(api.allowedOrigin),
	}
	if api.limiter != nil {
		middlewares = append(middlewares, RateLimit(api.limiter))
	}
	api.handler = Chain(api.mux, middlewares...)
	return api
}

// Handler 返回可以直接交给 http.Server 的 handler。
func (a *API) Handler() http.Handler {
	return a.handler
}

// Mux 返回内部路由，测试里可以直接检查路由注册。
func (a *API) Mux() *http.ServeMux {
	return a.mux
}

// routes 注册路由。
//
// 从 Go 1.22 起，ServeMux 的模式支持「方法 + 路径」和 {name} 通配符，
// 不再需要为每个方法手写一遍 switch r.Method。
func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", a.handleHealth)
	a.mux.HandleFunc("GET /api/tasks", a.handleListTasks)
	a.mux.HandleFunc("POST /api/tasks", a.handleCreateTask)
	a.mux.HandleFunc("GET /api/tasks/{id}", a.handleGetTask)
	a.mux.HandleFunc("PATCH /api/tasks/{id}", a.handleUpdateTask)
	a.mux.HandleFunc("DELETE /api/tasks/{id}", a.handleDeleteTask)
}

// handleHealth 提供给负载均衡和容器探针的健康检查。
func (a *API) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleListTasks 支持 done 过滤和 limit 截断。
func (a *API) handleListTasks(w http.ResponseWriter, r *http.Request) {
	done, err := parseOptionalBool(r, "done")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error(), "")
		return
	}
	limit, err := parseOptionalInt(r, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error(), "")
		return
	}

	tasks := a.store.List(done, limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"items": tasks,
		"count": len(tasks),
	})
}

// createTaskRequest 是 POST 的请求体。
type createTaskRequest struct {
	Title string `json:"title"`
}

// handleCreateTask 解析 JSON、校验字段，然后返回 201 和 Location 头。
func (a *API) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r) {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
			"Content-Type 必须是 application/json", "")
		return
	}

	var req createTaskRequest
	if err := a.decodeJSON(w, r, &req); err != nil {
		return // decodeJSON 已经写好响应
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "invalid_argument", "title 不能为空", "")
		return
	}

	task := a.store.Create(title)
	w.Header().Set("Location", "/api/tasks/"+strconv.Itoa(task.ID))
	writeJSON(w, http.StatusCreated, task)
}

// handleGetTask 用 PathValue 取出路径参数。
func (a *API) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error(), "")
		return
	}

	task, ok := a.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "任务不存在", "")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// updateTaskRequest 是 PATCH 的请求体，字段用指针区分「没传」和「传了零值」。
type updateTaskRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}

// handleUpdateTask 局部更新任务。
func (a *API) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error(), "")
		return
	}
	if !isJSON(r) {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
			"Content-Type 必须是 application/json", "")
		return
	}

	var req updateTaskRequest
	if err := a.decodeJSON(w, r, &req); err != nil {
		return
	}
	if req.Title == nil && req.Done == nil {
		writeError(w, http.StatusBadRequest, "invalid_argument",
			"至少要提供 title 或 done 中的一个", "")
		return
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "invalid_argument", "title 不能为空", "")
			return
		}
		req.Title = &trimmed
	}

	task, ok := a.store.Update(id, req.Title, req.Done)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "任务不存在", "")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// handleDeleteTask 删除任务，成功时不带响应体。
func (a *API) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error(), "")
		return
	}

	if !a.store.Delete(id) {
		writeError(w, http.StatusNotFound, "not_found", "任务不存在", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeJSON 读取并解析请求体，把各种失败翻译成合适的错误响应。
//
// http.MaxBytesReader 负责限制大小，DisallowUnknownFields 负责拦住拼错的字段——
// 后者能把「客户端写了 done，服务端只看 data」这类静默 bug 变成明确的 400。
func (a *API) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large",
				fmt.Sprintf("请求体不能超过 %d 字节", maxBodyBytes), "")
			return err
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "请求体不是合法 JSON", err.Error())
		return err
	}
	return nil
}

// pathID 解析 {id} 路径参数。
func pathID(r *http.Request) (int, error) {
	raw := r.PathValue("id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("id 必须是整数，收到 %q", raw)
	}
	if id <= 0 {
		return 0, fmt.Errorf("id 必须大于 0，收到 %d", id)
	}
	return id, nil
}

// isJSON 判断请求体是不是 JSON。
func isJSON(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
}

// parseOptionalBool 读取可选的布尔查询参数。
func parseOptionalBool(r *http.Request, name string) (*bool, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("%s 必须是布尔值，收到 %q", name, raw)
	}
	return &value, nil
}

// parseOptionalInt 读取可选的整数查询参数。
func parseOptionalInt(r *http.Request, name string) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是整数，收到 %q", name, raw)
	}
	if value < 0 {
		return 0, fmt.Errorf("%s 不能是负数，收到 %d", name, value)
	}
	return value, nil
}

// writeJSON 写出 JSON 响应：先设 Content-Type，再写状态码，最后编码响应体。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}
	// 走到这里响应头已经发出，编码失败也只能记录，不能再改状态码。
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 是统一的错误出口：所有失败路径都从这里写出同样形状的 JSON。
func writeError(w http.ResponseWriter, status int, code, message, details string) {
	writeJSON(w, status, APIError{Code: code, Message: message, Details: details})
}
