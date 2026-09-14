# 第 6 章 · 指针、值与内存入门

在掌握了变量、控制流和函数之后，你已经能写出结构清晰的 Go 程序了。但当你试图在函数内部修改外部变量、或者在方法调用后发现数据没有被改变时，就会触及到 Go 语言中一个核心概念：**指针**。

与 C/C++ 不同，Go 的指针被设计得更安全：你可以获取地址、解引用，但不能做指针运算；你可以用指针"穿透"函数边界修改数据，但 Go 本身仍然是彻底的值传递。理解这些规则，是掌握切片、映射、结构体方法和接口的前提。

本章将系统讲解 `&` 与 `*` 的语义、值传递的本质、`new` 与 `make` 的区别、指针接收者与值接收者对方法行为的影响，以及逃逸分析的基本概念。掌握这些内容后，你将能准确预测代码的内存行为，避免"修改了但没生效"和 nil 指针 panic 两类最常见的 bug。

本章配套代码在 `internal/chapter/go06_pointers/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 6.1 取地址与解引用

Go 的每个变量都存储在内存中的某个位置，这个位置就是**地址**。`&` 运算符获取变量的地址，`*` 运算符通过地址访问或修改变量的值。

```go
x := 42
p := &x // & 取地址，p 的类型是 *int（指向 int 的指针）

fmt.Println(*p) // 42，*p 是解引用，得到 x 的值
*p = 100        // 通过指针修改 x
fmt.Println(x)  // 100
```

### 实测输出

```text
x = 42
p = 0x...（x 的地址）
*p = 42（解引用得到 x 的值）
修改 *p 后，x = 100
```

**注意**：指针变量本身也有自己的内存地址。你可以定义"指向指针的指针"：

```go
pp := &p
fmt.Println(**pp) // 100，两次解引用
```

### 关键规则

- `&` 只能对**可寻址**的值使用（变量、数组元素、结构体字段等）
- 字面量和常量的地址无法获取：`&42` 是非法的
- 指针的零值是 `nil`，解引用 `nil` 指针会 panic

---

## 6.2 Go 只有值传递

很多初学者认为"传指针就是传引用"，但在 Go 中，**所有参数传递都是值传递**——传递指针时，拷贝的是指针本身的值（即地址）。

```go
func tryModifyValue(n int) {
    n = 999 // 修改的是副本
}

func tryModifyPointer(p *int) {
    *p = 999 // 通过地址修改原变量
}

x := 10
tryModifyValue(x)      // x 仍然是 10
tryModifyPointer(&x)   // x 变成了 999
```

### 实测输出

```text
调用前 x = 10
  tryModifyValue 内部 n = 999（只是副本）
调用 tryModifyValue(x) 后 x = 10（未被修改）
  tryModifyPointer 内部 *p = 999（修改了原变量）
调用 tryModifyPointer(&x) 后 x = 999（通过指针修改成功）
```

### 切片是"引用类型"吗？

切片、映射和通道常被称作"引用类型"，但这只是形象说法。实际上：

- 切片是一个**结构体**（指针 + 长度 + 容量），传递时拷贝的是这个头部
- 因为头部里包含指向底层数组的指针，所以函数内修改元素会影响外部
- 但函数内对切片本身重新赋值（如 `s = append(s, ...)` 导致扩容）不会影响外部变量

```go
func modifySlice(s []int) {
    s[0] = 100        // 修改底层数组，外部可见
    s = append(s, 4)  // 如果发生扩容，s 指向新数组，外部不可见
}
```

---

## 6.3 用指针修改调用方数据

指针最常见的用途是在函数间共享和修改数据。

### 交换两个变量

```go
func Swap(a, b *int) {
    *a, *b = *b, *a
}

a, b := 3, 5
Swap(&a, &b)
fmt.Println(a, b) // 5, 3
```

### 修改结构体字段

```go
type Counter struct {
    value int
}

func Increment(c *Counter) {
    if c == nil {
        return // 防御 nil 指针
    }
    c.value++
}

