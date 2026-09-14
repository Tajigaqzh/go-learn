package go26_http_server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// 超时是 HTTP 服务最容易漏掉的配置：http.ListenAndServe 的默认值全是 0，
// 也就是永不超时，一个慢慢发请求头的连接就能一直占着一个 goroutine。
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// NewServer 返回带超时配置的 http.Server。
func NewServer(handler http.Handler, logger *log.Logger) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          logger,
	}
}

// ListenAndServe 监听 addr，收到 ctx 取消后优雅关闭。
func ListenAndServe(ctx context.Context, addr string, handler http.Handler, logger *log.Logger) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	return ServeUntilCanceled(ctx, NewServer(handler, logger), ln, logger)
}

// ServeUntilCanceled 在 ln 上服务，直到 ctx 被取消，然后等在途请求结束再关闭。
//
// 生产代码里 ctx 通常来自 signal.NotifyContext，收到 SIGINT/SIGTERM 后触发。
func ServeUntilCanceled(ctx context.Context, srv *http.Server, ln net.Listener, logger *log.Logger) error {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()

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

	// Shutdown 返回后 Serve 一定会退出，这里回收它的返回值，避免 goroutine 泄漏。
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Println("服务器已优雅关闭")
	return nil
}
