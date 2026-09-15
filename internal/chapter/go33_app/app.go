// Package go33_app 用一个小型「用户服务」串起第 32 章讲过的工程手段，构成一个
// 可运行、可测试的服务端项目：分层架构（handler → service → repository）、
// 手动依赖注入、配置加载、SQLite 迁移、REST 接口与统一错误码、中间件链、
// 优雅关闭，以及配套的单元 / 集成测试与基准测试。
//
// 演示全部通过 httptest 或 127.0.0.1 的临时端口运行，数据库用内存 SQLite，
// 不依赖外部网络，输出可复现。
package go33_app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
)

// Option 是 App 的可选配置。
type Option func(*App)

// WithRequestIDGen 注入请求 ID 生成器，便于测试和演示稳定输出。
func WithRequestIDGen(gen func() string) Option {
	return func(a *App) { a.idGen = gen }
}

// idSeq 是默认请求 ID 的自增序列。
var idSeq atomic.Int64

// defaultIDGen 返回形如 req-1、req-2 的请求 ID。
func defaultIDGen() string {
	return fmt.Sprintf("req-%d", idSeq.Add(1))
}

// App 是组装完成的应用程序：持有分层依赖、路由和中间件配置。
type App struct {
	svc    *UserService
	logger *slog.Logger
	mux    *http.ServeMux
	idGen  func() string
}

// NewApp 装配 App 并注册路由，中间件在 Handler() 里包裹。
func NewApp(svc *UserService, logger *slog.Logger, opts ...Option) *App {
	a := &App{svc: svc, logger: logger, idGen: defaultIDGen}
	for _, opt := range opts {
		opt(a)
	}
	a.mux = http.NewServeMux()
	a.registerRoutes(a.mux)
	return a
}

// Handler 返回套好中间件链的根 handler：accessLog(recover(requestID(mux)))。
// accessLog 放在最外层，保证即使请求因 panic 被 recover 救回，也照样写出一条访问日志。
func (a *App) Handler() http.Handler {
	return chain(a.mux, a.accessLogMW, a.recoverMW, a.requestIDMW)
}

// Mux 暴露底层路由表，便于演示或测试临时挂一条路由（例如触发 panic）。
func (a *App) Mux() *http.ServeMux { return a.mux }

// registerRoutes 用 Go 1.22 的「方法 + 路径」模式注册所有路由。
func (a *App) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", a.handleHealthz)
	mux.HandleFunc("POST /api/v1/users", a.handleCreateUser)
	mux.HandleFunc("GET /api/v1/users", a.handleListUsers)
	mux.HandleFunc("GET /api/v1/users/{id}", a.handleGetUser)
}

// handleHealthz 是探活端点。
func (a *App) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// createUserRequest 是创建用户的请求体。
type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// handleCreateUser 解析 JSON、交给 service 校验与落库、返回 201。
func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, badRequest("请求体不是合法 JSON: %v", err))
		return
	}
	u, err := a.svc.Create(r.Context(), req.Name, req.Email)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// handleGetUser 解析路径参数并返回单个用户。
func (a *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, badRequest("id 必须是整数: %q", r.PathValue("id")))
		return
	}
	u, err := a.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// handleListUsers 返回全部用户。
func (a *App) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.svc.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	// 空结果也编码成 []，而不是 null。
	writeJSON(w, http.StatusOK, users)
}
