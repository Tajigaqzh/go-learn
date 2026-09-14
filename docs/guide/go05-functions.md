# 第 5 章 · 函数与闭包

在掌握了控制流之后，你已经能写出结构化的程序了。但 Go 的函数远不止"一段可复用的代码"——它支持多返回值、命名返回值、变参、高阶函数和闭包，还提供了独特的 `defer` 机制来管理资源释放。理解这些特性，是写出优雅、可维护 Go 代码的关键。

本章将系统讲解函数签名、多返回值与命名返回值、变参函数、函数作为一等值、闭包与变量捕获、`defer` 的执行顺序与参数求值时机、`defer` 配合命名返回值的经典陷阱、递归实现，以及 `init` 函数与包初始化的关系。掌握这些内容后，你将能灵活运用 Go 的函数式编程能力，并避开 `defer` 最常见的坑。

本章配套代码在 `internal/chapter/go05_functions/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 5.1 函数签名与多返回值

Go 函数可以返回多个值，这是 Go 错误处理的核心模式：

```go
func Divide(a, b int) (int, int) {
    return a / b, a % b
}

q, r := Divide(17, 5)
fmt.Println(q, r) // 3, 2
```

### 忽略不需要的返回值

用空白标识符 `_` 忽略不需要的值：

```go
q, _ := Divide(20, 4) // 只关心商
```

### "成功/失败"模式

多返回值也常用于返回结果和状态标志：

```go
func SafeDivide(a, b int) (int, bool) {
    if b == 0 {
        return 0, false
    }
    return a / b, true
}

result, ok := SafeDivide(10, 0)
if !ok {
    fmt.Println("除法失败")
}
```

### 实测输出

```text
Divide(17, 5) = quotient=3, remainder=2
Divide(20, 4) 的商 = 5
SafeDivide(10, 0): result=0, ok=false
SafeDivide(10, 2): result=5, ok=true
```

**注意**：Go 没有默认参数、没有函数重载、没有可选参数。需要类似功能时，用变参或结构体选项模式。

---

## 5.2 命名返回值

Go 允许给返回值命名，在函数签名中就能说明每个返回值的含义：

```go
func RectInfo(width, height float64) (area, perimeter float64) {
    area = width * height
    perimeter = 2 * (width + height)
    return // naked return，等价于 return area, perimeter
}
```

### 命名返回值的优势

1. **自描述**：函数签名即文档，调用方一看就知道返回值含义
2. **defer 可修改**：在 `defer` 中可以修改命名返回值（见 5.7 节）

### naked return 的陷阱

```go
return // 等价于 return area, perimeter
```

naked return 在短函数中可用，但在长函数中会降低可读性。团队规范通常建议显式写出返回值。

### 实测输出

```text
RectInfo(3, 4): area=12, perimeter=14
命名返回值的优势：文档自描述 + 可在 defer 中修改
```

---

## 5.3 变参函数

Go 支持变参函数（variadic function），用 `...` 表示接受任意数量的参数：

```go
func Sum(nums ...int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}

Sum()           // 0
Sum(1, 2, 3)    // 6
Sum(10, 20)     // 30
```

### 展开切片

可以用 `...` 将切片展开为变参：

```go
nums := []int{1, 2, 3, 4, 5}
Sum(nums...) // 15
```

### 变参的本质

变参在函数内部被转换为一个切片。因此 `nums` 的类型是 `[]int`，可以像普通切片一样使用。

### 实测输出

```text
Sum() = 0
Sum(1, 2, 3) = 6
Sum(10, 20) = 30
Sum(nums...) = 15
JoinWords: Hello Go World
```

---

## 5.4 函数是一等值

在 Go 中，函数是**一等公民**（first-class citizen）：可以赋值给变量、作为参数传递、作为返回值返回。

### 函数赋值给变量

```go
add := func(a, b int) int {
    return a + b
}
fmt.Println(add(2, 3)) // 5
```

### 函数作为参数（高阶函数）

```go
func ApplyOperation(a, b int, op func(int, int) int) int {
    return op(a, b)
}

