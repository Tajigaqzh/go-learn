package go29_cli_logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID 返回携带请求 ID 的 context。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// LoggerFromContext 返回自动带请求 ID 的派生 logger。
func LoggerFromContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
		return logger.With("request_id", requestID)
	}
	return logger
}

// RotatingWriter 在文件超过大小上限前把旧文件重命名为 path.1。
type RotatingWriter struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	file     *os.File
	size     int64
}

// NewRotatingWriter 打开日志文件并返回大小轮转写入器。
func NewRotatingWriter(path string, maxBytes int64) (*RotatingWriter, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("maxBytes 必须大于 0")
	}
	w := &RotatingWriter{path: path, maxBytes: maxBytes}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatingWriter) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("打开日志文件: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("读取日志文件状态: %w", err)
	}
	w.file, w.size = file, info.Size()
	return nil
}

// Write 实现 io.Writer；单条记录大于上限时仍完整写入当前新文件。
func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.size > 0 && w.size+int64(len(p)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("关闭待轮转日志: %w", err)
	}
	w.file = nil
	backup := w.path + ".1"
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除旧备份: %w", err)
	}
	if err := os.Rename(w.path, backup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("轮转日志: %w", err)
	}
	return w.open()
}

// Close 关闭当前日志文件。
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// LogFileNames 返回示例使用的当前文件和备份文件名。
func LogFileNames(path string) (string, string) {
	return filepath.Base(path), filepath.Base(path) + ".1"
}

var _ io.WriteCloser = (*RotatingWriter)(nil)
