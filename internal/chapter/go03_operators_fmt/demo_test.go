package go03_operators_fmt

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// TestArithmeticOperators 测试算术运算符
func TestArithmeticOperators(t *testing.T) {
	// 整数除法截断
	if 10/3 != 3 {
		t.Errorf("10 / 3 应为 3，实际为 %d", 10/3)
	}

	// 负数取余：结果符号与被除数相同
	if -10%3 != -1 {
		t.Errorf("-10 %% 3 应为 -1，实际为 %d", -10%3)
	}
	if 10%-3 != 1 {
		t.Errorf("10 %% -3 应为 1，实际为 %d", 10%-3)
	}
}

// TestComparisonOperators 测试比较运算符
func TestComparisonOperators(t *testing.T) {
	// 字符串比较：字典序
	if !("apple" < "banana") {
		t.Error("apple 应小于 banana")
	}

	// 结构体比较
	type Point struct{ X, Y int }
	p1 := Point{1, 2}
	p2 := Point{1, 2}
	if p1 != p2 {
		t.Error("相同字段的结构体应相等")
	}
}

// TestLogicalOperators 测试逻辑运算符短路求值
func TestLogicalOperators(t *testing.T) {
	called := false
	sideEffect := func() bool {
		called = true
		return true
	}

	// && 短路：左侧为 false，右侧不执行
	called = false
	_ = false && sideEffect()
	if called {
		t.Error("&& 短路失败，右侧不应执行")
	}

	// || 短路：左侧为 true，右侧不执行
	called = false
	_ = true || sideEffect()
	if called {
		t.Error("|| 短路失败，右侧不应执行")
	}

	// && 不短路：左侧为 true，右侧执行
	called = false
	_ = true && sideEffect()
	if !called {
		t.Error("&& 左侧为 true，右侧应执行")
	}

	// || 不短路：左侧为 false，右侧执行
	called = false
	_ = false || sideEffect()
	if !called {
		t.Error("|| 左侧为 false，右侧应执行")
	}
}

// TestBitwiseOperators 测试位运算符
func TestBitwiseOperators(t *testing.T) {
	a, b := 0b1100, 0b1010 // 12, 10

	if a&b != 0b1000 {
		t.Errorf("按位与错误：%d & %d = %d，应为 %d", a, b, a&b, 0b1000)
	}
	if a|b != 0b1110 {
		t.Errorf("按位或错误：%d | %d = %d，应为 %d", a, b, a|b, 0b1110)
	}
	if a^b != 0b0110 {
		t.Errorf("按位异或错误：%d ^ %d = %d，应为 %d", a, b, a^b, 0b0110)
	}

	// 位清除：a &^ b 清除 a 中 b 为 1 的位
	if a&^b != 0b0100 {
		t.Errorf("位清除错误：%d &^ %d = %d，应为 %d", a, b, a&^b, 0b0100)
	}

	// 移位
	if a<<1 != 24 {
		t.Errorf("左移错误：%d << 1 = %d，应为 24", a, a<<1)
	}
	if a>>1 != 6 {
		t.Errorf("右移错误：%d >> 1 = %d，应为 6", a, a>>1)
	}
}

// TestOperatorPrecedence 测试运算符优先级
func TestOperatorPrecedence(t *testing.T) {
	// 乘法优先于加法
	if 2+3*4 != 14 {
		t.Errorf("2 + 3 * 4 应为 14，实际为 %d", 2+3*4)
	}

	// 移位优先级：移位运算符优先级高于加法
	if 1<<2+3 != 7 {
		// 这是 (1 << 2) + 3 = 4 + 3 = 7
		t.Errorf("1 << 2 + 3 应为 7，实际为 %d", 1<<2+3)
	}
}

// TestPrintfFormats 测试 Printf 格式化
func TestPrintfFormats(t *testing.T) {
	// 测试整数格式化
	tests := []struct {
		format string
		value  int
		want   string
	}{
		{"%d", 42, "42"},
		{"%b", 42, "101010"},
		{"%o", 42, "52"},
		{"%x", 42, "2a"},
		{"%X", 42, "2A"},
		{"%05d", 42, "00042"},
		{"%+d", 42, "+42"},
	}

	for _, tt := range tests {
		got := fmt.Sprintf(tt.format, tt.value)
		if got != tt.want {
			t.Errorf("Sprintf(%q, %d) = %q, want %q", tt.format, tt.value, got, tt.want)
		}
	}
}

