// Package go09_structs_methods 演示 Go 的结构体与方法。
package go09_structs_methods

import (
	"fmt"
	"reflect"
	"unsafe"
)

// Demo 运行第 9 章的所有示例。
func Demo() {
	fmt.Println("========== go09_structs_methods: 结构体与方法 ==========")

	fmt.Println("\n--- 1. 结构体定义与初始化 ---")
	demoStructBasics()

	fmt.Println("\n--- 2. 匿名字段与嵌入 ---")
	demoEmbedding()

	fmt.Println("\n--- 3. 字段标签 ---")
	demoFieldTags()

	fmt.Println("\n--- 4. 结构体可比较性 ---")
	demoComparability()

	fmt.Println("\n--- 5. 方法定义 ---")
	demoMethods()

	fmt.Println("\n--- 6. 值接收者 vs 指针接收者 ---")
	demoReceivers()

	fmt.Println("\n--- 7. 方法值与方法表达式 ---")
	demoMethodValues()

	fmt.Println("\n--- 8. 嵌入与方法提升 ---")
	demoMethodPromotion()

	fmt.Println("\n--- 9. 构造函数惯例 ---")
	demoConstructors()

	fmt.Println("\n--- 10. 组合优于继承 ---")
	demoComposition()

	fmt.Println("\n--- 11. 内存对齐 ---")
	demoMemoryAlignment()

	fmt.Println("\n========== 结构体与方法演示结束 ==========")
}

// demoStructBasics 演示结构体定义与初始化
func demoStructBasics() {
	// 定义结构体类型
	type Person struct {
		Name string
		Age  int
	}

	// 零值初始化
	var p1 Person
	fmt.Printf("零值: %+v\n", p1)

	// 字段赋值
	p1.Name = "Alice"
	p1.Age = 25
	fmt.Printf("赋值后: %+v\n", p1)

	// 字面量初始化（按字段顺序）
	p2 := Person{"Bob", 30}
	fmt.Printf("字面量（按顺序）: %+v\n", p2)

	// 字面量初始化（按字段名）
	p3 := Person{
		Name: "Carol",
		Age:  28,
	}
	fmt.Printf("字面量（按字段名）: %+v\n", p3)

	// 部分字段初始化，其他字段为零值
	p4 := Person{Name: "David"}
	fmt.Printf("部分初始化: %+v\n", p4)

	// 匿名结构体
	temp := struct {
		X, Y int
	}{10, 20}
	fmt.Printf("匿名结构体: %+v\n", temp)
}

// demoEmbedding 演示匿名字段与嵌入
func demoEmbedding() {
	// 嵌入结构体
	type Address struct {
		City    string
		Country string
	}

	type Person struct {
		Name    string
		Age     int
		Address // 匿名字段（嵌入）
	}

	p := Person{
		Name: "Alice",
		Age:  25,
		Address: Address{
			City:    "Beijing",
			Country: "China",
		},
	}

	// 可以直接访问嵌入类型的字段
	fmt.Printf("Name: %s, City: %s\n", p.Name, p.City)

	// 也可以通过类型名访问
	fmt.Printf("Country: %s\n", p.Address.Country)

	// 嵌入多个类型
	type Contact struct {
		Email string
		Phone string
	}

	type Employee struct {
		Person  // 嵌入 Person
		Contact // 嵌入 Contact
		Salary  int
	}

	e := Employee{
		Person: Person{Name: "Bob", Age: 30, Address: Address{City: "Shanghai", Country: "China"}},
		Contact: Contact{
			Email: "bob@example.com",
			Phone: "123-4567",
		},
		Salary: 50000,
	}

	fmt.Printf("Employee: %s, Email: %s, City: %s\n", e.Name, e.Email, e.City)
}

// demoFieldTags 演示字段标签
func demoFieldTags() {
	type User struct {
		ID       int    `json:"id" db:"user_id"`
		Username string `json:"username" db:"user_name"`
		Password string `json:"-"` // "-" 表示 JSON 序列化时忽略
		Email    string `json:"email,omitempty" db:"email"`
	}

	u := User{ID: 1, Username: "alice", Password: "secret", Email: ""}

	// 通过反射读取字段标签
	t := reflect.TypeOf(u)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")
		fmt.Printf("字段 %s: json=%q, db=%q\n", field.Name, jsonTag, dbTag)
	}
}

// demoComparability 演示结构体可比较性
func demoComparability() {
	type Point struct {
		X, Y int
	}

	p1 := Point{1, 2}
	p2 := Point{1, 2}
	p3 := Point{3, 4}

	fmt.Printf("p1 == p2: %v\n", p1 == p2)
	fmt.Printf("p1 == p3: %v\n", p1 == p3)

	// 包含不可比较字段的结构体不可比较
	type Container struct {
		Data []int // 切片不可比较
	}

	// c1 := Container{Data: []int{1, 2}}
	// c2 := Container{Data: []int{1, 2}}
	// fmt.Println(c1 == c2) // 编译错误：invalid operation

	fmt.Println("包含切片、map、函数的结构体不可比较")
}

// Point 是一个二维点
type Point struct {
	X, Y float64
}

// Distance 计算到原点的距离（值接收者）
func (p Point) Distance() float64 {
	return p.X*p.X + p.Y*p.Y
}

// Scale 缩放点（指针接收者，修改原值）
func (p *Point) Scale(factor float64) {
	p.X *= factor
	p.Y *= factor
}

// demoMethods 演示方法定义
func demoMethods() {
	p := Point{3, 4}
	fmt.Printf("点: %+v\n", p)

	// 调用值接收者方法
	dist := p.Distance()
	fmt.Printf("到原点距离平方: %.2f\n", dist)

	// 调用指针接收者方法
	p.Scale(2)
	fmt.Printf("缩放后: %+v\n", p)
}

