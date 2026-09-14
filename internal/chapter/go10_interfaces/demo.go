// Package go10_interfaces 演示 Go 语言的接口与类型系统。
//
// 涵盖主题：
//   - 隐式实现
//   - 接口嵌入
//   - any（空接口）
//   - 类型断言与带 ok 形式
//   - type switch
//   - typed nil 陷阱
//   - io.Reader / io.Writer 实战
//   - 小接口设计
//   - 接口值的内存表示
//   - 依赖倒置
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go10_interfaces/
package go10_interfaces

import (
	"bytes"
	"fmt"
	"io"
	"math"
)

// Demo 是第 10 章的入口函数，按小节顺序演示接口与类型系统。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s ==========\n", Chapter, "interfaces")
	fmt.Println("========================================")

	section1_ImplicitImplementation()
	section2_InterfaceEmbedding()
	section3_Any()
	section4_TypeAssertion()
	section5_TypeSwitch()
	section6_TypedNil()
	section7_ReaderWriter()
	section8_SmallInterfaces()
	section9_InterfaceMemory()
	section10_DependencyInversion()

	fmt.Println("========================================")
	fmt.Println("========== 接口与类型系统演示结束 ==========")
	fmt.Println()
}

// --- 10.1 隐式实现 ---

// Shape 是一个几何形状接口。
type Shape interface {
	Area() float64
	Perimeter() float64
}

// Circle 表示圆形。
type Circle struct {
	Radius float64
}

// Area 计算圆的面积。
func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

// Perimeter 计算圆的周长。
func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

// Rectangle 表示矩形。
type Rectangle struct {
	Width, Height float64
}

// Area 计算矩形面积。
func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

// Perimeter 计算矩形周长。
func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

// PrintShapeInfo 接受任何实现了 Shape 接口的类型。
func PrintShapeInfo(s Shape) {
	fmt.Printf("  面积=%.2f, 周长=%.2f\n", s.Area(), s.Perimeter())
}

func section1_ImplicitImplementation() {
	fmt.Println("\n--- 10.1 隐式实现 ---")

	c := Circle{Radius: 5}
	r := Rectangle{Width: 3, Height: 4}

	fmt.Println("Circle:")
	PrintShapeInfo(c)

	fmt.Println("Rectangle:")
	PrintShapeInfo(r)

	// Go 没有 implements 关键字，只要类型实现了接口的所有方法，就自动满足该接口
	fmt.Println("Go 的隐式实现：不需要声明 implements，编译器自动检查")
}

// --- 10.2 接口嵌入 ---

// ReadWriter 通过嵌入 Reader 和 Writer 组合成新接口。
type ReadWriter interface {
	io.Reader
	io.Writer
}

func section2_InterfaceEmbedding() {
	fmt.Println("\n--- 10.2 接口嵌入 ---")

	var rw ReadWriter = &bytes.Buffer{}
	rw.Write([]byte("Hello"))
	data := make([]byte, 5)
	rw.Read(data)
	fmt.Printf("ReadWriter 接口嵌入演示：%s\n", string(data))

	fmt.Println("接口嵌入类似于结构体嵌入，可以组合多个接口形成新接口")
	fmt.Println("io.ReadWriter 就是 io.Reader + io.Writer 的组合")
}

// --- 10.3 any ---

func section3_Any() {
	fmt.Println("\n--- 10.3 any（空接口）---")

	// any 是 interface{} 的别名，Go 1.18 引入
	var x any = 42
	fmt.Printf("x = %v, 类型=%T\n", x, x)

	x = "Hello"
	fmt.Printf("x = %q, 类型=%T\n", x, x)

	x = []int{1, 2, 3}
	fmt.Printf("x = %v, 类型=%T\n", x, x)

	// 空接口丢失了类型信息，需要使用类型断言恢复
	fmt.Println("any 可以存储任意类型，但使用时需要类型断言")
}

// --- 10.4 类型断言 ---

func section4_TypeAssertion() {
	fmt.Println("\n--- 10.4 类型断言 ---")

	var s Shape = Circle{Radius: 3}

	// 带 ok 的形式：安全
	c, ok := s.(Circle)
	fmt.Printf("s.(Circle): ok=%t, radius=%.2f\n", ok, c.Radius)

	// 不带 ok：如果类型不匹配会 panic
	// r := s.(Rectangle) // panic

	// 先判断再断言
	if r, ok := s.(Rectangle); ok {
		fmt.Printf("是 Rectangle: width=%.2f\n", r.Width)
	} else {
		fmt.Println("s 不是 Rectangle")
	}
}

// --- 10.5 type switch ---

func section5_TypeSwitch() {
	fmt.Println("\n--- 10.5 type switch ---")

	shapes := []Shape{
		Circle{Radius: 2},
		Rectangle{Width: 3, Height: 4},
		Circle{Radius: 5},
	}

	for _, s := range shapes {
		switch v := s.(type) {
		case Circle:
			fmt.Printf("Circle, radius=%.2f\n", v.Radius)
		case Rectangle:
			fmt.Printf("Rectangle, width=%.2f, height=%.2f\n", v.Width, v.Height)
		default:
			fmt.Printf("未知类型: %T\n", v)
		}
	}
}

// --- 10.6 typed nil 陷阱 ---

// Printer 是可以打印的接口。
type Printer interface {
	Print()
}

// Document 表示文档。
type Document struct {
	Content string
}

// Print 打印文档内容。
func (d *Document) Print() {
	if d == nil {
		fmt.Println("  (nil Document)")
		return
	}
	fmt.Printf("  Document: %s\n", d.Content)
}