// TestPrintfFloat 测试浮点数格式化
func TestPrintfFloat(t *testing.T) {
	pi := math.Pi

	tests := []struct {
		format string
		want   string
	}{
		{"%.2f", "3.14"},
		{"%.4f", "3.1416"},
		{"%e", "3.141593e+00"},
		{"%E", "3.141593E+00"},
	}

	for _, tt := range tests {
		got := fmt.Sprintf(tt.format, pi)
		if got != tt.want {
			t.Errorf("Sprintf(%q, pi) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

// TestPrintfString 测试字符串格式化
func TestPrintfString(t *testing.T) {
	s := "Hello\nWorld"

	tests := []struct {
		format string
		want   string
	}{
		{"%s", "Hello\nWorld"},
		{"%q", `"Hello\nWorld"`},
		{"%.5s", "Hello"},
	}

	for _, tt := range tests {
		got := fmt.Sprintf(tt.format, s)
		if got != tt.want {
			t.Errorf("Sprintf(%q, %q) = %q, want %q", tt.format, s, got, tt.want)
		}
	}
}

// TestStringerInterface 测试 Stringer 接口
func TestStringerInterface(t *testing.T) {
	p := Point{3, 4}

	// %v 和 %s 应该调用 String 方法
	want := "Point(3, 4)"
	if got := fmt.Sprintf("%v", p); got != want {
		t.Errorf("%%v: got %q, want %q", got, want)
	}
	if got := fmt.Sprintf("%s", p); got != want {
		t.Errorf("%%s: got %q, want %q", got, want)
	}

	// %#v 应该显示 Go 语法，不调用 String
	wantGo := "go03_operators_fmt.Point{X:3, Y:4}"
	if got := fmt.Sprintf("%#v", p); got != wantGo {
		t.Errorf("%%#v: got %q, want %q", got, wantGo)
	}
}

// TestSprintfFprintf 测试 Sprintf 和 Fprintf
func TestSprintfFprintf(t *testing.T) {
	// Sprintf 返回字符串
	got := fmt.Sprintf("圆周率约等于 %.2f", math.Pi)
	want := "圆周率约等于 3.14"
	if got != want {
		t.Errorf("Sprintf: got %q, want %q", got, want)
	}

	// Fprintf 写入 io.Writer
	var buf strings.Builder
	n, err := fmt.Fprintf(&buf, "%d + %d = %d", 1, 2, 3)
	if err != nil {
		t.Errorf("Fprintf 返回错误: %v", err)
	}
	if n != 9 {
		t.Errorf("Fprintf 写入字节数: got %d, want 9", n)
	}
	if buf.String() != "1 + 2 = 3" {
		t.Errorf("Fprintf 结果: got %q, want %q", buf.String(), "1 + 2 = 3")
	}
}

// BenchmarkSprintf 基准测试 Sprintf
func BenchmarkSprintf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("value = %d", i)
	}
}

// BenchmarkStringBuilder 基准测试 strings.Builder
func BenchmarkStringBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		buf.WriteString("value = ")
		buf.WriteString(fmt.Sprint(i))
		_ = buf.String()
	}
}

// BenchmarkBitwiseOps 基准测试位运算
func BenchmarkBitwiseOps(b *testing.B) {
	x := 0b11110000
	y := 0b00001111
	for i := 0; i < b.N; i++ {
		_ = x & y
		_ = x | y
		_ = x ^ y
		_ = x << 2
		_ = x >> 2
	}
}

// BenchmarkArithmeticOps 基准测试算术运算
func BenchmarkArithmeticOps(b *testing.B) {
	x, y := 42, 7
	for i := 0; i < b.N; i++ {
		_ = x + y
		_ = x - y
		_ = x * y
		_ = x / y
		_ = x % y
	}
}
