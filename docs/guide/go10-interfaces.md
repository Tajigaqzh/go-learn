# 第 10 章 · 接口与类型系统

在掌握了结构体和方法之后，你已经能定义自己的数据类型并为其附加行为了。但当你需要编写"不关心具体类型，只关心能做什么"的代码时，就需要接口了。Go 的接口系统与 Java/C++ 截然不同：没有显式的 `implements` 关键字，实现关系由编译器自动推导。这种"鸭子类型"的设计让代码更松耦合、更易测试。

本章将系统讲解隐式实现、接口嵌入、`any` 与空接口、类型断言与 type switch、typed nil 陷阱、`io.Reader`/`io.Writer` 实战、小接口设计原则、接口值的内存表示，以及依赖倒置的实践。掌握这些内容后，你将能写出高度可复用、易于替换实现的 Go 代码。

本章配套代码在 `internal/chapter/go10_interfaces/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 10.1 隐式实现

Go 的接口实现是**隐式**的：只要类型实现了接口的所有方法，就自动满足该接口，不需要显式声明。

```go
type Shape interface {
    Area() float64
    Perimeter() float64
}

type Circle struct {
    Radius float64
}

func (c Circle) Area() float64 {
    return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
    return 2 * math.Pi * c.Radius
}
```

`Circle` 没有说 `implements Shape`，但它有 `Area()` 和 `Perimeter()` 方法，所以它可以赋值给 `Shape` 类型的变量：

```go
var s Shape = Circle{Radius: 5}
fmt.Println(s.Area()) // 78.54
```

### 接口作为函数参数

```go
func PrintShapeInfo(s Shape) {
    fmt.Printf("面积=%.2f, 周长=%.2f\n", s.Area(), s.Perimeter())
}

PrintShapeInfo(Circle{Radius: 5})       // OK
PrintShapeInfo(Rectangle{Width: 3, Height: 4}) // OK
```

### 实测输出

```text
Circle:
  面积=78.54, 周长=31.42
Rectangle:
  面积=12.00, 周长=14.00
Go 的隐式实现：不需要声明 implements，编译器自动检查
```

**优势**：
- 接口定义和实现可以位于不同包，互不知道对方存在
- 已有类型可以" retroactive "地满足新接口，无需修改原代码
- 鼓励面向接口编程，降低耦合

---

## 10.2 接口嵌入

接口可以嵌入其他接口，类似于结构体嵌入字段，组合成新接口：

```go
type ReadWriter interface {
    io.Reader
    io.Writer
}
```

`ReadWriter` 自动拥有 `io.Reader` 和 `io.Writer` 的所有方法。任何同时实现了 `Read` 和 `Write` 的类型都满足 `ReadWriter`。

### 实测输出

```text
ReadWriter 接口嵌入演示：Hello
接口嵌入类似于结构体嵌入，可以组合多个接口形成新接口
io.ReadWriter 就是 io.Reader + io.Writer 的组合
```

**标准库中的例子**：
- `io.ReadWriter` = `io.Reader` + `io.Writer`
- `io.ReadWriteCloser` = `io.Reader` + `io.Writer` + `io.Closer`

---

## 10.3 any（空接口）

`any` 是 Go 1.18 引入的 `interface{}` 的别名，表示"任意类型"。

```go
var x any = 42
x = "Hello"
x = []int{1, 2, 3}
```

### 空接口的局限

`any` 可以存储任意值，但丢失了类型信息，使用时必须通过类型断言恢复：

```go
var x any = 42
// x + 1 // 编译错误：invalid operation: x + 1 (mismatched types interface {} and int)
n := x.(int) // 类型断言
fmt.Println(n + 1) // 43
```

### 实测输出

```text
x = 42, 类型=int
x = "Hello", 类型=string
x = [1 2 3], 类型=[]int
any 可以存储任意类型，但使用时需要类型断言
```

**最佳实践**：
- 优先使用具体类型或明确的接口
- `any` 只在真正需要存储任意类型的场景使用（如 JSON 反序列化、通用容器）
- 不要滥用 `any` 来逃避类型系统

---

## 10.4 类型断言

类型断言用于从接口值中提取具体类型：

```go
var s Shape = Circle{Radius: 3}

