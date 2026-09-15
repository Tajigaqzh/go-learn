package go33_app

import (
	"net/http"
)

// chain 按顺序套中间件，列表里的第一个在最外层。
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// statusRecorder 记录 handler 写出的状态码，供访问日志读取。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader 记下状态码再转发。
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// requestIDMW 生成或沿用调用方的请求 ID，并写回响应头。最内层，最靠近业务 handler。
func (a *App) requestIDMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = a.idGen()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// accessLogMW 记录方法、路径、状态码和请求 ID。它是最外层中间件，能读到 requestIDMW 写出的头；
// 并且把 recoverMW 包在里面，保证被 panic 救回的请求也能记下 status=500。
func (a *App) accessLogMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		a.logger.Info("access",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"request_id", rec.Header().Get("X-Request-ID"),
		)
	})
}

// recoverMW 捕获 panic，把它转成 500 响应，避免单个 panic 掐断整个连接。
// 它排在 accessLogMW 内层，panic 救回后仍会回到 accessLogMW 写访问日志。
func (a *App) recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				a.logger.Error("panic recovered", "panic", p)
				// 前提：handler panic 前还没写过响应；否则这里只能记日志、无法改写状态码。
				writeError(w, internalErr("panic: %v", p))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