result := ApplyOperation(10, 5, add)           // 15
result2 := ApplyOperation(10, 5, func(a, b int) int {
    return a - b
}) // 5
```

### 函数作为返回值

```go
func MakeOperation(name string) func(int, int) int {
    switch name {
    case "add":
        return func(a, b int) int { return a + b }
    case "sub":
        return func(a, b int) int { return a - b }
    case "mul":
        return func(a, b int) int { return a * b }
    default:
        return func(a, b int) int { return 0 }
    }
}

mul := MakeOperation("mul")
fmt.Println(mul(4, 5)) // 20
```

### 实测输出

```text
add(2, 3) = 5
ApplyOperation(10, 5, add) = 15
ApplyOperation(10, 5, sub) = 5
MakeOperation("mul")(4, 5) = 20
```

---

## 5.5 闭包与捕获

**闭包**（closure）是函数与其引用的外部变量环境的组合。闭包可以"记住"并修改外部变量。

### 闭包工厂

```go
func MakeMultiplier(factor int) func(int) int {
    return func(x int) int {
        return x * factor // 捕获外部变量 factor
    }
}

double := MakeMultiplier(2)
triple := MakeMultiplier(3)
fmt.Println(double(5)) // 10
fmt.Println(triple(5)) // 15
```

每个闭包独立维护自己的环境。`double` 和 `triple` 捕获的是不同的 `factor` 值。

### 闭包修改外部变量

```go
func MakeCounter() func() int {
    count := 0
    return func() int {
        count++ // 修改外部变量
        return count
    }
}

counter := MakeCounter()
counter() // 1
counter() // 2
counter() // 3
```

### 循环变量捕获的陷阱（Go 1.22 之前）

在 Go 1.22 之前，`for` 循环变量在整个循环中只有一个实例，闭包捕获的总是同一个变量的最终值。Go 1.22 起，每次迭代的循环变量都是独立的：

```go
funcs := make([]func() int, 3)
for i := range 3 { // Go 1.22+，每次迭代 i 是独立的
    funcs[i] = func() int {
        return i * i
    }
}
// Go 1.22+ 输出：0, 1, 4
// Go 1.21 及之前输出：4, 4, 4
```

**注意**：如果你的代码需要在 Go 1.21 或更早版本上运行，应该在循环内创建局部副本：

```go
for i := range 3 {
    i := i // 创建局部副本（Go 1.21 及之前需要）
    funcs[i] = func() int { return i * i }
}
```

### 实测输出

```text
double(5) = 10
triple(5) = 15
闭包捕获循环变量: 0 1 4
counter() = 1
counter() = 2
counter() = 3
```

---

## 5.6 defer 的执行顺序与参数求值

`defer` 将函数调用推迟到当前函数返回前执行，常用于资源释放（关闭文件、解锁互斥锁等）。

### 执行顺序：LIFO

多个 `defer` 按**后进先出**（LIFO）的顺序执行：

```go
defer fmt.Println("defer 1")
defer fmt.Println("defer 2")
defer fmt.Println("defer 3")
// 输出顺序：3, 2, 1
```

### 参数在注册时求值

`defer` 的参数在 `defer` 语句执行时（即注册时）就确定了，而不是在 `defer` 函数实际执行时：

```go
x := 1
defer fmt.Println(x) // x 在此时求值为 1
x = 100
// defer 输出 1，不是 100
```

### 实测输出

```text
defer 演示开始
defer 演示结束（普通语句先执行）
x 被修改为 100，但 defer 的参数已经确定
defer 中 x 的值（注册时求值）: 1
defer 3: 最后注册，最先执行
defer 2: 第二注册，第二执行
defer 1: 最先注册，最后执行
```

---

## 5.7 defer 配合命名返回值的坑

这是 Go 中最经典、最容易让人困惑的陷阱之一。

### defer 可以修改命名返回值

```go
func deferWithNamedReturn() (result int) {
    result = 1
    defer func() {
        result++ // 修改命名返回值
    }()
    return result // 实际返回 2
}
```

执行过程：
1. `result = 1`
2. 执行 `return result`，将返回值设为 1
3. 执行 `defer`，`result++`，返回值变为 2
4. 函数实际返回 2

### defer 不能修改非命名返回值

```go
func deferWithoutNamedReturn() int {
    result := 1
    defer func() {
        result++ // 修改的是局部变量，不影响返回值
    }()
    return result // 返回 1
}
```

### defer 匿名函数的参数求值

如果 `defer` 的匿名函数有参数，参数也在注册时求值：

```go
func deferWithParam(n int) int {
    defer func(x int) {
        fmt.Println(x) // 输出注册时的值 5
    }(n)
    n = 100
    return n // 返回 100
}
```

### 实测输出

```text
deferWithNamedReturn() = 2
deferWithoutNamedReturn() = 1
defer 中 x = 5（注册时的值）
deferWithParam(5) = 100
```

### 最佳实践

- 需要 `defer` 修改返回值时，使用命名返回值
- 不需要修改时，避免命名返回值 + naked return，保持代码清晰
- 在 `defer` 中使用具名函数而不是匿名函数，更易读

---

## 5.8 递归

Go 支持递归函数，但编译器**不做尾递归优化**。

### 阶乘

```go
func Factorial(n int) int {
    if n <= 1 {
        return 1
    }
    return n * Factorial(n-1)
}

