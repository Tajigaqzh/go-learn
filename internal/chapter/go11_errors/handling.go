package go11_errors

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound 表示请求的资源不存在，属于「调用方可以据此分支」的哨兵错误。
	ErrNotFound = errors.New("resource not found")
	// ErrDivideByZero 表示除数为零。
	ErrDivideByZero = errors.New("divide by zero")
	// ErrFlushFailed 表示底层缓冲刷新失败。
	ErrFlushFailed = errors.New("flush failed")
)

// ValidationError 表示某个字段没通过校验，字段化信息供调用方按需读取。
type ValidationError struct {
	Field string
	Msg   string
}

// Error 实现 error 接口，只描述「哪个字段、出了什么问题」。
func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Msg)
}

// QueryError 给底层错误补上「哪次查询失败」，并保留错误链。
type QueryError struct {
	Query string
	Err   error
}

// Error 描述失败的操作，不重复底层错误本身的内容。
func (e *QueryError) Error() string {
	return fmt.Sprintf("query %q failed: %v", e.Query, e.Err)
}

// Unwrap 让 errors.Is / errors.As 能继续往链下走。
func (e *QueryError) Unwrap() error {
	return e.Err
}

// FindUser 查询用户；不存在时返回哨兵错误 ErrNotFound。
//
// 返回时用 %w 包装，调用方既能拿到上下文，又能用 errors.Is 判断原因。
func FindUser(id int) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("find user %d: %w", id, ErrNotFound)
	}
	return fmt.Sprintf("user-%d", id), nil
}

// queryUser 用自定义错误类型表达「哪条查询失败了」。
func queryUser(id int) error {
	if id <= 0 {
		return &QueryError{
			Query: fmt.Sprintf("select id=%d from users", id),
			Err:   ErrNotFound,
		}
	}
	return nil
}

// ValidateAge 校验年龄范围，返回带字段信息的 *ValidationError。
func ValidateAge(age int) error {
	if age < 0 || age > 150 {
		return &ValidationError{Field: "age", Msg: "out of range"}
	}
	return nil
}

// ValidateUser 一次性收集所有校验错误，再合并成一条错误返回。
//
// errors.Join 会保留每个子错误，调用方可以用 errors.Is / errors.As 逐个判断。
func ValidateUser(name string, age int) error {
	var errs []error
	if name == "" {
		errs = append(errs, &ValidationError{Field: "name", Msg: "must not be empty"})
	}
	if err := ValidateAge(age); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Divide 用哨兵错误表达业务上可预期的失败，调用方不需要 panic。
func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("divide %g by %g: %w", a, b, ErrDivideByZero)
	}
	return a / b, nil
}

// SafeDivide 把运行时 panic 转成普通错误。
//
// 适合放在「必须让程序活下来」的边界上，比如插件调用、脚本执行、请求级兜底；
// 不要用它掩盖自己代码里的 bug。
func SafeDivide(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("safe divide %d/%d: %v", a, b, r)
		}
	}()
	return a / b, nil
}

// flusher 模拟「写入成功但关闭时可能失败」的资源，比如 bufio.Writer 或文件句柄。
type flusher struct {
	fail bool
}

// Flush 返回预置的失败，用于演示 defer 里的错误处理。
func (f *flusher) Flush() error {
	if f.fail {
		return ErrFlushFailed
	}
	return nil
}

// Save 演示用命名返回值 + defer 把 Flush 的错误合并进最终返回值。
//
// 只有命名返回值才能被 defer 修改；同时用 err == nil 保证「已有错误时不覆盖」，
// 避免用关闭错误掩盖真正的失败原因。
func Save(payload string, failFlush bool) (err error) {
	f := &flusher{fail: failFlush}
	defer func() {
		if flushErr := f.Flush(); flushErr != nil && err == nil {
			err = fmt.Errorf("save %q: %w", payload, flushErr)
		}
	}()

	if payload == "" {
		return errors.New("empty payload")
	}
	return nil
}
