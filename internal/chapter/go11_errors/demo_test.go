package go11_errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestFindUserWrapsSentinel 验证包装后的错误仍能用 errors.Is 认出哨兵。
func TestFindUserWrapsSentinel(t *testing.T) {
	_, err := FindUser(0)
	if err == nil {
		t.Fatal("FindUser(0) 应该返回错误")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(%v, ErrNotFound) = false, want true", err)
	}
	if err == ErrNotFound {
		t.Error("返回值应该是包装后的错误，而不是哨兵本身")
	}
	if !strings.Contains(err.Error(), "find user 0") {
		t.Errorf("错误信息缺少上下文：%v", err)
	}
}

// TestFindUserOK 验证成功路径返回零值错误。
func TestFindUserOK(t *testing.T) {
	name, err := FindUser(7)
	if err != nil {
		t.Fatalf("FindUser(7) 返回错误：%v", err)
	}
	if name != "user-7" {
		t.Errorf("FindUser(7) = %q, want %q", name, "user-7")
	}
}

// TestValidationErrorAs 验证 errors.As 能穿过包装层取回字段。
func TestValidationErrorAs(t *testing.T) {
	err := fmt.Errorf("validate user: %w", ValidateAge(200))

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("errors.As(%v, *ValidationError) = false, want true", err)
	}
	if ve.Field != "age" {
		t.Errorf("Field = %q, want %q", ve.Field, "age")
	}

	// 直接类型断言在包装之后会失效，这是 errors.As 存在的理由。
	if _, ok := err.(*ValidationError); ok {
		t.Error("包装后的错误不应该能直接断言成 *ValidationError")
	}
}

// TestValidateUserJoinsAllErrors 验证多个校验错误会被完整收集。
func TestValidateUserJoinsAllErrors(t *testing.T) {
	err := ValidateUser("", 200)
	if err == nil {
		t.Fatal("空名字 + 超范围年龄应该返回错误")
	}

	var multi interface{ Unwrap() []error }
	if !errors.As(err, &multi) {
		t.Fatalf("%v 应该实现 Unwrap() []error", err)
	}
	if got := len(multi.Unwrap()); got != 2 {
		t.Errorf("子错误数量 = %d, want 2", got)
	}

	fields := make([]string, 0, 2)
	for _, sub := range multi.Unwrap() {
		var ve *ValidationError
		if errors.As(sub, &ve) {
			fields = append(fields, ve.Field)
		}
	}
	if len(fields) != 2 || fields[0] != "name" || fields[1] != "age" {
		t.Errorf("收集到的字段 = %v, want [name age]", fields)
	}
}

// TestValidateUserPasses 全部合法时返回 nil。
func TestValidateUserPasses(t *testing.T) {
	if err := ValidateUser("Alice", 30); err != nil {
		t.Errorf("ValidateUser(\"Alice\", 30) = %v, want nil", err)
	}
}

// TestDivide 验证除零返回哨兵错误，正常相除返回结果。
func TestDivide(t *testing.T) {
	if got, err := Divide(10, 4); err != nil || got != 2.5 {
		t.Errorf("Divide(10, 4) = (%v, %v), want (2.5, nil)", got, err)
	}

	_, err := Divide(10, 0)
	if !errors.Is(err, ErrDivideByZero) {
		t.Errorf("Divide(10, 0) 的错误 = %v, want ErrDivideByZero", err)
	}
}

// TestSafeDivideRecoversPanic 验证 panic 被转换成普通错误。
func TestSafeDivideRecoversPanic(t *testing.T) {
	result, err := SafeDivide(10, 0)
	if err == nil {
		t.Fatal("SafeDivide(10, 0) 应该把 panic 转成错误")
	}
	if result != 0 {
		t.Errorf("panic 之后 result = %d, want 0", result)
	}
	if !strings.Contains(err.Error(), "integer divide by zero") {
		t.Errorf("错误信息应保留 panic 内容：%v", err)
	}

	if got, err := SafeDivide(10, 2); err != nil || got != 5 {
		t.Errorf("SafeDivide(10, 2) = (%d, %v), want (5, nil)", got, err)
	}
}

// TestSaveReportsFlushError 验证 defer 里的 Flush 错误会进入返回值。
func TestSaveReportsFlushError(t *testing.T) {
	if err := Save("data", false); err != nil {
		t.Errorf("Save(\"data\", false) = %v, want nil", err)
	}

	err := Save("data", true)
	if !errors.Is(err, ErrFlushFailed) {
		t.Errorf("Save(\"data\", true) = %v, want ErrFlushFailed", err)
	}

	// 已有错误时不被关闭错误覆盖，真正的失败原因优先。
	err = Save("", true)
	if err == nil || errors.Is(err, ErrFlushFailed) {
		t.Errorf("Save(\"\", true) = %v, want 保留 empty payload", err)
	}
}

// TestSentinelMessagesAreLowercase 提醒错误信息按「小写、无句号」书写。
func TestSentinelMessagesAreLowercase(t *testing.T) {
	for name, err := range map[string]error{
		"ErrNotFound":     ErrNotFound,
		"ErrDivideByZero": ErrDivideByZero,
		"ErrFlushFailed":  ErrFlushFailed,
	} {
		msg := err.Error()
		if msg != strings.ToLower(msg) {
			t.Errorf("%s 的信息应小写：%q", name, msg)
		}
		if strings.HasSuffix(msg, ".") {
			t.Errorf("%s 的信息不应以句号结尾：%q", name, msg)
		}
	}
}

// ExampleValidateUser 演示 Join 出来的错误是多行文本。
func ExampleValidateUser() {
	err := ValidateUser("", 200)
	for _, line := range strings.Split(err.Error(), "\n") {
		fmt.Println(line)
	}
	// Output:
	// invalid name: must not be empty
	// invalid age: out of range
}

// ExampleQueryError 演示自定义错误类型的输出格式。
func ExampleQueryError() {
	err := queryUser(0)
	fmt.Println(err)
	fmt.Println(errors.Is(err, ErrNotFound))
	// Output:
	// query "select id=0 from users" failed: resource not found
	// true
}
