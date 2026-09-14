# 第 9 章 · 结构体与方法

前面几章我们学习了 Go 的基本类型、控制流、函数、指针、切片和 map。这些是构建程序的基础工具，但实际编程中，我们需要把**相关的数据组织在一起**，并为这些数据定义**行为**。这就是结构体（struct）和方法（method）的作用。

结构体是 Go 中最重要的复合类型，用于将不同类型的字段组合成一个逻辑单元。方法是绑定到特定类型上的函数，让类型具有行为。Go 没有传统的类和继承，而是通过**组合（composition）** 和**接口（interface）** 来实现代码复用和多态。

本章配套代码在 `internal/chapter/go09_structs_methods/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 9.1 结构体定义与初始化

结构体用 `type` 和 `struct` 关键字定义：

```go
type Person struct {
    Name string
    Age  int
}
```

**零值初始化**：

```go
var p1 Person
fmt.Printf("零值: %+v\n", p1)
```

**输出：**
```
零值: {Name: Age:0}
```

结构体的零值是所有字段都为其类型的零值。字符串字段是空串，整数字段是 0。

**字段赋值**：

```go
p1.Name = "Alice"
p1.Age = 25
fmt.Printf("赋值后: %+v\n", p1)
```

**输出：**
```
赋值后: {Name:Alice Age:25}
```

**字面量初始化（按字段顺序）**：

```go
p2 := Person{"Bob", 30}
fmt.Printf("字面量（按顺序）: %+v\n", p2)
```

**输出：**
```
字面量（按顺序）: {Name:Bob Age:30}
```

这种方式简洁，但**不推荐**，因为字段顺序变化时代码会出错。

**字面量初始化（按字段名）**：

```go
p3 := Person{
    Name: "Carol",
    Age:  28,
}
fmt.Printf("字面量（按字段名）: %+v\n", p3)
```

**输出：**
```
字面量（按字段名）: {Name:Carol Age:28}
```

这是**推荐的写法**，字段顺序可以任意，增加字段时不会破坏现有代码。

**部分字段初始化**：

```go
p4 := Person{Name: "David"}
fmt.Printf("部分初始化: %+v\n", p4)
```

**输出：**
```
部分初始化: {Name:David Age:0}
```

未初始化的字段使用零值。

**匿名结构体**：

```go
temp := struct {
    X, Y int
}{10, 20}
fmt.Printf("匿名结构体: %+v\n", temp)
```

**输出：**
```
匿名结构体: {X:10 Y:20}
```

匿名结构体常用于临时数据，比如 JSON 解析、测试数据等。

## 9.2 匿名字段与嵌入

结构体可以嵌入其他类型作为**匿名字段**，嵌入类型的字段和方法会**提升**到外层类型：

```go
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
```

**输出：**
```
Name: Alice, City: Beijing
Country: China
```

`p.City` 是 `p.Address.City` 的简写，这叫**字段提升**。

**嵌入多个类型**：

```go
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
```

**输出：**
```
Employee: Bob, Email: bob@example.com, City: Shanghai
```

**嵌入 vs 命名字段**：

- **嵌入**（匿名字段）：字段和方法提升到外层，更像「是一个」的关系
- **命名字段**：必须通过字段名访问，更像「有一个」的关系

Go 推崇**组合优于继承**，嵌入是 Go 实现类似继承效果的方式，但本质上仍然是组合。

## 9.3 字段标签

字段标签（field tag）是附加在结构体字段后面的元数据字符串，常用于 JSON 序列化、数据库映射、验证等：

```go
type User struct {
    ID       int    `json:"id" db:"user_id"`
    Username string `json:"username" db:"user_name"`
    Password string `json:"-"` // "-" 表示 JSON 序列化时忽略
    Email    string `json:"email,omitempty" db:"email"`
}
```

**通过反射读取字段标签**：

```go
u := User{ID: 1, Username: "alice", Password: "secret", Email: ""}