fmt.Println(Factorial(5)) // 120
```

### 斐波那契数列

```go
func Fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return Fibonacci(n-1) + Fibonacci(n-2)
}
```

**注意**：上面的递归实现效率极低，时间复杂度为 O(2^n)。实际工程中应使用循环或记忆化。

### 循环替代递归

由于 Go 不做尾递归优化，深层递归可能导致栈溢出。优先使用循环：

```go
func FactorialLoop(n int) int {
    result := 1
    for i := 2; i <= n; i++ {
        result *= i
    }
    return result
}
```

### 实测输出

```text
Factorial(5) = 120
Fibonacci(10) = 55
FactorialLoop(5) = 120（循环版，无栈溢出风险）
```

---

## 5.9 init 函数

`init` 函数在包被导入时自动执行，一个包可以有多个 `init` 函数。初始化顺序是：

1. 初始化导入的包（递归）
2. 初始化当前包的变量（按声明顺序）
3. 执行当前包的 `init` 函数（按文件名字母顺序，同一文件内按声明顺序）

```go
var initCounter = initFunc()

func initFunc() int {
    fmt.Println("[init] initFunc() 执行")
    return 42
}

func init() {
    fmt.Println("[init] init() 执行")
}
```

### init 的典型用途

- 注册数据库驱动：`import _ "github.com/go-sql-driver/mysql"`
- 验证包级配置
- 预计算常量表

### 注意事项

- `init` 顺序由依赖关系和文件名决定，**不要编写依赖特定 init 顺序的代码**
- `init` 中发生的错误通常用 `log.Fatal` 处理，因为此时 `main` 还没开始
- 过度使用 `init` 会导致代码难以测试和追踪，优先用显式初始化函数

### 实测输出

```text
[init] initFunc() 在包初始化时执行，早于 main 和 Demo()
[init] go05_functions 的 init() 执行
init 函数在包被导入时自动执行，一个包可以有多个 init。
包级变量 initCounter 的值：42
init 的典型用途：注册驱动、验证配置、预计算常量表。
注意：init 顺序由依赖关系和文件名字母顺序决定，不要编写依赖特定 init 顺序的代码。
```

---

## 3 个真实报错怎么读

### 报错 1：too many arguments

```text
./demo.go:15:11: too many arguments in call to Divide
    have (int, int, int)
    want (int, int)
