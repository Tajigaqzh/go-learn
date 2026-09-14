package go09_structs_methods

import (
	"reflect"
	"testing"
	"unsafe"
)

// TestStructBasics 测试结构体基本操作
func TestStructBasics(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	// 零值
	var p1 Person
	if p1.Name != "" || p1.Age != 0 {
		t.Errorf("零值不正确")
	}

	// 字面量初始化
	p2 := Person{"Alice", 25}
	if p2.Name != "Alice" || p2.Age != 25 {
		t.Errorf("字面量初始化失败")
	}

	// 部分初始化
	p3 := Person{Name: "Bob"}
	if p3.Name != "Bob" || p3.Age != 0 {
		t.Errorf("部分初始化失败")
	}
}

// TestEmbedding 测试嵌入
func TestEmbedding(t *testing.T) {
	type Address struct {
		City string
	}

	type Person struct {
		Name string
		Address
	}

	p := Person{
		Name:    "Alice",
		Address: Address{City: "Beijing"},
	}

	// 直接访问嵌入字段
	if p.City != "Beijing" {
		t.Errorf("嵌入字段访问失败")
	}

	// 通过类型名访问
	if p.Address.City != "Beijing" {
		t.Errorf("通过类型名访问失败")
	}
}

// TestFieldTags 测试字段标签
func TestFieldTags(t *testing.T) {
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	typ := reflect.TypeOf(User{})
	field, _ := typ.FieldByName("ID")
	tag := field.Tag.Get("json")

	if tag != "id" {
		t.Errorf("字段标签读取失败，期望 'id'，实际 '%s'", tag)
	}
}

// TestComparability 测试可比较性
func TestComparability(t *testing.T) {
	type Point struct {
		X, Y int
	}

	p1 := Point{1, 2}
	p2 := Point{1, 2}
	p3 := Point{3, 4}

	if p1 != p2 {
		t.Errorf("相同的结构体应该相等")
	}

	if p1 == p3 {
		t.Errorf("不同的结构体不应该相等")
	}
}

// TestMethods 测试方法
func TestMethods(t *testing.T) {
	type Counter struct {
		value int
	}

	// 值接收者
	increment := func(c Counter) Counter {
		c.value++
		return c
	}

	c1 := Counter{10}
	c2 := increment(c1)

	if c1.value != 10 {
		t.Errorf("值接收者不应该修改原值")
	}
	if c2.value != 11 {
		t.Errorf("返回的新值应该是 11")
	}
}

// TestPointerReceiver 测试指针接收者
func TestPointerReceiver(t *testing.T) {
	type Counter struct {
		value int
	}

	type CounterWithMethod struct {
		value int
	}

	increment := func(c *CounterWithMethod) {
		c.value++
	}

	c := &CounterWithMethod{10}
	increment(c)

	if c.value != 11 {
		t.Errorf("指针接收者应该修改原值，期望 11，实际 %d", c.value)
	}
}

// TestMethodPromotion 测试方法提升
func TestMethodPromotion(t *testing.T) {
	type Engine struct {
		power int
	}

	start := func(e Engine) int {
		return e.power
	}

	type Car struct {
		Engine
		brand string
	}

	c := Car{
		Engine: Engine{power: 200},
		brand:  "Toyota",
	}

	// 模拟方法提升
	power := start(c.Engine)
	if power != 200 {
		t.Errorf("方法提升失败")
	}
}

// TestMemoryAlignment 测试内存对齐
func TestMemoryAlignment(t *testing.T) {
	type BadOrder struct {
		a bool  // 1 字节
		b int64 // 8 字节
		c bool  // 1 字节
	}

	type GoodOrder struct {
		b int64 // 8 字节
		a bool  // 1 字节
		c bool  // 1 字节
	}

	badSize := unsafe.Sizeof(BadOrder{})
	goodSize := unsafe.Sizeof(GoodOrder{})

	// BadOrder 因为对齐会有填充
	if badSize <= goodSize {
		t.Logf("BadOrder: %d 字节, GoodOrder: %d 字节", badSize, goodSize)
	}

	// GoodOrder 应该更紧凑（或相等）
	if goodSize > 16 {
		t.Errorf("GoodOrder 大小不应该超过 16 字节，实际 %d", goodSize)
	}
}

// TestStructCopy 测试结构体复制
func TestStructCopy(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p1 := Person{"Alice", 25}
	p2 := p1
	p2.Name = "Bob"

	if p1.Name != "Alice" {
		t.Errorf("结构体复制后修改不应该影响原结构体")
	}
	if p2.Name != "Bob" {
		t.Errorf("复制后的结构体修改失败")
	}
}

// TestAnonymousStruct 测试匿名结构体
func TestAnonymousStruct(t *testing.T) {
	point := struct {
		X, Y int
	}{10, 20}

	if point.X != 10 || point.Y != 20 {
		t.Errorf("匿名结构体初始化失败")
	}
}

// TestNestedStruct 测试嵌套结构体
func TestNestedStruct(t *testing.T) {
	type Address struct {
		City    string
		Country string
	}

	type Person struct {
		Name    string
		Address Address
	}

	p := Person{
		Name: "Alice",
		Address: Address{
			City:    "Beijing",
			Country: "China",
		},
	}

	if p.Address.City != "Beijing" {
		t.Errorf("嵌套结构体访问失败")
	}
}

// TestStructPointer 测试结构体指针
func TestStructPointer(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p1 := &Person{"Alice", 25}
	p2 := p1
	p2.Name = "Bob"

	if p1.Name != "Bob" {
		t.Errorf("结构体指针应该共享数据")
	}
}

// TestZeroValueStruct 测试零值结构体
func TestZeroValueStruct(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	var c Config
	if c.Host != "" {
		t.Errorf("字符串字段零值应该是空串")
	}
	if c.Port != 0 {
		t.Errorf("整数字段零值应该是 0")
	}
}

// BenchmarkStructCopy 测试结构体复制性能
func BenchmarkStructCopy(b *testing.B) {
	type Large struct {
		data [1024]byte
	}

	src := Large{}

	b.Run("ValueCopy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = src
		}
	})

	b.Run("PointerCopy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = &src
		}
	})
}

// BenchmarkMethodCall 测试方法调用性能
func BenchmarkMethodCall(b *testing.B) {
	type Counter struct {
		value int
	}

	valueMethod := func(c Counter) int {
		return c.value
	}

	pointerMethod := func(c *Counter) int {
		return c.value
	}

	c := Counter{value: 100}

	b.Run("ValueReceiver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = valueMethod(c)
		}
	})

	b.Run("PointerReceiver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = pointerMethod(&c)
		}
	})
}