t := reflect.TypeOf(u)
for i := 0; i < t.NumField(); i++ {
    field := t.Field(i)
    jsonTag := field.Tag.Get("json")
    dbTag := field.Tag.Get("db")
    fmt.Printf("字段 %s: json=%q, db=%q\n", field.Name, jsonTag, dbTag)
}
```

**输出：**
```
字段 ID: json="id", db="user_id"
字段 Username: json="username", db="user_name"
字段 Password: json="-", db=""
字段 Email: json="email,omitempty", db="email"
```

**常用标签**：

- `json:"name"`：JSON 序列化时的字段名
- `json:"-"`：忽略该字段
- `json:"name,omitempty"`：字段为零值时忽略
- `db:"column_name"`：数据库列名
- `validate:"required,email"`：参数校验规则

字段标签是字符串，解析由库决定。`encoding/json`、`database/sql` 等标准库和第三方库会读取标签。

## 9.4 结构体可比较性

如果结构体的**所有字段都是可比较类型**，那么结构体本身也是可比较的：

```go
type Point struct {
    X, Y int
}

p1 := Point{1, 2}
p2 := Point{1, 2}
p3 := Point{3, 4}

fmt.Printf("p1 == p2: %v\n", p1 == p2)
fmt.Printf("p1 == p3: %v\n", p1 == p3)
```

**输出：**
```
p1 == p2: true
p1 == p3: false
```

比较规则：逐字段比较，所有字段都相等则结构体相等。

**包含不可比较字段的结构体不可比较**：

```go
type Container struct {
    Data []int // 切片不可比较
}

c1 := Container{Data: []int{1, 2}}
c2 := Container{Data: []int{1, 2}}
// fmt.Println(c1 == c2) // 编译错误：invalid operation
```

切片、map、函数不可比较，包含它们的结构体也不可比较。可以用 `reflect.DeepEqual` 比较，但性能较差。

**可比较的结构体可以作为 map key**：

```go
m := make(map[Point]string)
m[Point{1, 2}] = "origin nearby"
fmt.Println(m[Point{1, 2}]) // "origin nearby"
```

## 9.5 方法定义

方法是绑定到特定类型上的函数。语法是在 `func` 和函数名之间加上**接收者（receiver）**：

```go
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
```

**调用方法**：

```go
p := Point{3, 4}
fmt.Printf("点: %+v\n", p)

// 调用值接收者方法
dist := p.Distance()
fmt.Printf("到原点距离平方: %.2f\n", dist)

// 调用指针接收者方法
p.Scale(2)
fmt.Printf("缩放后: %+v\n", p)
```

**输出：**
```
点: {X:3 Y:4}
到原点距离平方: 25.00
缩放后: {X:6 Y:8}
```

**方法 vs 函数**：

- 方法有接收者，语法是 `receiver.Method()`
- 函数没有接收者，语法是 `Package.Function()`
- 方法可以访问接收者的字段，函数不行

**方法只能在同一个包内定义**：不能为外部包的类型定义方法（除非用类型别名）。

## 9.6 值接收者 vs 指针接收者

**值接收者**：方法接收类型的副本，**不会修改原值**：

```go
type Counter struct {
    value int
}

// Increment 值接收者（不会修改原值）
func (c Counter) Increment() {
    c.value++
    fmt.Printf("  值接收者内: %d\n", c.value)
}

// Value 返回当前值
func (c Counter) Value() int {
    return c.value
}

c := Counter{value: 10}
fmt.Printf("初始: %d\n", c.Value())

c.Increment()
fmt.Printf("调用 Increment 后: %d（未改变）\n", c.Value())
```

**输出：**
```
初始: 10
  值接收者内: 11
调用 Increment 后: 10（未改变）
```

**指针接收者**：方法接收指针，**会修改原值**：

```go
// IncrementPtr 指针接收者（修改原值）
func (c *Counter) IncrementPtr() {
    c.value++
    fmt.Printf("  指针接收者内: %d\n", c.value)
}

c.IncrementPtr()
fmt.Printf("调用 IncrementPtr 后: %d（已改变）\n", c.Value())
```

**输出：**
```
  指针接收者内: 11
调用 IncrementPtr 后: 11（已改变）
```

**Go 会自动转换**：

```go
// 值调用指针接收者方法（自动取地址）
c.IncrementPtr() // 等价于 (&c).IncrementPtr()

// 指针调用值接收者方法（自动解引用）
p := &c
p.Increment() // 等价于 (*p).Increment()
```

**什么时候用哪个**：

- **指针接收者**：
  - 需要修改接收者
  - 接收者很大，复制代价高
  - 保持一致性：如果某个方法用了指针接收者，其他方法也应该用
  
- **值接收者**：
  - 不需要修改接收者
  - 接收者很小（几个字段）
  - 接收者是不可变类型（如 `time.Time`）

**经验法则**：如果不确定，用指针接收者。

## 9.7 方法值与方法表达式

**方法值（Method Value）**：绑定了接收者的方法，可以当作函数使用：

```go
p := Point{3, 4}

