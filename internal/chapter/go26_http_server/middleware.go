package go26_http_server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
)

// Middleware 把下一个 handler 包装成新的 handler，是中间件的统一签名。
type Middleware func(http.Handler) http.Handler

// Chain 按书写顺序把中间件套在 h 外面：
// Chain(h, A, B) 的执行顺序是 A -> B -> h，A 最先看到请求、最后看到响应。
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// statusRecorder 记录状态码和响应字节数，供日志和恢复中间件使用。
//
// http.ResponseWriter 本身不暴露「已经写了什么」，所以要在中间件里包一层。
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

// WriteHeader 只转发第一次调用：后续调用会被 http 包判为多余并记日志。
func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write 记录字节数，并按需要补上隐式的 200。
func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

// Status 返回最终写出的状态码，没写过时按 200 算。
func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

// requestIDKey 是 context 里存放请求 ID 的键类型。
//
// 用未导出的空结构体做键，可以避免和其他包的键冲突。
type requestIDKey struct{}

// RequestIDFrom 取出中间件写入的请求 ID，没有时返回空串。
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestID 是使用默认生成器的请求 ID 中间件。
func RequestID() Middleware {
	return RequestIDWithGenerator(defaultRequestID)
}

// RequestIDWithGenerator 允许替换 ID 生成器，演示和测试可以用固定值。
//
// 优先沿用调用方传来的 X-Request-ID，这样跨服务调用能串成一条链路。
func RequestIDWithGenerator(gen func() string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = gen()
			}
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
		})
	}
}

// defaultRequestID 生成 16 位十六进制随机 ID。
func defaultRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf[:])
}

// Logging 输出一行结构化访问日志。
//
// 一条日志至少要有「谁、请求了什么、结果如何」：请求 ID、方法、路径、状态码、
// 响应字节数。生产环境通常还会补上耗时、客户端 IP 和 User-Agent。
func Logging(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			logger.Printf("request_id=%s method=%s path=%s status=%d bytes=%d",
				RequestIDFrom(r.Context()), r.Method, r.URL.Path, rec.Status(), rec.bytes)
		})
	}
}

// Recover 把 handler 里的 panic 变成 500 响应，避免整个连接被直接掐断。
func Recover(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w}
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				logger.Printf("panic recovered: request_id=%s panic=%v",
					RequestIDFrom(r.Context()), v)
				if rec.status == 0 {
					writeError(rec, http.StatusInternalServerError, "internal",
						"服务器内部错误", "")
				}
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

// CORS 处理跨域响应头和预检请求。
//
// allowedOrigin 传 "*" 时放开所有来源；传具体来源时只回显匹配的那个，
// 并补上 Vary: Origin，避免缓存把 A 站的响应发给 B 站。
func CORS(allowedOrigin string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" {
				switch {
				case allowedOrigin == "*":
					w.Header().Set("Access-Control-Allow-Origin", "*")
				case origin == allowedOrigin:
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}
			}

			// 预检请求只需要回答「允许什么」，不进业务逻辑。
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit 在超过速率时直接返回 429，不再进入业务逻辑。
//
// 这里用一个全局令牌桶演示原理；按 IP 或按用户限流时，把 Limiter 放进
// map[string]*Limiter 里，再按 key 取用即可。
func RateLimit(limiter *Limiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				writeError(w, http.StatusTooManyRequests, "rate_limited", "请求过于频繁", "")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
