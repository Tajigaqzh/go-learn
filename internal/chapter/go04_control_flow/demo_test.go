package go04_control_flow

import (
	"testing"
)

// TestIfWithInit 测试 if 初始化语句的作用域
func TestIfWithInit(t *testing.T) {
	if x := 10; x > 5 {
		if x != 10 {
			t.Errorf("期望 x=10，实际 %d", x)
		}
	}
	// x 在这里不可见
}

// TestForLoopVariableCapture 测试 Go 1.22+ 循环变量语义
func TestForLoopVariableCapture(t *testing.T) {
	var funcs []func() int
	for i := 0; i < 3; i++ {
		funcs = append(funcs, func() int {
			return i
		})
	}
	// Go 1.22+：每次迭代 i 是新变量，应该捕获 0, 1, 2
	expected := []int{0, 1, 2}
	for idx, f := range funcs {
		got := f()
		if got != expected[idx] {
			t.Errorf("funcs[%d]() = %d，期望 %d", idx, got, expected[idx])
		}
	}
}

// TestRangeSliceModification 测试 range 遍历时修改切片
func TestRangeSliceModification(t *testing.T) {
	nums := []int{1, 2, 3}
	// range 遍历时修改元素不影响迭代次数（迭代前已确定）
	count := 0
	for i, v := range nums {
		nums[i] = v * 2
		count++
	}
	if count != 3 {
		t.Errorf("期望迭代 3 次，实际 %d 次", count)
	}
	expected := []int{2, 4, 6}
	for i, v := range nums {
		if v != expected[i] {
			t.Errorf("nums[%d] = %d，期望 %d", i, v, expected[i])
		}
	}
}

// TestRangeStringRune 测试 range 遍历字符串得到 rune
func TestRangeStringRune(t *testing.T) {
	s := "Go语言"
	runes := []rune{}
	for _, r := range s {
		runes = append(runes, r)
	}
	expected := []rune{'G', 'o', '语', '言'}
	if len(runes) != len(expected) {
		t.Fatalf("rune 个数不匹配：得到 %d，期望 %d", len(runes), len(expected))
	}
	for i, r := range runes {
		if r != expected[i] {
			t.Errorf("runes[%d] = %c，期望 %c", i, r, expected[i])
		}
	}
}

// TestSwitchWithoutBreak 测试 switch 默认 break
func TestSwitchWithoutBreak(t *testing.T) {
	count := 0
	v := 1
	switch v {
	case 1:
		count++
	case 2:
		count++
	}
	// Go 的 switch 默认 break，不会穿透
	if count != 1 {
		t.Errorf("switch 不应穿透，count 应该是 1，实际 %d", count)
	}
}

// TestSwitchFallthrough 测试 fallthrough 穿透
func TestSwitchFallthrough(t *testing.T) {
	count := 0
	v := 1
	switch v {
	case 1:
		count++
		fallthrough
	case 2:
		count++
		fallthrough
	case 3:
		count++
	}
	// fallthrough 强制穿透
	if count != 3 {
		t.Errorf("fallthrough 后 count 应该是 3，实际 %d", count)
	}
}

// TestSwitchNoExpression 测试无表达式 switch
func TestSwitchNoExpression(t *testing.T) {
	x := 25
	result := ""
	switch {
	case x < 0:
		result = "负数"
	case x < 10:
		result = "个位数"
	case x < 100:
		result = "两位数"
	default:
		result = "三位数以上"
	}
	if result != "两位数" {
		t.Errorf("期望「两位数」，实际「%s」", result)
	}
}

// TestBreakLabel 测试标签 break
func TestBreakLabel(t *testing.T) {
	found := false
Outer:
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i*j == 6 {
				found = true
				break Outer
			}
		}
	}
	if !found {
		t.Error("应该找到 i*j == 6")
	}
}

// TestContinueSkipsIteration 测试 continue 跳过迭代
func TestContinueSkipsIteration(t *testing.T) {
	sum := 0
	for i := 1; i <= 10; i++ {
		if i%2 == 0 {
			continue
		}
		sum += i
	}
	// 只累加奇数：1+3+5+7+9 = 25
	if sum != 25 {
		t.Errorf("奇数和应该是 25，实际 %d", sum)
	}
}

// BenchmarkForLoop 基准测试：for 循环
func BenchmarkForLoop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sum := 0
		for j := 0; j < 100; j++ {
			sum += j
		}
		_ = sum
	}
}

// BenchmarkRangeSlice 基准测试：range 遍历切片
func BenchmarkRangeSlice(b *testing.B) {
	nums := make([]int, 100)
	for i := range nums {
		nums[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		for _, v := range nums {
			sum += v
		}
		_ = sum
	}
}

// BenchmarkSwitch 基准测试：switch
func BenchmarkSwitch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := i % 10
		var result string
		switch v {
		case 0:
			result = "zero"
		case 1:
			result = "one"
		case 2:
			result = "two"
		default:
			result = "other"
		}
		_ = result
	}
}

// BenchmarkIfElseChain 基准测试：if-else 链
func BenchmarkIfElseChain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := i % 10
		var result string
		if v == 0 {
			result = "zero"
		} else if v == 1 {
			result = "one"
		} else if v == 2 {
			result = "two"
		} else {
			result = "other"
		}
		_ = result
	}
}