// 方法值：绑定了接收者
distFunc := p.Distance
fmt.Printf("方法值调用: %.2f\n", distFunc())
```

**输出：**
```
方法值调用: 25.00
```

`distFunc` 记住了 `p`，调用 `distFunc()` 等价于 `p.Distance()`。

**方法表达式（Method Expression）**：未绑定接收者的方法，第一个参数是接收者：

```go
// 方法表达式：未绑定接收者
distExpr := Point.Distance
fmt.Printf("方法表达式调用: %.2f\n", distExpr(p))
```

**输出：**
```
方法表达式调用: 25.00
```

`distExpr` 的类型是 `func(Point) float64`，需要显式传递接收者。

**指针接收者的方法表达式**：

```go
scaleExpr := (*Point).Scale
scaleExpr(&p, 2)
fmt.Printf("缩放后: %+v\n", p)
```

**输出：**
```
缩放后: {X:6 Y:8}
```

方法值和方法表达式在高阶函数、回调、goroutine 中很有用。

## 9.8 嵌入与方法提升

嵌入类型的方法会**提升**到外层类型：

```go
type Engine struct {
    Power int
}

// Start 启动引擎
func (e Engine) Start() {
    fmt.Println("  引擎启动，功率:", e.Power)
}

type Car struct {
    Engine // 嵌入 Engine
    Brand  string
}

c := Car{
    Engine: Engine{Power: 200},
    Brand:  "Toyota",
}

// 可以直接调用嵌入类型的方法
c.Start()

// 也可以通过类型名调用
c.Engine.Start()
```

**输出：**
```
  引擎启动，功率: 200
  引擎启动，功率: 200
```

**方法提升规则**：

- 嵌入类型的方法自动提升到外层类型
- 如果外层类型定义了同名方法，外层方法**覆盖**嵌入类型的方法
- 如果嵌入多个类型，都有同名方法，**编译错误**（歧义）

**方法提升与接口**：

如果嵌入类型实现了某个接口，外层类型**自动实现**该接口（通过方法提升）。

## 9.9 构造函数惯例

Go 没有构造函数，但惯例是定义 `NewXxx` 函数：

```go
type Person struct {
    Name string
    Age  int
}

// NewPerson 构造函数
func NewPerson(name string, age int) *Person {
    return &Person{
        Name: name,
        Age:  age,
    }
}

p1 := NewPerson("Alice", 25)
fmt.Printf("NewPerson: %+v\n", p1)
```

**输出：**
```
NewPerson: &{Name:Alice Age:25}
```

**为什么返回指针**：

- 避免复制大结构体
- 接收者方法通常是指针接收者，返回指针更一致
- 可以在构造函数中做初始化、校验、设置默认值

**带默认值的构造函数**：

```go
// NewPersonWithDefaults 带默认值的构造函数
func NewPersonWithDefaults() *Person {
    return &Person{
        Name: "Unknown",
        Age:  0,
    }
}

p2 := NewPersonWithDefaults()
fmt.Printf("NewPersonWithDefaults: %+v\n", p2)
```

**输出：**
```
NewPersonWithDefaults: &{Name:Unknown Age:0}
```

**Functional Options 模式**（可选参数）：

```go
type Server struct {
    host string
    port int
}

type Option func(*Server)

func WithPort(port int) Option {
    return func(s *Server) {
        s.port = port
    }
}

