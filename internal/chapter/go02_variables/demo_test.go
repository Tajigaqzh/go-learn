package go02_variables

import (
	"math"
	"testing"
)

// TestZeroValues 测试各类型的零值
func TestZeroValues(t *testing.T) {
	var i int
	var f float64
	var b bool
	var s string

	if i != 0 {
		t.Errorf("int 零值应为 0，实际为 %d", i)
	}
	if f != 0.0 {
		t.Errorf("float64 零值应为 0.0，实际为 %f", f)
	}
	if b != false {
		t.Errorf("bool 零值应为 false，实际为 %t", b)
	}
	if s != "" {
		t.Errorf("string 零值应为空字符串，实际为 %q", s)
	}
}

// TestIota 测试 iota 常量生成
func TestIota(t *testing.T) {
	const (
		a = iota // 0
		b        // 1
		c        // 2
	)

	if a != 0 || b != 1 || c != 2 {
		t.Errorf("iota 生成错误：a=%d, b=%d, c=%d", a, b, c)
	}

	const (
		_  = iota             // 0
		KB = 1 << (10 * iota) // 1024
		MB                    // 1048576
	)

	if KB != 1024 {
		t.Errorf("KB 应为 1024，实际为 %d", KB)
	}
	if MB != 1048576 {
		t.Errorf("MB 应为 1048576，实际为 %d", MB)
	}
}

// TestIntegerOverflow 测试整数溢出行为
func TestIntegerOverflow(t *testing.T) {
	var x int8 = 127
	x = x + 1
	if x != -128 {
		t.Errorf("int8 溢出应回绕为 -128，实际为 %d", x)
	}

	var y uint8 = 255
	y = y + 1
	if y != 0 {
		t.Errorf("uint8 溢出应回绕为 0，实际为 %d", y)
	}
}

// TestFloatPrecision 测试浮点精度问题
func TestFloatPrecision(t *testing.T) {
	a := 0.1
	b := 0.2
	c := a + b

	// 直接比较会失败
	if c == 0.3 {
		t.Log("0.1 + 0.2 == 0.3 意外成立")
	}

	// 应使用 epsilon 比较
	epsilon := 1e-9
	if math.Abs(c-0.3) > epsilon {
		t.Errorf("0.1 + 0.2 与 0.3 的差距超过 epsilon")
	}
}

// TestNaN 测试 NaN 的特殊性质
func TestNaN(t *testing.T) {
	nan := math.NaN()

	// NaN 不等于任何值，包括自身
	if nan == nan {
		t.Error("NaN 不应等于自身")
	}

	// 应使用 math.IsNaN 检测
	if !math.IsNaN(nan) {
		t.Error("math.IsNaN 应返回 true")
	}
}

// TestByteAndRune 测试 byte 与 rune 的区别
func TestByteAndRune(t *testing.T) {
	s := "Go语言"

	// len 返回字节数，不是字符数
	if len(s) != 8 {
		t.Errorf("字符串字节长度应为 8，实际为 %d", len(s))
	}

	// []rune 返回字符数
	runes := []rune(s)
	if len(runes) != 4 {
		t.Errorf("字符串字符数应为 4，实际为 %d", len(runes))
	}

	// byte 是 uint8 别名
	var b byte = 65
	if b != uint8(65) {
		t.Error("byte 应与 uint8 等价")
	}

	// rune 是 int32 别名
	var r rune = '中'
	if r != int32(20013) {
		t.Errorf("'中' 的 Unicode 码点应为 20013，实际为 %d", r)
	}
}

// TestTypeConversion 测试类型转换
func TestTypeConversion(t *testing.T) {
	// 浮点转整数：截断
	var pi float64 = 3.14159
	var intPi int = int(pi)
	if intPi != 3 {
		t.Errorf("float64(3.14159) 转 int 应为 3，实际为 %d", intPi)
	}

	// 有损转换
	var big int64 = 1000
	var small int8 = int8(big) // 1000 % 256 = 232，但实际是有符号截断
	expectedSmall := int8(-24) // 1000 的低 8 位
	if small != expectedSmall {
		t.Logf("int64(1000) 转 int8 发生截断，实际为 %d", small)
	}
}

// TestTypeDefinitionVsAlias 测试类型定义与别名
func TestTypeDefinitionVsAlias(t *testing.T) {
	type Celsius float64
	type MyInt = int

	var c Celsius = 100.0
	var f float64 = 100.0

	// Celsius 和 float64 是不同类型，不能直接比较
	// if c == f {} // 编译错误
	if float64(c) != f {
		t.Error("类型定义转换后应相等")
	}

	// MyInt 和 int 是同一类型，可以直接比较
	var a MyInt = 10
	var b int = 10
	if a != MyInt(b) {
		t.Error("类型别名应与原类型兼容")
	}
}

// BenchmarkStringConcat 比较字符串拼接性能
func BenchmarkStringConcat(b *testing.B) {
	s := ""
	for i := 0; i < b.N; i++ {
		s = s + "x"
	}
}

// BenchmarkByteSliceConcat 测试 []byte 拼接性能
func BenchmarkByteSliceConcat(b *testing.B) {
	var buf []byte
	for i := 0; i < b.N; i++ {
		buf = append(buf, 'x')
	}
}

// BenchmarkTypeConversion 测试类型转换开销
func BenchmarkTypeConversion(b *testing.B) {
	var i int = 42
	var f float64
	for n := 0; n < b.N; n++ {
		f = float64(i)
	}
	_ = f
}

// BenchmarkRuneConversion 测试 string 与 []rune 转换开销
func BenchmarkRuneConversion(b *testing.B) {
	s := "Go语言编程"
	for n := 0; n < b.N; n++ {
		_ = []rune(s)
	}
}
