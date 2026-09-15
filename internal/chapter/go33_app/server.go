package go33_app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// 超时是 HTTP 服务最容易漏掉的配置：默认值全是 0（永不超时），
// 一个慢慢发请求头的连接就能一直占着一个 goroutine。
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// NewHTTPServer 返回带超时配置的 http.Server。
func NewHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// Run 监听 addr 提供服务，直到 ctx 被取消后优雅关闭再返回。
// 生产环境里 ctx 通常来自 signal.NotifyContext，收到 SIGINT/SIGTERM 后触发。
func Run(ctx context.Context, addr string, handler http.Handler, logger *slog.Logger) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	srv := NewHTTPServer(handler)
	srv.Addr = addr

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	// Shutdown 返回后 Serve 一定已退出；回收返回值，避免 goroutine 泄漏。
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Info("服务器已优雅关闭")
	return nil
}