func NewServer(host string, opts ...Option) *Server {
    s := &Server{host: host, port: 8080} // 默认值
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// 使用
s := NewServer("localhost", WithPort(9090))
```

这个模式在标准库（如 `grpc`、`http.Server`）中很常见。

## 9.10 组合优于继承

Go 没有类和继承，而是通过**组合**和**接口**实现代码复用和多态。

**组合示例**：

```go
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

logger := ConsoleLogger{}
service := NewService(logger)
service.DoWork()
```

**输出：**
```
  [Console] Service is working
```

**为什么组合优于继承**：

- **灵活性**：可以在运行时替换依赖（如换成 `FileLogger`）
- **可测试性**：可以注入 mock 对象
- **避免继承链**：继承容易导致深层次的类层次结构，难以理解和修改
- **接口隔离**：只依赖需要的行为（接口），而不是整个类

Go 的设计哲学是**小接口 + 组合**，而不是大类层次结构。

## 9.11 内存对齐

Go 会对结构体字段进行**内存对齐**，以提高访问效率。字段顺序影响结构体大小：

```go
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
```

**输出：**
```
BadOrder 大小: 24 字节
GoodOrder 大小: 16 字节
```

**为什么 `BadOrder` 更大**：

```
BadOrder 内存布局（64 位系统）：
[a:1字节][填充:7字节][b:8字节][c:1字节][填充:7字节]
总共 24 字节

GoodOrder 内存布局：
[b:8字节][a:1字节][c:1字节][填充:6字节]
总共 16 字节
```

`int64` 需要 8 字节对齐，编译器会在 `a` 后面填充 7 字节，使 `b` 对齐到 8 字节边界。

**查看字段偏移**：

```go
fmt.Println("\nBadOrder 字段偏移:")
fmt.Printf("  a 偏移: %d\n", unsafe.Offsetof(BadOrder{}.a))
fmt.Printf("  b 偏移: %d\n", unsafe.Offsetof(BadOrder{}.b))
fmt.Printf("  c 偏移: %d\n", unsafe.Offsetof(BadOrder{}.c))

fmt.Println("\nGoodOrder 字段偏移:")
fmt.Printf("  b 偏移: %d\n", unsafe.Offsetof(GoodOrder{}.b))
fmt.Printf("  a 偏移: %d\n", unsafe.Offsetof(GoodOrder{}.a))
fmt.Printf("  c 偏移: %d\n", unsafe.Offsetof(GoodOrder{}.c))
```

**输出：**
```
BadOrder 字段偏移:
  a 偏移: 0
  b 偏移: 8
  c 偏移: 16

GoodOrder 字段偏移:
  b 偏移: 0
  a 偏移: 8
  c 偏移: 9
```

**优化建议**：

- 把大字段（`int64`、`float64`、指针）放在前面
- 把小字段（`bool`、`int8`）放在后面
- 相同大小的字段放在一起

对于热点结构体（如频繁分配的对象），优化内存对齐可以减少内存占用，提升缓存命中率。

---

## 5 个真实报错怎么读

### 报错 1：结构体字面量缺少字段名

```go
type Person struct {
    Name string
    Age  int
    City string
}

p := Person{"Alice", 25} // 缺少 City
```

**编译器输出：**
```
too few values in Person{...}
```

**原因：** 按顺序初始化时，必须提供所有字段。

**修复：** 使用字段名初始化（推荐）：
```go
p := Person{Name: "Alice", Age: 25}
```

或提供所有字段：
```go
p := Person{"Alice", 25, "Beijing"}
```

### 报错 2：不可比较的结构体作为 map key

```go
type Container struct {
    Data []int
}

m := make(map[Container]string)
```

**编译器输出：**
```
invalid map key type Container
```

**原因：** 切片不可比较，包含切片的结构体也不可比较。

**修复：** 使用数组或字符串作为 key：
```go
type Container struct {
    Data [10]int // 数组可比较
}
```

或用字符串表示：
```go
m := make(map[string]string)
key := fmt.Sprintf("%v", container.Data)
m[key] = "value"
```

### 报错 3：方法接收者类型不匹配

```go
type Counter int

func (c *Counter) Increment() {
    *c++
}

func main() {
    Counter(10).Increment()
}
```

**编译器输出：**
```
cannot call pointer method Increment on Counter
```

**原因：** `Counter(10)` 是一个临时值，不能取地址。

**修复：** 先赋值给变量：
```go
c := Counter(10)
c.Increment()
```

或改用值接收者（如果不需要修改）：
```go
func (c Counter) Value() int {
    return int(c)
}

fmt.Println(Counter(10).Value()) // OK
```

### 报错 4：嵌入字段冲突

```go
type A struct {
    Name string
}

func (a A) Greet() {
    fmt.Println("A:", a.Name)
}

type B struct {
    Name string
}

func (b B) Greet() {
    fmt.Println("B:", b.Name)
}

type C struct {
    A
    B
}

func main() {
    c := C{}
    c.Greet() // 歧义
}
```

**编译器输出：**
```
ambiguous selector c.Greet
```

**原因：** `A` 和 `B` 都有 `Greet` 方法，编译器不知道调用哪个。

**修复：** 显式指定类型：
```go
c.A.Greet()
c.B.Greet()
```

或在 `C` 上定义自己的 `Greet` 方法（覆盖）：
```go
func (c C) Greet() {
    c.A.Greet()
    c.B.Greet()
}
```

### 报错 5：结构体赋值时忘记取地址

```go
type Person struct {
    Name string
}

func NewPerson(name string) *Person {
    return Person{Name: name} // 类型不匹配
}
```

**编译器输出：**
```
cannot use Person{...} (value of type Person) as type *Person in return statement
```

**原因：** 函数返回类型是 `*Person`，但返回的是 `Person`。

**修复：** 返回指针：
```go
func NewPerson(name string) *Person {
    return &Person{Name: name}
}
```

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 按顺序初始化时字段数量不匹配 | 必须提供所有字段 | 用字段名初始化 `Person{Name: "Alice"}` |
| 包含切片的结构体不能作为 map key | 切片不可比较 | 改用数组或字符串 key |
| 临时值不能调用指针接收者方法 | 临时值不能取地址 | 先赋值给变量 |
| 嵌入多个类型有同名方法，调用歧义 | 编译器不知道调用哪个 | 显式指定 `c.A.Method()` 或覆盖 |
| 结构体很大，复制代价高 | 值传递复制整个结构体 | 传指针 `func(p *Person)` |
| 字段顺序不同导致内存浪费 | 内存对齐填充 | 把大字段放前面 |
| 方法接收者是值，修改不生效 | 值接收者操作副本 | 改用指针接收者 `func (p *Person)` |
| 构造函数返回值和实际不匹配 | 忘记取地址 | `return &Person{}` |
| 嵌入类型的私有字段无法访问 | 跨包私有字段不导出 | 通过嵌入类型的公开方法访问 |

---

## 练习

### 第 1 题

定义一个 `Rectangle` 结构体，包含 `Width` 和 `Height` 字段。为它定义两个方法：`Area()` 计算面积，`Perimeter()` 计算周长。

::: details 第 1 题参考答案
```go
type Rectangle struct {
    Width  float64
    Height float64
}

// Area 计算面积
func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

// Perimeter 计算周长
func (r Rectangle) Perimeter() float64 {
    return 2 * (r.Width + r.Height)
}

// 使用
r := Rectangle{Width: 10, Height: 5}
fmt.Printf("面积: %.2f, 周长: %.2f\n", r.Area(), r.Perimeter())
// 输出：面积: 50.00, 周长: 30.00
```

**为什么用值接收者：** 不需要修改接收者，且结构体很小（两个字段），复制代价低。
:::

### 第 2 题

定义一个 `Stack` 结构体（栈），支持 `Push`、`Pop` 和 `IsEmpty` 方法。用切片实现。

::: details 第 2 题参考答案
```go
type Stack struct {
    items []int
}

// Push 入栈
func (s *Stack) Push(item int) {
    s.items = append(s.items, item)
}

// Pop 出栈，返回元素和是否成功
func (s *Stack) Pop() (int, bool) {
    if len(s.items) == 0 {
        return 0, false
    }
    index := len(s.items) - 1
    item := s.items[index]
    s.items = s.items[:index]
    return item, true
}

// IsEmpty 判断是否为空
func (s *Stack) IsEmpty() bool {
    return len(s.items) == 0
}

// 使用
s := Stack{}
s.Push(1)
s.Push(2)
s.Push(3)
fmt.Println(s.Pop()) // 3 true
fmt.Println(s.Pop()) // 2 true
fmt.Println(s.IsEmpty()) // false
```

**为什么用指针接收者：** `Push` 和 `Pop` 需要修改切片，必须用指针接收者。
:::

### 第 3 题

定义一个 `Employee` 结构体，嵌入 `Person` 结构体，并添加 `Salary` 字段。为 `Employee` 定义一个 `String()` 方法，返回格式化的字符串。

::: details 第 3 题参考答案
```go
type Person struct {
    Name string
    Age  int
}

type Employee struct {
    Person
    Salary int
}

// String 实现 fmt.Stringer 接口
func (e Employee) String() string {
    return fmt.Sprintf("%s (age %d), salary: $%d", e.Name, e.Age, e.Salary)
}

// 使用
e := Employee{
    Person: Person{Name: "Alice", Age: 30},
    Salary: 50000,
}
fmt.Println(e)
// 输出：Alice (age 30), salary: $50000
```

**为什么实现 `String()`：** `fmt.Stringer` 接口让 `fmt.Print` 系列函数自动调用 `String()` 方法，提供友好的输出。
:::

### 第 4 题

定义一个 `Counter` 结构体，包含 `value` 字段（私有）。提供 `Increment`、`Decrement` 和 `Value` 方法，保证 `value` 不能为负数。

::: details 第 4 题参考答案
```go
type Counter struct {
    value int
}

// NewCounter 构造函数
func NewCounter(initial int) *Counter {
    if initial < 0 {
        initial = 0
    }
    return &Counter{value: initial}
}

// Increment 增加
func (c *Counter) Increment() {
    c.value++
}

// Decrement 减少，但不能为负
func (c *Counter) Decrement() {
    if c.value > 0 {
        c.value--
    }
}

// Value 返回当前值
func (c *Counter) Value() int {
    return c.value
}

// 使用
c := NewCounter(2)
c.Increment()
fmt.Println(c.Value()) // 3
c.Decrement()
c.Decrement()
c.Decrement() // 不会变成负数
fmt.Println(c.Value()) // 0
```

**为什么字段私有：** 封装保证不变量（`value >= 0`），外部无法直接修改字段，只能通过方法访问。
:::

### 第 5 题

优化下面结构体的内存布局，减少内存占用（64 位系统）：

```go
type Data struct {
    a bool
    b int64
    c bool
    d int32
    e bool
}
```

::: details 第 5 题参考答案
**原结构体大小：** 40 字节

**内存布局：**
```
[a:1][填充:7][b:8][c:1][填充:3][d:4][e:1][填充:7]
= 40 字节
```

**优化后：**
```go
type Data struct {
    b int64  // 8 字节，偏移 0
    d int32  // 4 字节，偏移 8
    a bool   // 1 字节，偏移 12
    c bool   // 1 字节，偏移 13
    e bool   // 1 字节，偏移 14
    // 填充 1 字节，偏移 15
}
```

**优化后大小：** 16 字节

**原则：**
1. 按字段大小降序排列：`int64` → `int32` → `bool`
2. 相同大小的字段放在一起
3. 减少填充字节

**验证：**
```go
fmt.Println(unsafe.Sizeof(Data{})) // 16
```
:::

### 第 6 题

解释下面代码的输出，并说明为什么：

```go
type Person struct {
    Name string
}

func (p Person) SetName(name string) {
    p.Name = name
}

func main() {
    p := Person{Name: "Alice"}
    p.SetName("Bob")
    fmt.Println(p.Name) // 输出什么？
}
```

::: details 第 6 题参考答案
**输出：** `Alice`

**原因：** `SetName` 是值接收者，接收的是 `p` 的副本。修改副本的 `Name` 字段不会影响原 `p`。

**修复：** 改用指针接收者：
```go
func (p *Person) SetName(name string) {
    p.Name = name
}

p := Person{Name: "Alice"}
p.SetName("Bob")
fmt.Println(p.Name) // Bob
```

**规则：** 如果方法需要修改接收者，必须用指针接收者。
:::

---

## 小结

- 结构体用 `type` 和 `struct` 定义，字面量初始化推荐用字段名而不是顺序。
- 匿名字段（嵌入）会提升字段和方法到外层类型，实现类似继承的效果。
- 字段标签用于元数据，常用于 JSON 序列化、数据库映射等，通过反射读取。
- 结构体可比较性取决于所有字段是否可比较，可比较的结构体可以作为 map key。
- 方法是绑定到类型上的函数，接收者可以是值或指针；指针接收者可以修改原值。
- 值接收者复制接收者，指针接收者操作原值；如果不确定，用指针接收者。
- Go 会自动转换：值可以调用指针接收者方法（自动取地址），指针可以调用值接收者方法（自动解引用）。
- 方法值绑定了接收者，方法表达式未绑定接收者，需要显式传递。
- 构造函数惯例是 `NewXxx`，返回指针；可以用 Functional Options 模式实现可选参数。
- Go 推崇组合优于继承，通过嵌入和接口实现代码复用和多态。
- 内存对齐影响结构体大小，把大字段放前面可以减少内存浪费。

下一章我们将学习**接口与类型系统**：隐式实现、接口嵌入、类型断言、type switch、typed nil 陷阱、`io.Reader`/`io.Writer` 实战，以及接口的设计原则。
