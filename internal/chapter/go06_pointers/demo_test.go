package go06_pointers

import (
	"fmt"
	"math"
	"testing"
)

// TestSwap 验证指针交换功能。
func TestSwap(t *testing.T) {
	a, b := 3, 5
	Swap(&a, &b)
	if a != 5 || b != 3 {
		t.Errorf("Swap(3, 5) = (%d, %d), want (5, 3)", a, b)
	}
}

// TestIncrement 验证通过指针修改结构体。
func TestIncrement(t *testing.T) {
	c := Counter{value: 10}
	Increment(&c)
	if c.value != 11 {
		t.Errorf("Increment() = %d, want 11", c.value)
	}
}

// TestIncrementNil 验证 nil 指针安全检查。
func TestIncrementNil(t *testing.T) {
	// 不应 panic
	Increment(nil)
}

// TestNewCounter 验证工厂函数。
func TestNewCounter(t *testing.T) {
	c := NewCounter(42)
	if c == nil {
		t.Fatal("NewCounter(42) returned nil")
	}
	if c.value != 42 {
		t.Errorf("NewCounter(42).value = %d, want 42", c.value)
	}
}

// TestVectorLength 验证值接收者方法。
func TestVectorLength(t *testing.T) {
	v := Vector{X: 3, Y: 4}
	got := v.Length()
	want := 5.0 // 3-4-5 三角形
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Vector{3,4}.Length() = %f, want %f", got, want)
	}
}

// TestVectorScale 验证指针接收者方法。
func TestVectorScale(t *testing.T) {
	v := Vector{X: 3, Y: 4}
	v.Scale(2)
	if v.X != 6 || v.Y != 8 {
		t.Errorf("Scale(2) = (%f, %f), want (6, 8)", v.X, v.Y)
	}
}

// TestVectorScaleNil 验证 nil 接收者安全检查。
func TestVectorScaleNil(t *testing.T) {
	var v *Vector
	// 不应 panic
	v.Scale(2)
}

// TestAllocateOnHeap 验证堆分配函数。
func TestAllocateOnHeap(t *testing.T) {
	p := allocateOnHeap()
	if p == nil {
		t.Fatal("allocateOnHeap() returned nil")
	}
	if *p != 42 {
		t.Errorf("*p = %d, want 42", *p)
	}
}

// TestChapterMetadata 保证章节编号和标题跟文档里的章节名一致。
func TestChapterMetadata(t *testing.T) {
	if Chapter != 6 {
		t.Errorf("Chapter = %d, want 6", Chapter)
	}
	if ChapterTitle != "指针、值与内存入门" {
		t.Errorf("ChapterTitle = %q, want 指针、值与内存入门", ChapterTitle)
	}
}

// ExampleSwap 是示例测试。
func ExampleSwap() {
	a, b := 1, 2
	Swap(&a, &b)
	fmt.Println(a, b)
	// Output: 2 1
}