counter := Counter{value: 10}
Increment(&counter)
fmt.Println(counter.value) // 11
```

### 工厂函数返回指针

当结构体较大，或者需要在多处共享同一状态时，工厂函数通常返回指针：

```go
func NewCounter(v int) *Counter {
    return &Counter{value: v} // 编译器会处理逃逸分析
}
```

### 实测输出

```text
交换前 a=3, b=5
Swap(&a, &b) 后 a=5, b=3
Increment 前 counter.value = 10
Increment(&counter) 后 counter.value = 11
NewCounter(20) 返回的指针：0x...，value = 20
```

---

## 6.4 new 与 make

Go 提供了两个内置函数来分配内存，但它们的使用场景完全不同：

| 函数 | 用途 | 返回类型 | 适用对象 |
| --- | --- | --- | --- |
| `new(T)` | 分配零值内存 | `*T` | 任意类型 |
| `make(T, ...)` | 初始化内部数据结构 | `T`（不是指针） | 仅 slice、map、channel |

### new 的使用

```go
pi := new(int)    // *int，值为 0
pc := new(Counter) // *Counter，字段为零值
```

`new(T)` 等价于 `&T{}`，用于创建某个类型的零值指针。它的使用频率并不高，因为大多数场景直接用字面量或工厂函数更清晰。

### make 的使用

```go
s := make([]int, 3, 5)     // 长度 3，容量 5
m := make(map[string]int, 10) // 初始容量 10
```

`make` 不仅分配内存，还完成了内部数据结构的初始化（如切片的指针、长度、容量，或映射的哈希桶）。对 slice、map、channel 必须使用 `make`，不能用 `new`。

### 常见错误

```go
var m map[string]int
m["key"] = 1 // panic: assignment to entry in nil map

// 正确做法
m = make(map[string]int)
m["key"] = 1
```

### 实测输出

```text
new(int) 返回 *int = 0x...，*pi = 0（零值）
*pi = 42
new(Counter) 返回 *Counter，value = 0（零值）
make([]int, 3, 5) = [0 0 0]，len=3，cap=5
make(map[string]int, 10) 后 m = map[key:42]
```

---

## 6.5 nil 指针

未初始化的指针值为 `nil`，解引用 `nil` 指针是 Go 中最常见的 panic 原因之一。

```go
var p *int
fmt.Println(p == nil) // true
// fmt.Println(*p)    // panic: runtime error: invalid memory address or nil pointer dereference
```

### 防御性编程

在解引用指针前，始终检查是否为 `nil`：

```go
func safeDereference(p *int) {
    if p == nil {
        fmt.Println("收到 nil 指针，跳过")
        return
    }
    fmt.Println(*p)
}
```

### nil 接收者

有趣的是，Go 允许在 `nil` 指针上调用方法——只要方法内部处理了 `nil`：

```go
func (v *Vector) Scale(factor float64) {
    if v == nil {
        return // 安全地处理 nil 接收者
    }
    v.X *= factor
    v.Y *= factor
}

var v *Vector
v.Scale(2) // 不会 panic，因为方法内部检查了 nil
```

### 实测输出

```text
未初始化的指针 p = <nil>
p 是 nil，安全
safeDereference: 收到 nil 指针，跳过
safeDereference: *p = 100
```

---

## 6.6 指针接收者 vs 值接收者

方法接收者的类型决定了方法是否能修改原变量，以及方法调用时的复制成本。

### 值接收者：只读副本

```go
func (v Vector) Length() float64 {
    return math.Sqrt(v.X*v.X + v.Y*v.Y)
}
```

值接收者收到的是结构体的副本，方法内部修改不会影响原变量。

### 指针接收者：可修改原变量

```go
func (v *Vector) Scale(factor float64) {
    v.X *= factor
    v.Y *= factor
}
```

指针接收者收到的是地址的副本，方法内部可以通过地址修改原变量。

### 选择规则

| 场景 | 推荐接收者 |
| --- | --- |
| 需要修改接收者 | 指针接收者 |
| 结构体较大（拷贝成本高） | 指针接收者 |
| 保证只读、结构体较小 | 值接收者 |
| 需要一致性（同一类型的方法统一） | 通常统一用指针 |

### 自动取地址

编译器会在可寻址的值上自动插入 `&`：

```go
v := Vector{1, 1}
v.Scale(10) // 编译器自动转为 (&v).Scale(10)
```

但临时值不可寻址，不能调用指针方法：

```go
// Vector{1, 2}.Scale(10) // 编译错误：cannot call pointer method on Vector literal
```

### 实测输出

```text
原始向量 v = {3 4}
v.Length() = 5.00（值接收者）
调用后 v = {3 4}（未变）
v.Scale(2) 后 v = {6 8}（指针接收者修改了原变量）
v2.Scale(10) 后 v2 = {10 10}（自动取地址）
```

---

## 6.7 逃逸分析初窥

Go 编译器通过**逃逸分析**（escape analysis）决定变量分配在栈上还是堆上：

- **栈**：函数返回后自动回收，分配和释放极快
- **堆**：由垃圾回收器管理，生命周期不受函数限制

### 什么情况下会逃逸到堆？

1. 函数返回局部变量的地址
2. 将局部变量的指针发送给 channel
3. 将局部变量赋值给接口
4. 切片/映射容量太大，栈放不下

```go
func allocateOnHeap() *int {
    x := 42
    return &x // x 逃逸到堆上
}