// 带 ok 的形式：安全
c, ok := s.(Circle)
fmt.Println(ok, c.Radius) // true, 3

// 不带 ok：如果类型不匹配会 panic
// r := s.(Rectangle) // panic
```

### 先判断再断言

```go
if r, ok := s.(Rectangle); ok {
    fmt.Println("是 Rectangle")
} else {
    fmt.Println("s 不是 Rectangle")
}
```

### 实测输出

```text
s.(Circle): ok=true, radius=3.00
s 不是 Rectangle
```

---

## 10.5 type switch

当需要对多种类型做不同处理时，使用 `type switch`：

```go
switch v := s.(type) {
case Circle:
    fmt.Printf("Circle, radius=%.2f\n", v.Radius)
case Rectangle:
    fmt.Printf("Rectangle, width=%.2f, height=%.2f\n", v.Width, v.Height)
default:
    fmt.Printf("未知类型: %T\n", v)
}
```

**注意**：`type switch` 中的 `v` 在每个 `case` 中的类型是具体类型，不是接口类型。

### 实测输出

```text
Circle, radius=2.00
Rectangle, width=3.00, height=4.00
Circle, radius=5.00
```

---

## 10.6 typed nil 陷阱

这是 Go 接口系统中最经典的陷阱之一：

```go
var d *Document = nil
var p Printer = d // p 不是 nil！

fmt.Println(d == nil) // true
fmt.Println(p == nil) // false！
```

### 原因

接口值内部由两部分组成：**类型描述符** + **动态值**。当把 `nil` 指针赋给接口时，接口值的类型描述符是 `*Document`（非 nil），只有动态值是 nil。因此接口值本身不是 nil。

### 安全处理

```go
func (d *Document) Print() {
    if d == nil {
        fmt.Println("(nil Document)")
        return
    }
    // ...
}
```

### 实测输出

```text
d == nil: true
p == nil: false
typed nil：接口值本身不是 nil，只是它持有的具体值为 nil
  (nil Document)
判断接口是否为 nil：只能用 reflect.ValueOf(v).IsNil() 或避免存储 nil 指针到接口
```

**最佳实践**：
- 函数返回接口时，确保返回真正的 `nil` 而不是 typed nil
- 方法接收者要处理 nil 情况
- 避免将可能为 nil 的指针赋值给接口变量

---

## 10.7 io.Reader / io.Writer 实战

`io.Reader` 和 `io.Writer` 是 Go 标准库中最重要的两个接口：

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

### 为什么重要

几乎所有 IO 操作都基于这两个接口：文件、网络连接、缓冲区、压缩流、加密流……它们实现了统一的抽象。

```go
var buf bytes.Buffer
buf.Write([]byte("Hello, Go!"))

data := make([]byte, 5)
buf.Read(data)

// io.Copy 连接 Reader 和 Writer
var dest bytes.Buffer
io.Copy(&dest, strings.NewReader("Copy me"))
```

### 实测输出

```text
写入后 buf: "Hello, Go!"
读取 5 字节: "Hello", 剩余: ", Go!"
io.Copy 复制了 7 字节: "Copy me"
io.Reader/io.Writer 是 Go 最重要的接口，几乎所有 IO 操作都基于它们
```

**设计思想**：小接口 + 组合 = 强大的流处理管道。

---

## 10.8 小接口设计

Go 的设计理念是**接口越小越好**。小接口更容易实现，组合更灵活。

### 标准库中的小接口

| 接口 | 方法 | 使用者 |
| --- | --- | --- |
| `fmt.Stringer` | `String() string` | `fmt.Printf` |
| `io.Reader` | `Read([]byte) (int, error)` | `io.Copy`, `json.Decoder` |
| `io.Writer` | `Write([]byte) (int, error)` | `fmt.Fprintf`, `log.Logger` |
| `error` | `Error() string` | 错误处理 |

### 接口组合

```go
type Stringer interface {
    String() string
}

