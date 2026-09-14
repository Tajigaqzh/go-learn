package go05_functions

import (
	"fmt"
	"testing"
)

// TestDivide 验证多返回值函数。
func TestDivide(t *testing.T) {
	q, r := Divide(17, 5)
	if q != 3 || r != 2 {
		t.Errorf("Divide(17, 5) = (%d, %d), want (3, 2)", q, r)
	}
}

// TestSafeDivide 验证带布尔标志的安全除法。
func TestSafeDivide(t *testing.T) {
	_, ok := SafeDivide(10, 0)
	if ok {
		t.Error("SafeDivide(10, 0) should return ok=false")
	}

	result, ok := SafeDivide(10, 2)
	if !ok || result != 5 {
		t.Errorf("SafeDivide(10, 2) = (%d, %t), want (5, true)", result, ok)
	}
}

// TestRectInfo 验证命名返回值。
func TestRectInfo(t *testing.T) {
	area, perimeter := RectInfo(3, 4)
	if area != 12 {
		t.Errorf("RectInfo(3, 4) area = %f, want 12", area)
	}
	if perimeter != 14 {
		t.Errorf("RectInfo(3, 4) perimeter = %f, want 14", perimeter)
	}
}

// TestSum 验证变参函数。
func TestSum(t *testing.T) {
	if Sum() != 0 {
		t.Errorf("Sum() = %d, want 0", Sum())
	}
	if Sum(1, 2, 3) != 6 {
		t.Errorf("Sum(1,2,3) = %d, want 6", Sum(1, 2, 3))
	}
	nums := []int{1, 2, 3, 4}
	if Sum(nums...) != 10 {
		t.Errorf("Sum(nums...) = %d, want 10", Sum(nums...))
	}
}

// TestApplyOperation 验证高阶函数。
func TestApplyOperation(t *testing.T) {
	add := func(a, b int) int { return a + b }
	result := ApplyOperation(3, 4, add)
	if result != 7 {
		t.Errorf("ApplyOperation(3, 4, add) = %d, want 7", result)
	}
}

// TestMakeOperation 验证函数作为返回值。
func TestMakeOperation(t *testing.T) {
	mul := MakeOperation("mul")
	if mul(3, 4) != 12 {
		t.Errorf("MakeOperation(\"mul\")(3,4) = %d, want 12", mul(3, 4))
	}
}

// TestMakeMultiplier 验证闭包捕获外部变量。
func TestMakeMultiplier(t *testing.T) {
	double := MakeMultiplier(2)
	if double(5) != 10 {
		t.Errorf("MakeMultiplier(2)(5) = %d, want 10", double(5))
	}

	triple := MakeMultiplier(3)
	if triple(5) != 15 {
		t.Errorf("MakeMultiplier(3)(5) = %d, want 15", triple(5))
	}
}

// TestMakeCounter 验证闭包修改外部变量。
func TestMakeCounter(t *testing.T) {
	counter := MakeCounter()
	if counter() != 1 {
		t.Errorf("counter() = %d, want 1", counter())
	}
	if counter() != 2 {
		t.Errorf("counter() = %d, want 2", counter())
	}
	if counter() != 3 {
		t.Errorf("counter() = %d, want 3", counter())
	}
}

// TestFactorial 验证递归阶乘。
func TestFactorial(t *testing.T) {
	if Factorial(5) != 120 {
		t.Errorf("Factorial(5) = %d, want 120", Factorial(5))
	}
	if Factorial(0) != 1 {
		t.Errorf("Factorial(0) = %d, want 1", Factorial(0))
	}
}

// TestFibonacci 验证斐波那契数列。
func TestFibonacci(t *testing.T) {
	if Fibonacci(10) != 55 {
		t.Errorf("Fibonacci(10) = %d, want 55", Fibonacci(10))
	}
}

// TestFactorialLoop 验证循环版阶乘。
func TestFactorialLoop(t *testing.T) {
	if FactorialLoop(5) != 120 {
		t.Errorf("FactorialLoop(5) = %d, want 120", FactorialLoop(5))
	}
}

// TestDeferNamedReturn 验证 defer 修改命名返回值。
func TestDeferNamedReturn(t *testing.T) {
	if deferWithNamedReturn() != 2 {
		t.Errorf("deferWithNamedReturn() = %d, want 2", deferWithNamedReturn())
	}
}

// TestDeferWithoutNamedReturn 验证 defer 不能修改非命名返回值。
func TestDeferWithoutNamedReturn(t *testing.T) {
	if deferWithoutNamedReturn() != 1 {
		t.Errorf("deferWithoutNamedReturn() = %d, want 1", deferWithoutNamedReturn())
	}
}

// TestChapterMetadata 保证章节编号和标题跟文档里的章节名一致。
func TestChapterMetadata(t *testing.T) {
	if Chapter != 5 {
		t.Errorf("Chapter = %d, want 5", Chapter)
	}
	if ChapterTitle != "函数与闭包" {
		t.Errorf("ChapterTitle = %q, want 函数与闭包", ChapterTitle)
	}
}

// ExampleSum 是示例测试。
func ExampleSum() {
	fmt.Println(Sum(1, 2, 3))
	// Output: 6
}