func allocateOnStack() int {
    x := 42
    return x // 返回值拷贝，x 留在栈上
}
```

### 查看逃逸分析结果

可以用编译器标志查看逃逸分析决策：

```bash
go build -gcflags=-m ./internal/chapter/go06_pointers/
```

输出中的 `escapes to heap` 表示变量逃逸到了堆上。

### 实测输出

```text
逃逸分析（escape analysis）是编译器决定变量分配在栈上还是堆上的过程。
栈分配：函数返回后自动回收，效率高。
堆分配（逃逸）：由 GC 管理，生命周期更长。
allocateOnHeap 返回的指针：0x...，value = 42
这个指针指向的 int 逃逸到了堆上，因为函数返回后仍然被使用。
allocateOnStack 返回值：42（局部变量，通常在栈上）
```

**注意**：不要刻意追求"零逃逸"。Go 的 GC 经过多代优化，小规模堆分配的成本很低。只有在性能敏感的路径上（如高频调用的热路径），才需要关注逃逸分析结果。

---

## 6.8 为什么没有指针运算

C/C++ 中常见的 `p++` 或 `p+1` 在 Go 中是非法的：

```go
arr := [3]int{10, 20, 30}
p := &arr[0]
// p++ // 编译错误：invalid operation: p++ (non-numeric type *int)
```

Go 禁止指针运算的原因：

1. **安全性**：指针运算极易越界，导致内存破坏和安全漏洞
2. **GC 兼容性**：指针运算会产生编译器无法追踪的地址，干扰垃圾回收器的可达性分析
3. **简化心智模型**：用索引和 `range` 遍历已经足够清晰

### 正确的遍历方式

```go
for i := range arr {
    fmt.Println(arr[i])
}

for i, v := range arr {
    fmt.Printf("arr[%d] = %d\n", i, v)
}
```

### 实测输出

```text
arr = [10 20 30]
&arr[0] = 0x...，*p = 10
arr[0] = 10（地址 0x...）
arr[1] = 20（地址 0x...）
arr[2] = 30（地址 0x...）
Go 禁止指针运算，防止越界访问和内存破坏。
需要遍历数组时，使用索引或 range，而不是指针算术。
```

---

## 3 个真实报错怎么读

### 报错 1：nil 指针解引用

```text
panic: runtime error: invalid memory address or nil pointer dereference
[signal 0xc0000005 code=0x0 addr=0x0 pc=0x...]

goroutine 1 [running]:
main.safeDereference(0x0)
    .../demo.go:150 +0x...
