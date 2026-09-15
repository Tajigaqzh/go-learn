package go33_app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Code 是稳定的业务错误码，给调用方程序判断用；不要把人类可读的 message 当判据。
type Code string

// 统一错误码集合。机器按 code 分支，状态码则交给 *AppError.Status。
const (
	CodeInvalidArgument Code = "invalid_argument"
	CodeNotFound        Code = "not_found"
	CodeConflict        Code = "already_exists"
	CodeInternal        Code = "internal"
)

// AppError 是贯穿 handler → service → repository 各层的业务错误：
// Code 给程序、Message 给人、Status 给 HTTP 传输层。
type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

// Error 让 *AppError 满足 error 接口。
func (e *AppError) Error() string { return e.Message }

// newAppError 构造带错误码的业务错误。
func newAppError(code Code, status int, message string) *AppError {
	return &AppError{Code: code, Status: status, Message: message}
}

// badRequest / notFound / conflict / internalErr 是各层的便捷构造器。
func badRequest(format string, args ...any) *AppError {
	return newAppError(CodeInvalidArgument, http.StatusBadRequest, fmt.Sprintf(format, args...))
}

func notFound(format string, args ...any) *AppError {
	return newAppError(CodeNotFound, http.StatusNotFound, fmt.Sprintf(format, args...))
}

func conflict(format string, args ...any) *AppError {
	return newAppError(CodeConflict, http.StatusConflict, fmt.Sprintf(format, args...))
}

func internalErr(format string, args ...any) *AppError {
	return newAppError(CodeInternal, http.StatusInternalServerError, fmt.Sprintf(format, args...))
}

// responseOf 把任意 error 解析成 HTTP 层需要的三要素：状态码、code、message。
// *AppError 直接取出；其他错误统一归为 500/internal，避免把内部细节泄露给调用方。
func responseOf(err error) (int, Code, string) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Status, ae.Code, ae.Message
	}
	return http.StatusInternalServerError, CodeInternal, "服务器内部错误"
}

// writeJSON 写一个 JSON 响应体。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 把错误统一成 JSON 错误体写回；这是所有失败路径的唯一出口。
func writeError(w http.ResponseWriter, err error) {
	status, code, msg := responseOf(err)
	writeJSON(w, status, &AppError{Code: code, Status: status, Message: msg})
}