// Counter 是一个计数器
type Counter struct {
	value int
}

// Increment 值接收者（不会修改原值）
func (c Counter) Increment() {
	c.value++
	fmt.Printf("  值接收者内: %d\n", c.value)
}

// IncrementPtr 指针接收者（修改原值）
func (c *Counter) IncrementPtr() {
	c.value++
	fmt.Printf("  指针接收者内: %d\n", c.value)
}

// Value 返回当前值
func (c Counter) Value() int {
	return c.value
}

// demoReceivers 演示值接收者 vs 指针接收者
func demoReceivers() {
	c := Counter{value: 10}
	fmt.Printf("初始: %d\n", c.Value())

	// 值接收者不会修改原值
	c.Increment()
	fmt.Printf("调用 Increment 后: %d（未改变）\n", c.Value())

	// 指针接收者会修改原值
	c.IncrementPtr()
	fmt.Printf("调用 IncrementPtr 后: %d（已改变）\n", c.Value())

	// Go 会自动转换：值调用指针接收者方法
	c.IncrementPtr() // 等价于 (&c).IncrementPtr()
	fmt.Printf("再次调用: %d\n", c.Value())

	// 指针调用值接收者方法
	p := &c
	p.Increment() // 等价于 (*p).Increment()
	fmt.Printf("指针调用值接收者: %d\n", c.Value())
}

// demoMethodValues 演示方法值与方法表达式
func demoMethodValues() {
	p := Point{3, 4}

	// 方法值（Method Value）：绑定了接收者
	distFunc := p.Distance
	fmt.Printf("方法值调用: %.2f\n", distFunc())

	// 方法表达式（Method Expression）：未绑定接收者
	distExpr := Point.Distance
	fmt.Printf("方法表达式调用: %.2f\n", distExpr(p))

	// 指针接收者的方法表达式
	scaleExpr := (*Point).Scale
	scaleExpr(&p, 2)
	fmt.Printf("缩放后: %+v\n", p)
}

// Engine 是引擎
type Engine struct {
	Power int
}

// Start 启动引擎
func (e Engine) Start() {
	fmt.Println("  引擎启动，功率:", e.Power)
}

// Car 是汽车
type Car struct {
	Engine // 嵌入 Engine
	Brand  string
}

// demoMethodPromotion 演示嵌入与方法提升
func demoMethodPromotion() {
	c := Car{
		Engine: Engine{Power: 200},
		Brand:  "Toyota",
	}

	// 可以直接调用嵌入类型的方法
	c.Start()

	// 也可以通过类型名调用
	c.Engine.Start()

	fmt.Println("方法提升：嵌入类型的方法自动提升到外层类型")
}

// NewPerson 构造函数
func NewPerson(name string, age int) *Person {
	return &Person{
		Name: name,
		Age:  age,
	}
}

// Person 是人
type Person struct {
	Name string
	Age  int
}

// NewPersonWithDefaults 带默认值的构造函数
func NewPersonWithDefaults() *Person {
	return &Person{
		Name: "Unknown",
		Age:  0,
	}
}

// demoConstructors 演示构造函数惯例
func demoConstructors() {
	p1 := NewPerson("Alice", 25)
	fmt.Printf("NewPerson: %+v\n", p1)

	p2 := NewPersonWithDefaults()
	fmt.Printf("NewPersonWithDefaults: %+v\n", p2)

	fmt.Println("惯例：构造函数命名为 NewXxx，返回指针")
}

// Logger 是日志接口
type Logger interface {
	Log(message string)
}

// ConsoleLogger 是控制台日志
type ConsoleLogger struct{}

// Log 实现 Logger 接口
func (c ConsoleLogger) Log(message string) {
	fmt.Println("  [Console]", message)
}

// Service 是服务
type Service struct {
	logger Logger
}

// NewService 构造函数（依赖注入）
func NewService(logger Logger) *Service {
	return &Service{logger: logger}
}

// DoWork 执行工作
func (s *Service) DoWork() {
	s.logger.Log("Service is working")
}

// demoComposition 演示组合优于继承
func demoComposition() {
	logger := ConsoleLogger{}
	service := NewService(logger)
	service.DoWork()

	fmt.Println("组合：通过接口注入依赖，而不是继承")
}

// demoMemoryAlignment 演示内存对齐
func demoMemoryAlignment() {
	// 字段顺序影响结构体大小
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

	fmt.Printf("BadOrder 大小: %d 字节\n", unsafe.Sizeof(BadOrder{}))
	fmt.Printf("GoodOrder 大小: %d 字节\n", unsafe.Sizeof(GoodOrder{}))

	// 查看字段偏移
	fmt.Println("\nBadOrder 字段偏移:")
	fmt.Printf("  a 偏移: %d\n", unsafe.Offsetof(BadOrder{}.a))
	fmt.Printf("  b 偏移: %d\n", unsafe.Offsetof(BadOrder{}.b))
	fmt.Printf("  c 偏移: %d\n", unsafe.Offsetof(BadOrder{}.c))

	fmt.Println("\nGoodOrder 字段偏移:")
	fmt.Printf("  b 偏移: %d\n", unsafe.Offsetof(GoodOrder{}.b))
	fmt.Printf("  a 偏移: %d\n", unsafe.Offsetof(GoodOrder{}.a))
	fmt.Printf("  c 偏移: %d\n", unsafe.Offsetof(GoodOrder{}.c))

	fmt.Println("\n建议：把大字段放在前面，减少内存浪费")
}