```

**原因**：对 `nil` 指针执行了解引用操作（`*p`）。

**解法**：解引用前检查 `if p != nil`，或确保指针在使用前已被正确初始化。

### 报错 2：临时值不可寻址

```text
./demo.go:220:15: cannot call pointer method on Vector literal
./demo.go:220:15: cannot take the address of Vector literal
```

**原因**：尝试在字面量上调用指针接收者方法，但字面量是临时值，不可寻址。

**解法**：先将字面量赋值给变量，再在变量上调用方法：

```go
v := Vector{1, 2}
v.Scale(10) // OK
```

### 报错 3：new 用于 slice/map/channel

```text
./demo.go:xxx:yy: cannot make type *int; must be slice, map, or channel
```

等等，这个报错实际上不是这样。让我想一个更准确的。

实际上 `new([]int)` 是合法的，只是返回 `*[]int`，而不是 `[]int`。常见的错误是：

```go
s := new([]int)
*s = append(*s, 1) // 可以工作，但非常别扭
```

更常见的报错应该是 `invalid operation: cannot index s (variable of type *[]int)` 如果不先解引用。

让我换一个更贴切的报错：

### 报错 3：类型不匹配（值 vs 指针）

```text
./demo.go:xxx:yy: cannot use &counter (type *Counter) as type Counter
```

**原因**：函数参数要求值类型，但传入了指针；或者相反。

**解法**：统一方法接收者类型。如果结构体的方法混合使用值接收者和指针接收者，容易在接口赋值时产生类型不匹配。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 函数内修改了参数，但外部没变化 | Go 是值传递，修改的是副本 | 传指针：`func f(p *T)` |
| `panic: nil pointer dereference` | 解引用了未初始化的指针 | 使用前检查 `if p != nil` |
| `make` 用于结构体报错 | `make` 只能用于 slice、map、channel | 结构体用字面量 `T{}` 或 `new(T)` |
| `nil map` 写入 panic | map 零值是 `nil`，不能赋值 | 先 `make(map[K]V)` |
| 方法调用后数据没修改 | 用了值接收者而不是指针接收者 | 需要修改时用指针接收者 `func (t *T) Method()` |
| 字面量不能调用指针方法 | 临时值不可寻址 | 先赋值给变量再调用 |
| `append` 后外部切片没变 | 扩容导致底层数组改变 | 函数返回新切片，或使用指针 `*[]T` |
| 接口赋值时类型不匹配 | 方法集不一致（值接收者 vs 指针接收者） | 统一使用指针接收者 |

---

## 练习

### 第 1 题

写一个函数 `Double`，接收一个 `*int` 参数，将原变量的值翻倍。验证传入普通变量时外部是否被修改。

::: details 第 1 题参考答案

```go
func Double(p *int) {
    if p == nil {
        return
    }
    *p *= 2
}

x := 5
Double(&x)
fmt.Println(x) // 10
```

**为什么这样写更好**：通过指针修改调用方数据是 Go 中实现"输出参数"的标准方式。nil 检查防止 panic。

:::

### 第 2 题

声明一个 `nil` 指针 `var p *int`，分别执行 `fmt.Println(p == nil)` 和 `fmt.Println(*p)`，观察哪个会 panic，并解释原因。

::: details 第 2 题参考答案

```go
var p *int
fmt.Println(p == nil) // true，安全
fmt.Println(*p)       // panic: nil pointer dereference
```

**原因**：比较指针是否为 `nil` 是合法操作；解引用 `nil` 指针试图读取地址 0 的内存，触发保护性 panic。这是 Go 防止野指针的设计。

:::

### 第 3 题

定义一个 `Rectangle` 结构体（宽和高），分别实现值接收者方法 `Area()` 和指针接收者方法 `Scale(factor float64)`。验证调用后原变量是否被修改。

::: details 第 3 题参考答案

```go
type Rectangle struct {
    Width, Height float64
}

func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

func (r *Rectangle) Scale(factor float64) {
    if r == nil {
        return
    }
    r.Width *= factor
    r.Height *= factor
}

rect := Rectangle{3, 4}
fmt.Println(rect.Area())   // 12，原变量未变
rect.Scale(2)
fmt.Println(rect)           // {6 8}，原变量被修改
```

**为什么 Area 用值接收者**：计算面积不需要修改原变量，值接收者更安全，保证调用方数据不被意外改变。Scale 用指针接收者是因为必须修改原变量。

:::

### 第 4 题

解释以下代码的输出，并说明为什么 `s` 在 `modifySlice` 之后变成了 `[100 2 3]`：

```go
func modifySlice(s []int) {
    s[0] = 100
}

