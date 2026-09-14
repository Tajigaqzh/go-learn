package go19_testing

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAdd 基本的单元测试。
func TestAdd(t *testing.T) {
	got := Add(2, 3)
	want := 5
	if got != want {
		t.Errorf("Add(2, 3) = %d, want %d", got, want)
	}
}

// TestAddTable 表驱动测试。
func TestAddTable(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"正数", 2, 3, 5},
		{"负数", -1, -2, -3},
		{"零", 0, 0, 0},
		{"混合", -5, 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Add(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// TestAddParallel 并行子测试。
func TestAddParallel(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"case1", 1, 1, 2},
		{"case2", 2, 2, 4},
		{"case3", 3, 3, 6},
	}

	for _, tt := range tests {
		tt := tt // 捕获循环变量（Go 1.22+ 不需要）
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // 标记为可并行
			got := Add(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// TestReverse 测试字符串反转。
func TestReverse(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "olleh"},
		{"世界", "界世"},
		{"", ""},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Reverse(tt.input)
			if got != tt.want {
				t.Errorf("Reverse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestIsPalindrome 测试回文判断。
func TestIsPalindrome(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"noon", true},
		{"hello", false},
		{"", true},
		{"a", true},
		{"Aa", true}, // 不区分大小写
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsPalindrome(tt.input)
			if got != tt.want {
				t.Errorf("IsPalindrome(%q) = %t, want %t", tt.input, got, tt.want)
			}
		})
	}
}

// assertEqual 是一个辅助函数，演示 t.Helper() 的作用。
func assertEqual(t *testing.T, got, want int) {
	t.Helper() // 标记为辅助函数，错误时跳过这一帧
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

// TestWithHelper 使用辅助函数的测试。
func TestWithHelper(t *testing.T) {
	assertEqual(t, Add(1, 2), 3)
	assertEqual(t, Add(5, 7), 12)
}

// BenchmarkAdd 基准测试：加法。
func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(2, 3)
	}
}

// BenchmarkReverse 基准测试：字符串反转。
func BenchmarkReverse(b *testing.B) {
	s := "hello world"
	b.ResetTimer() // 重置计时器
	for i := 0; i < b.N; i++ {
		Reverse(s)
	}
}

// BenchmarkConcat 基准测试：字符串拼接（低效版本）。
func BenchmarkConcat(b *testing.B) {
	strs := []string{"a", "b", "c", "d", "e"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Concat(strs)
	}
}

// BenchmarkConcatBuilder 基准测试：字符串拼接（高效版本）。
func BenchmarkConcatBuilder(b *testing.B) {
	strs := []string{"a", "b", "c", "d", "e"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcatBuilder(strs)
	}
}

// FuzzReverse 模糊测试：字符串反转。
func FuzzReverse(f *testing.F) {
	// 提供种子语料
	f.Add("hello")
	f.Add("世界")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		// 反转两次应该得到原字符串
		reversed := Reverse(s)
		doubleReversed := Reverse(reversed)
		if s != doubleReversed {
			t.Errorf("Reverse(Reverse(%q)) = %q, want %q", s, doubleReversed, s)
		}
	})
}

// ExampleAdd 示例测试：加法。
func ExampleAdd() {
	result := Add(2, 3)
	fmt.Println(result)
	// Output: 5
}

// ExampleReverse 示例测试：字符串反转。
func ExampleReverse() {
	result := Reverse("hello")
	fmt.Println(result)
	// Output: olleh
}

// TestHTTPHandler 测试 HTTP 处理器（使用 httptest）。
func TestHTTPHandler(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}

	req := httptest.NewRequest("GET", "/hello", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	want := "Hello, World!"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// TestWithCleanup 演示 t.Cleanup 的使用。
func TestWithCleanup(t *testing.T) {
	// 模拟需要清理的资源
	resource := "allocated"

	t.Cleanup(func() {
		// 测试结束时自动调用
		resource = "cleaned"
		t.Logf("Cleanup called, resource = %s", resource)
	})

	// 测试逻辑
	if resource != "allocated" {
		t.Errorf("resource should be allocated")
	}
}

// TestWithTempDir 演示 t.TempDir 的使用。
func TestWithTempDir(t *testing.T) {
	dir := t.TempDir() // 创建临时目录，测试结束时自动删除

	// 测试逻辑：可以在 dir 里创建文件
	if dir == "" {
		t.Error("TempDir should return a non-empty path")
	}
	t.Logf("TempDir created: %s", dir)
}