func section6_TypedNil() {
	fmt.Println("\n--- 10.6 typed nil 陷阱 ---")

	var d *Document = nil
	var p Printer = d

	fmt.Printf("d == nil: %t\n", d == nil)
	fmt.Printf("p == nil: %t\n", p == nil)

	// p 不是 nil！它是一个 typed nil（类型为 *Document，值为 nil 的接口值）
	fmt.Println("typed nil：接口值本身不是 nil，只是它持有的具体值为 nil")

	// 可以安全调用，因为 Print 方法内部检查了 nil
	p.Print()

	// 正确的 nil 检查方式
	fmt.Println("判断接口是否为 nil：只能用 reflect.ValueOf(v).IsNil() 或避免存储 nil 指针到接口")
}

// --- 10.7 io.Reader / io.Writer 实战 ---

func section7_ReaderWriter() {
	fmt.Println("\n--- 10.7 io.Reader / io.Writer 实战 ---")

	// bytes.Buffer 同时实现了 io.Reader 和 io.Writer
	var buf bytes.Buffer

	// 写入
	buf.Write([]byte("Hello, Go!"))
	fmt.Printf("写入后 buf: %q\n", buf.String())

	// 读取
	data := make([]byte, 5)
	n, _ := buf.Read(data)
	fmt.Printf("读取 %d 字节: %q, 剩余: %q\n", n, string(data), buf.String())

	// io.Copy 可以连接 Reader 和 Writer
	var dest bytes.Buffer
	src := bytes.NewReader([]byte("Copy me"))
	copied, _ := io.Copy(&dest, src)
	fmt.Printf("io.Copy 复制了 %d 字节: %q\n", copied, dest.String())

	fmt.Println("io.Reader/io.Writer 是 Go 最重要的接口，几乎所有 IO 操作都基于它们")
}

// --- 10.8 小接口设计 ---

// Stringer 是字符串表示接口（与 fmt.Stringer 相同）。
type Stringer interface {
	String() string
}

// Formatter 是格式化接口。
type Formatter interface {
	Format() string
}

// Logger 是日志接口（由小接口组合而成）。
type Logger interface {
	Stringer
	Formatter
}

// SimpleLog 是一个简单的日志条目。
type SimpleLog struct {
	Level   string
	Message string
}

// String 实现 Stringer。
func (l SimpleLog) String() string {
	return fmt.Sprintf("[%s] %s", l.Level, l.Message)
}

// Format 实现 Formatter。
func (l SimpleLog) Format() string {
	return fmt.Sprintf("%s: %s", l.Level, l.Message)
}

func section8_SmallInterfaces() {
	fmt.Println("\n--- 10.8 小接口设计 ---")

	log := SimpleLog{Level: "INFO", Message: "系统启动"}

	var s Stringer = log
	var f Formatter = log

	fmt.Printf("Stringer: %s\n", s.String())
	fmt.Printf("Formatter: %s\n", f.Format())

	fmt.Println("Go 推荐小接口设计：接口越小，实现越容易，组合越灵活")
	fmt.Println("fmt.Stringer 只有一个 String() 方法，是最成功的小接口之一")
}

// --- 10.9 接口值的内存表示 ---

func section9_InterfaceMemory() {
	fmt.Println("\n--- 10.9 接口值的内存表示 ---")

	var s Shape = Circle{Radius: 3}
	fmt.Printf("接口值类型: %T, 值: %v\n", s, s)

	// 接口值内部由两部分组成：(类型指针, 数据指针)
	// 当存储值类型时，数据会被复制到接口值中
	// 当存储指针类型时，接口值保存的是指针

	var p Shape = &Circle{Radius: 5}
	fmt.Printf("接口值（存指针）类型: %T, 值: %v\n", p, p)

	fmt.Println("接口值 = 类型描述符 + 动态值")
	fmt.Println("typed nil 陷阱的原因：接口值的类型描述符不为 nil，只是动态值为 nil")
}

// --- 10.10 依赖倒置 ---

// DataStore 是数据存储接口。
type DataStore interface {
	Get(key string) (string, bool)
	Set(key, value string)
}

// MemoryStore 是内存中的数据存储实现。
type MemoryStore struct {
	data map[string]string
}

// NewMemoryStore 创建内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

// Get 获取值。
func (m *MemoryStore) Get(key string) (string, bool) {
	v, ok := m.data[key]
	return v, ok
}

// Set 设置值。
func (m *MemoryStore) Set(key, value string) {
	m.data[key] = value
}

// UserService 依赖 DataStore 接口，而不是具体实现。
type UserService struct {
	store DataStore
}

// NewUserService 创建用户服务。
func NewUserService(store DataStore) *UserService {
	return &UserService{store: store}
}

// GetUserName 获取用户名。
func (u *UserService) GetUserName(id string) (string, bool) {
	return u.store.Get("user:" + id)
}

func section10_DependencyInversion() {
	fmt.Println("\n--- 10.10 依赖倒置 ---")

	// 使用内存存储
	store := NewMemoryStore()
	store.Set("user:1", "Alice")

	service := NewUserService(store)
	name, ok := service.GetUserName("1")
	fmt.Printf("GetUserName(\"1\"): name=%s, ok=%t\n", name, ok)

	fmt.Println("依赖倒置：高层模块(UserService)依赖抽象(DataStore)，不依赖具体实现")
	fmt.Println("好处：可以方便地替换存储实现（如换成 RedisStore、DBStore），便于测试")
}
