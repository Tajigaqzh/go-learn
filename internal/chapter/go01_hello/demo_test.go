package go01_hello

import (
	"fmt"
	"strings"
	"testing"
)

// TestGetGreeting 测试 GetGreeting 函数的基本行为。
func TestGetGreeting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"空字符串", "", "Hello, World!"},
		{"普通名字", "Go", "Hello, Go!"},
		{"中文名字", "世界", "Hello, 世界!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetGreeting(tt.input)
			if got != tt.want {
				t.Errorf("GetGreeting(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestInternalHelper 演示测试未导出函数。
func TestInternalHelper(t *testing.T) {
	got := internalHelper()
	want := "internal"
	if got != want {
		t.Errorf("internalHelper() = %q, want %q", got, want)
	}
}

// BenchmarkGetGreeting 测试 GetGreeting 的性能。
func BenchmarkGetGreeting(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetGreeting("Benchmark")
	}
}

// TestChapterMetadata 保证章节编号和标题跟文档里的章节名一致。
func TestChapterMetadata(t *testing.T) {
	if Chapter != 1 {
		t.Errorf("Chapter = %d, want 1", Chapter)
	}
	if !strings.Contains(ChapterTitle, "第一个 Go 程序") {
		t.Errorf("ChapterTitle = %q, want 包含「第一个 Go 程序」", ChapterTitle)
	}
}

// ExampleGetGreeting 是示例测试，会出现在 go doc 里。
//
// 两个约束：// Output: 必须是函数里最后一段注释，示例函数也要排在文件最后，
// 否则 go vet 会报 "output comment block must be the last comment block"。
func ExampleGetGreeting() {
	fmt.Println(GetGreeting("Go"))
	// Output: Hello, Go!
}