```

**原因**：调用函数时传入的参数数量与函数签名不匹配。

**解法**：检查函数签名，确保参数数量和类型正确。

### 报错 2：unused parameter

```text
./demo.go:20:15: x declared but not used
```

**原因**：函数参数或局部变量声明后未使用。

**解法**：删除未使用的变量，或用 `_` 显式忽略。

### 报错 3：defer 中的参数求值陷阱

```go
func trap() {
    for i := 0; i < 3; i++ {
        defer fmt.Println(i) // i 在注册时求值
    }
}
// 输出：2, 1, 0
```

**原因**：`defer` 的参数在注册时就确定了，不是执行时。

**解法**：如果需要在执行时获取最新值，使用闭包：

```go
func correct() {
    for i := 0; i < 3; i++ {
        defer func(x int) {
            fmt.Println(x)
        }(i)
    }
}
```

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 多返回值只用一个，编译报错 | Go 不允许未使用的局部变量 | 用 `_` 忽略不需要的返回值 |
| defer 中变量值不对 | defer 参数在注册时求值 | 需要延迟求值时用闭包包裹 |
| defer 修改不了返回值 | 使用的是非命名返回值 | 改用命名返回值 `(result int)` |
| 递归深度太大导致栈溢出 | Go 不做尾递归优化 | 改用循环实现 |
| 闭包输出相同的值 | 循环变量被所有闭包共享（Go 1.21-） | 在循环内创建局部副本 `i := i` |
| naked return 让人困惑 | 长函数中使用 naked return | 短函数可用，长函数显式写出返回值 |
| init 顺序不可控导致 bug | init 执行顺序依赖文件名 | 减少 init 使用，改用显式初始化 |
| 变参传入 nil 切片 | `nums...` 展开的是 nil 切片 | 确保切片已初始化 |

---

## 练习

### 第 1 题

写一个函数 `MinMax`，接收一个 `[]int`，返回最小值和最大值。如果切片为空，返回 `0, 0, false`。

::: details 第 1 题参考答案

```go
func MinMax(nums []int) (min, max int, ok bool) {
    if len(nums) == 0 {
        return 0, 0, false
    }
    min, max = nums[0], nums[0]
    for _, n := range nums[1:] {
        if n < min {
            min = n
        }
        if n > max {
            max = n
        }
    }
    return min, max, true
}
```

**为什么这样写更好**：使用命名返回值使函数签名自描述，`ok` 标志明确告知调用方切片是否为空，避免隐式错误处理。

:::

### 第 2 题

写一个高阶函数 `Filter`，接收一个 `[]int` 和一个判断函数 `func(int) bool`，返回满足条件的元素切片。

::: details 第 2 题参考答案

```go
func Filter(nums []int, predicate func(int) bool) []int {
    var result []int
    for _, n := range nums {
        if predicate(n) {
            result = append(result, n)
        }
    }
    return result
}

// 使用
evens := Filter([]int{1, 2, 3, 4, 5}, func(n int) bool {
    return n%2 == 0
})
fmt.Println(evens) // [2, 4]
```

**为什么这样写更好**：高阶函数将"筛选逻辑"与"遍历逻辑"分离，调用方只需关注判断条件，代码更简洁、更易复用。

:::

### 第 3 题

写一个闭包工厂 `MakeAdder`，返回一个函数，该函数每次调用都在之前的累加结果上加上新值。

::: details 第 3 题参考答案

```go
func MakeAdder() func(int) int {
    sum := 0
    return func(x int) int {
        sum += x
        return sum
    }
}