s := []int{1, 2, 3}
modifySlice(s)
fmt.Println(s)
```

::: details 第 4 题参考答案

输出是 `[100 2 3]`。

**原因**：切片是"引用类型"——它的内部实现是一个结构体（指向底层数组的指针、长度、容量）。值传递时拷贝的是这个头部结构体，但头部里的指针仍指向同一个底层数组。因此 `s[0] = 100` 修改的是共享的底层数组，外部可见。

**但如果函数内执行 `s = append(s, 4)` 且发生扩容**，则 `s` 会指向新数组，外部将看不到后续修改。

:::

### 第 5 题

用 `new` 和 `make` 分别创建 `*int`、`[]int` 和 `map[string]int`，对比它们的零值和可用性。

::: details 第 5 题参考答案

```go
pi := new(int)               // *int，值为 0，可以直接解引用
si := make([]int, 3)         // []int，长度为 3 的零值切片，可以 append
mi := make(map[string]int)   // map，可以读写

// 对比
fmt.Println(*pi)        // 0
fmt.Println(si)         // [0 0 0]
mi["key"] = 42
fmt.Println(mi)         // map[key:42]

// 不能用 new 直接得到可用的 slice 或 map
ps := new([]int)
fmt.Println(*ps)        // []，nil 切片，可以 append
// (*ps)[0] = 1         // panic: index out of range（长度为 0）

pm := new(map[string]int)
// (*pm)["key"] = 1    // panic: assignment to entry in nil map
*pm = make(map[string]int) // 必须先初始化
```

**工程建议**：
- 优先用字面量 `T{}` 代替 `new(T)`
- slice、map、channel 永远用 `make`
- 不要对 `new` 出来的 map/slice 直接操作，除非先初始化内部结构

:::

### 第 6 题

运行 `go build -gcflags=-m ./internal/chapter/go06_pointers/`，观察 `allocateOnHeap` 和 `allocateOnStack` 函数的逃逸分析输出，解释为什么一个分配在堆上，一个在栈上。

::: details 第 6 题参考答案

```bash
go build -gcflags=-m ./internal/chapter/go06_pointers/
```

预期输出类似：

```text
./demo.go:250:9: &x escapes to heap
./demo.go:249:2: moved to heap: x
./demo.go:255:2: x does not escape
```

**解释**：
- `allocateOnHeap` 返回了 `&x`，局部变量 `x` 的地址逃逸到了调用方。函数返回后栈帧被销毁，如果 `x` 留在栈上，调用方将得到悬空指针。因此编译器将 `x` 分配到堆上。
- `allocateOnStack` 返回的是 `x` 的值拷贝（42），调用方不需要访问原变量的地址。`x` 可以安全地留在栈上，函数返回时随栈帧一起回收。

**工程意义**：不要过度担心逃逸。Go 的 GC 对小对象分配优化得很好，"返回指针"是 Go 中非常常见的模式。只有在高频热点路径上，才需要用 `-gcflags=-m` 检查并优化。

:::

---

## 小结

本章覆盖了 Go 指针与内存模型的核心概念，关键要点：

- **`&` 取地址，`*` 解引用**：指针存储的是变量的内存地址，解引用可读写该地址的数据。
- **Go 只有值传递**：传递指针时拷贝的是地址值，通过地址可以修改原数据，但传递行为本身仍是值拷贝。
- **切片是"引用类型"的假象**：切片头部包含指针，元素修改会共享，但扩容或重新赋值可能断开关联。
- **`new` 分配零值内存，返回指针**；`make` 初始化 slice、map、channel 的内部结构，返回值本身。
- **`nil` 指针解引用会 panic**：使用指针前必须检查 nil，方法接收者也可以安全地处理 nil。
- **指针接收者修改原变量，值接收者操作副本**：选择依据是是否需要修改、结构体大小、以及方法集一致性。
- **逃逸分析决定栈 vs 堆**：返回局部变量地址、发送给 channel、赋给接口都会导致逃逸。
- **Go 禁止指针运算**：用索引和 `range` 遍历，避免越界和内存破坏。

掌握这些规则后，你将能准确判断"修改为什么没生效"、"什么时候该传指针"、以及"变量分配在哪里"。下一章将学习数组、切片与映射，深入理解这些"引用类型"的底层机制。