type Formatter interface {
    Format() string
}

type Logger interface {
    Stringer
    Formatter
}
```

### 实测输出

```text
Stringer: [INFO] 系统启动
Formatter: INFO: 系统启动
Go 推荐小接口设计：接口越小，实现越容易，组合越灵活
fmt.Stringer 只有一个 String() 方法，是最成功的小接口之一
```

**Rob Pike 的接口设计建议**：
> "The bigger the interface, the weaker the abstraction."（接口越大，抽象越弱。）

---

## 10.9 接口值的内存表示

接口值在内存中由两部分组成：

1. **类型描述符**（itab/type）：指向类型信息的指针
2. **动态值**（data）：指向实际数据的指针

```go
var s Shape = Circle{Radius: 3}
// 接口值：类型=*Circle 的 itab，数据=Circle{3} 的副本

var p Shape = &Circle{Radius: 5}
// 接口值：类型=*Circle 的 itab，数据=&Circle{5} 的指针
```

### 值 vs 指针存储的区别

- 存储值类型时：数据会被**复制**到接口值中（或堆上）
- 存储指针类型时：接口值保存的是**指针**

### 实测输出

```text
接口值类型: go10_interfaces.Circle, 值: {3}
接口值（存指针）类型: *go10_interfaces.Circle, 值: &{5}
接口值 = 类型描述符 + 动态值
typed nil 陷阱的原因：接口值的类型描述符不为 nil，只是动态值为 nil
```

---

## 10.10 依赖倒置

依赖倒置原则（Dependency Inversion Principle）：高层模块不应依赖低层模块，两者都应依赖抽象。

### 例子：用户服务与数据存储

```go
// 抽象：DataStore 接口
type DataStore interface {
    Get(key string) (string, bool)
    Set(key, value string)
}

// 低层实现：内存存储
type MemoryStore struct{ ... }
func (m *MemoryStore) Get(key string) (string, bool) { ... }
func (m *MemoryStore) Set(key, value string) { ... }

// 高层模块：UserService
type UserService struct {
    store DataStore // 依赖接口，不是具体实现
}

func NewUserService(store DataStore) *UserService {
    return &UserService{store: store}
}
```

### 好处

- **可替换性**：可以轻松换成 `RedisStore`、`DBStore` 等
- **可测试性**：测试时用 `MockStore` 代替真实存储
- **松耦合**：高层和低层通过接口契约交互

### 实测输出

```text
GetUserName("1"): name=Alice, ok=true
依赖倒置：高层模块(UserService)依赖抽象(DataStore)，不依赖具体实现
好处：可以方便地替换存储实现（如换成 RedisStore、DBStore），便于测试
```

---

## 3 个真实报错怎么读

### 报错 1：类型断言失败 panic

```text
panic: interface conversion: go10_interfaces.Shape is go10_interfaces.Circle, not go10_interfaces.Rectangle

goroutine 1 [running]:
main.section4_TypeAssertion(...)
    .../demo.go:xxx +0x...
```

**原因**：对接口值执行了不匹配的类型断言 `s.(Rectangle)`，而 `s` 实际存储的是 `Circle`。

**解法**：使用带 `ok` 的形式：`v, ok := s.(Rectangle)`，或在断言前用 `type switch` 判断。

### 报错 2：类型不匹配赋值

```text
./demo.go:xx:yy: cannot use d (type *Document) as type Printer in assignment:
    *Document does not implement Printer (missing Print method)
```

**原因**：`*Document` 没有实现 `Printer` 接口的所有方法（可能是方法接收者是值类型而非指针类型）。

**解法**：检查方法接收者类型。值接收者的方法对值和指针都可用，指针接收者的方法只对指针可用。

### 报错 3：typed nil 判断错误

```go
var d *Document = nil
var p Printer = d