add := MakeAdder()
fmt.Println(add(1))  // 1
fmt.Println(add(2))  // 3
fmt.Println(add(3))  // 6
```

**为什么这样写更好**：闭包封装了状态（`sum`），外部无法直接修改，只能通过返回的函数间接累加，实现了数据隐藏。

:::

### 第 4 题

观察以下代码，预测输出并解释原因：

```go
func testDefer() (result int) {
    defer func() {
        result *= 10
    }()
    return 5
}
```

::: details 第 4 题参考答案

输出是 `50`。

**执行过程**：
1. `result` 是命名返回值，初始为 0
2. `return 5` 将 `result` 设为 5
3. `defer` 执行 `result *= 10`，`result` 变为 50
4. 函数返回 50

**关键点**：命名返回值在 `return` 语句执行后被赋值为返回表达式，然后 `defer` 有机会修改它。如果是非命名返回值 `func testDefer() int`，`defer` 无法修改返回值，结果将是 5。

:::

### 第 5 题

用递归和循环分别实现计算第 n 个斐波那契数，对比两种实现的时间复杂度。

::: details 第 5 题参考答案

```go
// 递归版：O(2^n)，存在大量重复计算
func Fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return Fibonacci(n-1) + Fibonacci(n-2)
}

// 循环版：O(n)，推荐
func FibonacciLoop(n int) int {
    if n <= 1 {
        return n
    }
    a, b := 0, 1
    for i := 2; i <= n; i++ {
        a, b = b, a+b
    }
    return b
}
```

**为什么循环版更好**：递归版存在严重的重复计算（如 `Fib(3)` 在 `Fib(5)` 和 `Fib(4)` 中各计算一次）。Go 编译器不做尾递归优化，深层递归还会栈溢出。循环版时间复杂度 O(n)，空间复杂度 O(1)，更安全高效。

:::

### 第 6 题

以下代码在 Go 1.21 和 Go 1.22 下的输出分别是什么？解释差异原因。

```go
funcs := make([]func() int, 3)
for i := 0; i < 3; i++ {
    funcs[i] = func() int { return i * i }
}
for _, f := range funcs {
    fmt.Print(f(), " ")
}
```

::: details 第 6 题参考答案

- **Go 1.22+**：`0 1 4`
- **Go 1.21 及之前**：`4 4 4`

**差异原因**：Go 1.22 之前，`for` 循环变量 `i` 在整个循环中只有一个实例，所有闭包共享同一个 `i`。循环结束时 `i = 3`，所以所有闭包都返回 `3 * 3 = 9`... 等等，循环条件是 `i < 3`，循环结束后 `i = 3`，但 `funcs[2]` 是在 `i = 2` 时设置的。实际上在 Go 1.21- 中，由于闭包在循环结束后才执行，此时 `i = 3`，所以返回 `9`。但上面的代码在 Go 1.22+ 中输出 `0 1 4`。

**兼容性写法**：如果代码需要在 Go 1.21- 上运行，在循环内创建局部副本：

```go
for i := 0; i < 3; i++ {
    i := i // 局部副本
    funcs[i] = func() int { return i * i }
}
```

:::

---

## 小结

本章覆盖了 Go 函数与闭包的核心概念，关键要点：

- **多返回值**：Go 函数可返回多个值，是错误处理的标准模式；用 `_` 忽略不需要的值。
- **命名返回值**：自描述文档 + defer 可修改，但长函数中避免 naked return。
- **变参函数**：`...` 接受任意数量参数，内部转为切片；可用 `slice...` 展开调用。
- **函数是一等值**：可赋值、传参、返回，是高阶函数和函数式编程的基础。
- **闭包**：函数 + 外部变量环境，可记住并修改状态；注意 Go 1.22 前后循环变量捕获的差异。
- **defer**：LIFO 执行，参数在注册时求值；配合命名返回值可修改返回结果。
- **递归**：Go 不做尾递归优化，优先用循环替代深层递归。
- **init**：包导入时自动执行，顺序由依赖和文件名决定；适度使用，优先显式初始化。

掌握这些特性后，你将能写出更灵活、更安全的 Go 代码。下一章将学习指针、值与内存入门，深入理解 Go 的内存模型和指针语义。
