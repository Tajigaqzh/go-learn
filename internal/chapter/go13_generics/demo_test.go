package go13_generics

import (
	"slices"
	"testing"
)

func TestMin(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want int
	}{
		{"单元素", []int{42}, 42},
		{"升序", []int{1, 2, 3}, 1},
		{"降序", []int{3, 2, 1}, 1},
		{"重复最小值", []int{5, 1, 3, 1, 2}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Min(tt.in); got != tt.want {
				t.Errorf("Min(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestMinFloat(t *testing.T) {
	in := []float64{3.14, 1.41, 2.71}
	want := 1.41
	if got := Min(in); got != want {
		t.Errorf("Min(%v) = %v, want %v", in, got, want)
	}
}

func TestUnique(t *testing.T) {
	ints := []int{1, 2, 3, 2, 1}
	got := Unique(ints)
	want := []int{1, 2, 3}
	if !slices.Equal(got, want) {
		t.Errorf("Unique(%v) = %v, want %v", ints, got, want)
	}

	strs := []string{"go", "rust", "go"}
	gotStr := Unique(strs)
	wantStr := []string{"go", "rust"}
	if !slices.Equal(gotStr, wantStr) {
		t.Errorf("Unique(%v) = %v, want %v", strs, gotStr, wantStr)
	}
}

func TestSumNumbers(t *testing.T) {
	ints := []int{1, 2, 3}
	if got := SumNumbers(ints); got != 6 {
		t.Errorf("SumNumbers(%v) = %v, want 6", ints, got)
	}

	floats := []float64{1.1, 2.2, 3.3}
	want := 6.6
	if got := SumNumbers(floats); got < want-0.01 || got > want+0.01 {
		t.Errorf("SumNumbers(%v) = %v, want ~%v", floats, got, want)
	}
}

func TestSumIntegers(t *testing.T) {
	type MyInt int
	myInts := []MyInt{10, 20, 30}
	if got := SumIntegers(myInts); got != 60 {
		t.Errorf("SumIntegers(%v) = %v, want 60", myInts, got)
	}
}

func TestStack(t *testing.T) {
	s := NewStack[int]()

	// 空栈弹出应该返回 false
	if _, ok := s.Pop(); ok {
		t.Error("Pop from empty stack should return false")
	}

	// 压入元素
	s.Push(10)
	s.Push(20)
	s.Push(30)

	// 按 LIFO 顺序弹出
	if v, ok := s.Pop(); !ok || v != 30 {
		t.Errorf("Pop() = %v, %t; want 30, true", v, ok)
	}
	if v, ok := s.Pop(); !ok || v != 20 {
		t.Errorf("Pop() = %v, %t; want 20, true", v, ok)
	}
	if v, ok := s.Pop(); !ok || v != 10 {
		t.Errorf("Pop() = %v, %t; want 10, true", v, ok)
	}
	if _, ok := s.Pop(); ok {
		t.Error("Pop from empty stack should return false")
	}
}

func TestOldest(t *testing.T) {
	people := []Person{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 35},
	}
	got := Oldest(people)
	if got.Name != "Charlie" || got.Age != 35 {
		t.Errorf("Oldest(%v) = %+v, want Charlie/35", people, got)
	}
}

func TestMinInt(t *testing.T) {
	in := []int{3, 1, 4, 1, 5}
	if got := MinInt(in); got != 1 {
		t.Errorf("MinInt(%v) = %v, want 1", in, got)
	}
}

func TestMinString(t *testing.T) {
	in := []string{"go", "rust", "python"}
	if got := MinString(in); got != "go" {
		t.Errorf("MinString(%v) = %v, want go", in, got)
	}
}
