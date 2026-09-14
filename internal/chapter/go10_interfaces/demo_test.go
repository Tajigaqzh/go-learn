package go10_interfaces

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

// TestCircleArea 验证 Circle 实现 Shape 接口。
func TestCircleArea(t *testing.T) {
	c := Circle{Radius: 3}
	want := 28.274333882308138 // Pi * 9
	if got := c.Area(); got != want {
		t.Errorf("Circle.Area() = %f, want %f", got, want)
	}
}

// TestRectangleArea 验证 Rectangle 实现 Shape 接口。
func TestRectangleArea(t *testing.T) {
	r := Rectangle{Width: 3, Height: 4}
	if got := r.Area(); got != 12 {
		t.Errorf("Rectangle.Area() = %f, want 12", got)
	}
	if got := r.Perimeter(); got != 14 {
		t.Errorf("Rectangle.Perimeter() = %f, want 14", got)
	}
}

// TestPrintShapeInfo 验证接口多态。
func TestPrintShapeInfo(t *testing.T) {
	// 只要编译通过，就说明 Circle 和 Rectangle 都实现了 Shape
	var _ Shape = Circle{Radius: 1}
	var _ Shape = Rectangle{Width: 1, Height: 1}
}

// TestTypeAssertion 验证类型断言。
func TestTypeAssertion(t *testing.T) {
	var s Shape = Circle{Radius: 3}

	c, ok := s.(Circle)
	if !ok {
		t.Error("s.(Circle) should succeed")
	}
	if c.Radius != 3 {
		t.Errorf("c.Radius = %f, want 3", c.Radius)
	}

	_, ok = s.(Rectangle)
	if ok {
		t.Error("s.(Rectangle) should fail")
	}
}

// TestTypeSwitch 验证 type switch。
func TestTypeSwitch(t *testing.T) {
	shapes := []Shape{Circle{Radius: 2}, Rectangle{Width: 3, Height: 4}}
	got := make([]string, 0, 2)

	for _, s := range shapes {
		switch v := s.(type) {
		case Circle:
			got = append(got, "circle")
			_ = v.Radius
		case Rectangle:
			got = append(got, "rectangle")
			_ = v.Width
		default:
			t.Fatalf("unexpected type %T", v)
		}
	}

	if got[0] != "circle" || got[1] != "rectangle" {
		t.Errorf("type switch results = %v, want [circle rectangle]", got)
	}
}

// TestTypedNil 验证 typed nil 行为。
func TestTypedNil(t *testing.T) {
	var d *Document = nil
	var p Printer = d

	// p 不是 nil！
	if p == nil {
		t.Error("typed nil interface should not be nil")
	}

	// 但可以安全调用（方法内部处理了 nil）
	p.Print()
}

// TestMemoryStore 验证 DataStore 接口实现。
func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	store.Set("key", "value")

	v, ok := store.Get("key")
	if !ok || v != "value" {
		t.Errorf("Get(\"key\") = (%q, %t), want (\"value\", true)", v, ok)
	}

	_, ok = store.Get("missing")
	if ok {
		t.Error("Get(\"missing\") should return false")
	}
}

// TestUserService 验证依赖倒置。
func TestUserService(t *testing.T) {
	store := NewMemoryStore()
	store.Set("user:1", "Alice")

	service := NewUserService(store)
	name, ok := service.GetUserName("1")
	if !ok || name != "Alice" {
		t.Errorf("GetUserName(\"1\") = (%q, %t), want (\"Alice\", true)", name, ok)
	}
}

// TestReadWriter 验证接口嵌入。
func TestReadWriter(t *testing.T) {
	var rw ReadWriter = &bytes.Buffer{}
	_, err := rw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data := make([]byte, 5)
	n, err := rw.Read(data)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 5 || string(data) != "hello" {
		t.Errorf("Read = %q, want \"hello\"", string(data))
	}
}

// TestIOCopy 验证 io.Reader/io.Writer 实战。
func TestIOCopy(t *testing.T) {
	var dest bytes.Buffer
	src := bytes.NewReader([]byte("Copy me"))
	copied, err := io.Copy(&dest, src)
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}
	if copied != 7 || dest.String() != "Copy me" {
		t.Errorf("io.Copy result = %q, want \"Copy me\"", dest.String())
	}
}

// TestChapterMetadata 保证章节编号和标题跟文档里的章节名一致。
func TestChapterMetadata(t *testing.T) {
	if Chapter != 10 {
		t.Errorf("Chapter = %d, want 10", Chapter)
	}
	if ChapterTitle != "接口与类型系统" {
		t.Errorf("ChapterTitle = %q, want 接口与类型系统", ChapterTitle)
	}
}

// ExampleCircle 是示例测试。
func ExampleCircle() {
	c := Circle{Radius: 1}
	fmt.Printf("%.2f\n", c.Area())
	// Output: 3.14
}