if p == nil {
    // 不会执行！p 不是 nil
}
```

**原因**：接口值包含类型信息，即使动态值为 nil，接口值本身也不是 nil。

**解法**：
- 函数返回接口时，直接返回 `nil` 而不是 typed nil
- 或者使用 `reflect.ValueOf(v).IsNil()` 判断（有性能开销）
- 最佳做法：避免将可能为 nil 的指针赋值给接口变量

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 类型断言 panic | 接口存储的类型与断言类型不匹配 | 使用 `v, ok := x.(T)` 安全形式 |
| `p == nil` 为 false | typed nil：接口值含类型信息 | 返回接口时直接返回 `nil`，不通过变量 |
| 值类型无法调用指针方法 | 方法集不包含指针接收者方法 | 取地址：`(&v).Method()` |
| 实现了方法但接口不满足 | 方法签名不匹配（参数/返回值不同） | 检查方法签名是否与接口完全一致 |
| 接口赋值后修改原值无效 | 接口值存储的是值副本 | 接口中存储指针而非值 |
| `any` 不能做运算 | 丢失了类型信息 | 先做类型断言再运算 |
| type switch 中修改无效 | case 中变量是副本 | 指针类型用 `*T` case，或在 case 中取地址 |

---

## 练习

### 第 1 题

定义一个 `Speaker` 接口（`Speak() string`），让 `Dog` 和 `Cat` 结构体分别实现它，然后写一个函数 `MakeSound(s Speaker)` 调用并打印返回值。

::: details 第 1 题参考答案

```go
type Speaker interface {
    Speak() string
}

type Dog struct{}

func (d Dog) Speak() string {
    return "Woof!"
}

type Cat struct{}

func (c Cat) Speak() string {
    return "Meow!"
}

func MakeSound(s Speaker) {
    fmt.Println(s.Speak())
}

MakeSound(Dog{}) // Woof!
MakeSound(Cat{}) // Meow!
```

**为什么这样写更好**：通过接口 `Speaker` 解耦，以后添加 `Bird`、`Cow` 等新类型时，不需要修改 `MakeSound` 函数。

:::

### 第 2 题

观察以下代码，解释为什么 `p == nil` 输出 `false`：

```go
type Printer interface{ Print() }
type Doc struct{}
func (d *Doc) Print() {}

var d *Doc = nil
var p Printer = d
fmt.Println(p == nil)
```

::: details 第 2 题参考答案

输出是 `false`。

**原因**：接口值 `p` 由两部分组成：类型描述符（`*Doc`）和动态值（`nil`）。虽然动态值是 nil，但类型描述符不是 nil，所以 `p` 本身不是 nil。

**正确做法**：函数返回接口时，如果底层值为 nil，应该直接返回 `nil`：

```go
func NewDoc() Printer {
    // return &Doc{} // OK
    return nil // 也 OK
    // var d *Doc = nil; return d // 错误！返回 typed nil
}
```

:::

### 第 3 题

写一个 `type switch`，处理 `[]Shape` 中的 `Circle` 和 `Rectangle`，分别打印不同的描述信息。

::: details 第 3 题参考答案

```go
shapes := []Shape{
    Circle{Radius: 2},
    Rectangle{Width: 3, Height: 4},
}

