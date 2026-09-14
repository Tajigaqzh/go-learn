package go11_errors

import (
	"errors"
	"fmt"
	"testing"
)

// benchErr 接收基准结果，防止编译器把没人用的调用优化掉。
var benchErr error

// BenchmarkErrorSentinel 复用包级哨兵：没有额外分配。
func BenchmarkErrorSentinel(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchErr = ErrNotFound
	}
}

// BenchmarkErrorNewInline 每次就地 errors.New：一次分配。
func BenchmarkErrorNewInline(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchErr = errors.New("cache miss")
	}
}

// BenchmarkErrorWrapFmt 用 fmt.Errorf + %w 补上下文：分配更多，但保留错误链。
func BenchmarkErrorWrapFmt(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchErr = fmt.Errorf("load user 42: %w", ErrNotFound)
	}
}