for _, s := range shapes {
    switch v := s.(type) {
    case Circle:
        fmt.Printf("圆，半径=%.2f\n", v.Radius)
    case Rectangle:
        fmt.Printf("矩形，宽=%.2f，高=%.2f\n", v.Width, v.Height)
    default:
        fmt.Printf("未知形状: %T\n", v)
    }
}
```

**注意**：`type switch` 中 `case Circle:` 匹配的是值类型，如果接口中存储的是 `*Circle`，应该用 `case *Circle:`。

:::

### 第 4 题

`io.Reader` 和 `io.Writer` 为什么设计成只有一个方法的小接口？结合标准库中的 `io.Copy` 说明小接口的优势。

::: details 第 4 题参考答案

```go
// io.Copy 的实现
copy(dst Writer, src Reader) (written int64, err error)
```

**原因分析**：
- `io.Reader` 只有一个 `Read` 方法，`io.Writer` 只有一个 `Write` 方法
- 几乎所有数据源（文件、网络、内存、压缩流）都实现了 `io.Reader`
- 几乎所有数据目标（文件、网络、内存、日志）都实现了 `io.Writer`
- `io.Copy` 只需要这两个接口，就能连接任意数据源和数据目标

**优势**：小接口更容易实现，组合更灵活。如果 `io.Reader` 要求 10 个方法，很多类型将无法直接使用 `io.Copy`。

:::

### 第 5 题

实现一个 `MockStore`（内存中的 `DataStore` 接口实现），用于测试 `UserService`。验证 `GetUserName` 的行为。

::: details 第 5 题参考答案

```go
type MockStore struct {
    data map[string]string
}

func NewMockStore() *MockStore {
    return &MockStore{data: make(map[string]string)}
}

func (m *MockStore) Get(key string) (string, bool) {
    v, ok := m.data[key]
    return v, ok
}

func (m *MockStore) Set(key, value string) {
    m.data[key] = value
}

// 测试
mock := NewMockStore()
mock.Set("user:1", "TestUser")
service := NewUserService(mock)
name, ok := service.GetUserName("1")
// name = "TestUser", ok = true
```

**为什么这样写更好**：`UserService` 依赖 `DataStore` 接口而不是具体实现，测试时可以用 `MockStore` 替代真实数据库，避免测试依赖外部服务。

:::

### 第 6 题

以下代码有什么问题？如何修复？

```go
func getPrinter() Printer {
    var d *Document = nil
    if d == nil {
        return d // 想返回 nil
    }
    return d
}

func main() {
    p := getPrinter()
    if p == nil {
        fmt.Println("p is nil")
    } else {
        fmt.Println("p is not nil")
    }
}
```

::: details 第 6 题参考答案

输出是 `p is not nil`，与预期相反。

**问题**：`getPrinter()` 返回了一个 typed nil。`d` 是 `*Document` 类型的 nil，赋值给 `Printer` 接口后，接口值的类型描述符是 `*Document`，所以接口值不是 nil。

**修复**：

```go
func getPrinter() Printer {
    var d *Document = nil
    if d == nil {
        return nil // 直接返回 nil
    }
    return d
}
```

**关键区别**：
- `return nil`：接口值的类型和值都是 nil，是真正的 nil
- `return d`（d 是 nil 指针）：接口值类型为 `*Document`，值是 nil，是 typed nil

:::

---

## 小结

本章覆盖了 Go 接口与类型系统的核心概念，关键要点：

- **隐式实现**：不需要 `implements`，编译器自动检查方法集，实现关系可跨包建立。
- **接口嵌入**：可以组合多个接口形成新接口，如 `io.ReadWriter`。
- **`any` 与空接口**：可存储任意类型，但丢失类型信息，使用时需类型断言。
- **类型断言**：`v, ok := x.(T)` 是安全形式，直接断言不匹配会 panic。
- **type switch**：处理接口值的多种具体类型，比多个 `if` 更简洁。
- **typed nil 陷阱**：接口值含类型信息，nil 指针赋给接口后接口值不是 nil。
- **`io.Reader`/`io.Writer`**：Go 最重要的小接口，统一了所有 IO 抽象。
- **小接口设计**：接口越小，实现越容易，组合越灵活；避免大而全的接口。
- **依赖倒置**：高层依赖抽象（接口），不依赖具体实现；便于替换和测试。

掌握这些概念后，你将能写出高度解耦、易于测试和扩展的 Go 代码。下一章将学习错误处理，深入理解 Go 中 `error` 接口的设计哲学和最佳实践。
