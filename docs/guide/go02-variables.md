# 第 2 章 · 变量、常量与基本类型

Go 的类型系统简洁而严格：所有变量都必须有明确的类型，不同类型之间不能隐式转换。这种设计虽然在初期会让习惯了动态语言或宽松类型转换的开发者感到不适，但它能在编译期就暴露大量潜在错误，避免运行时的类型混乱。

本章将系统讲解变量声明、零值初始化、常量与 `iota` 生成器、整数族与浮点数的边界行为、`byte` 与 `rune` 的 Unicode 处理，以及类型定义与类型别名的区别。掌握这些基础后，你将能准确控制数据的表示范围、避免溢出与精度陷阱，并写出类型安全的 Go 代码。

本章配套代码在 `internal/chapter/go02_variables/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 2.1 var 与 := 声明

Go 提供了三种变量声明方式：

### 方式一：`var` 显式类型

```go
var a int = 10
var name string = "Alice"
```

明确指定类型，适合需要强调类型或初始值类型不明确的场景。

### 方式二：`var` 类型推导

```go
var b = 20       // 推导为 int
var pi = 3.14    // 推导为 float64
```

编译器根据右侧表达式推导类型，简洁且类型安全。

### 方式三：短声明 `:=`（仅函数内）

```go
c := 30
active := true
```

`:=` 是最常用的函数内声明方式，简洁高效。**注意**：`:=` 只能在函数内使用，包级别变量必须用 `var`。

### 批量声明

```go
// 同类型批量
var x, y, z int = 1, 2, 3

// 混合类型批量
var name, age, active = "Alice", 25, true

// 块式声明（包级别常用）
var (
    host = "localhost"
    port = 8080
)
```

### `:=` 的重新赋值规则

`:=` 要求至少有一个新变量，已存在的变量会被重新赋值：

```go
d := 40
d, e := 50, 60  // d 被重新赋值为 50，e 是新变量
```

---

## 2.2 零值

Go 的所有变量在声明时都会被自动初始化为**零值**（zero value），不存在未初始化变量：

| 类型 | 零值 | 说明 |
| --- | --- | --- |
| `int`, `int8`, `int64`, `uint` 等 | `0` | 所有整数类型 |
| `float32`, `float64` | `0.0` | 浮点数 |
| `bool` | `false` | 布尔值 |
| `string` | `""` | 空字符串（不是 `nil`） |
| 指针、切片、map、channel、函数 | `nil` | 引用类型 |

### 零值的可用性

- `string` 零值是空字符串，可以直接拼接：`s := ""; s = s + "hello"`
- `nil` 切片可以直接 `append`：`var s []int; s = append(s, 1, 2, 3)`
- `nil` map **不能**直接赋值，会 panic：`var m map[string]int; m["key"] = 1  // panic`

必须先用 `make` 初始化 map：

```go
m := make(map[string]int)
m["key"] = 1  // OK
```

---

## 2.3 作用域与遮蔽

Go 的作用域规则清晰：花括号 `{}` 创建新作用域，内层可以声明与外层同名的变量，形成**遮蔽**（shadowing）。

### 基本遮蔽

```go
x := 10
fmt.Println(x)  // 10

{
    x := 20  // 内层新变量，遮蔽外层
    fmt.Println(x)  // 20
    x = 25  // 修改内层 x
}

fmt.Println(x)  // 10，外层未受影响
```

### `if` 初始化语句的作用域

```go
if y := 100; y > 50 {
    fmt.Println(y)  // 100
} else {
    fmt.Println(y)  // 100，else 块也能访问
}
// fmt.Println(y)  // 编译错误：undefined: y
```

`y` 的作用域仅限 `if-else` 块。

### `for` 循环变量作用域

Go 1.22 之前，`for` 循环变量在整个循环中只有一个实例，容易在闭包中捕获错误的值。Go 1.22 起，每次迭代的循环变量都是独立的：

```go
for i := 0; i < 3; i++ {
    fmt.Println(i)  // 每次迭代的 i 是独立的
}
// fmt.Println(i)  // 编译错误：undefined: i
```

---

## 2.4 const 与 iota

### 常量声明

常量必须在**编译期**确定，不能是运行时计算的值：

```go
const pi = 3.14159
const greeting = "Hello"

const (
    StatusOK    = 200
    StatusError = 500
)
```

常量可以是整数、浮点数、字符串、布尔值，但不能是切片、map、指针等引用类型。

### `iota` 常量生成器

`iota` 是 Go 的常量生成器，从 0 开始自动递增：

```go
const (
    Sunday = iota  // 0
    Monday         // 1
    Tuesday        // 2
    Wednesday      // 3
)
```

### `iota` 跳过与表达式

```go
const (
    _  = iota             // 0，跳过
    KB = 1 << (10 * iota) // 1 << 10 = 1024
    MB                    // 1 << 20 = 1048576
    GB                    // 1 << 30 = 1073741824
)
```

每个 `const` 块的 `iota` 会重置为 0。

### 无类型常量

Go 的常量是**无类型**的，具有高精度，可以赋值给任何兼容类型：

```go
const small = 1
var i8 int8 = small   // OK
var i64 int64 = small // OK

const big = 1e100
var f float64 = big   // OK
// var i int = big    // 编译错误：constant 1e+100 overflows int
```

无类型常量在赋值时才确定具体类型，这是 Go 编译期类型推导的重要特性。

---

## 2.5 整数类型

Go 提供了多种整数类型，按位宽和有无符号分类：

| 类型 | 位宽 | 范围 |
| --- | --- | --- |
| `int8` | 8 位 | -128 ~ 127 |
| `uint8` (`byte`) | 8 位 | 0 ~ 255 |
| `int16` | 16 位 | -32768 ~ 32767 |
| `uint16` | 16 位 | 0 ~ 65535 |
| `int32` (`rune`) | 32 位 | -2^31 ~ 2^31-1 |
| `uint32` | 32 位 | 0 ~ 2^32-1 |
| `int64` | 64 位 | -2^63 ~ 2^63-1 |
| `uint64` | 64 位 | 0 ~ 2^64-1 |
| `int` | 32 或 64 位 | 取决于平台 |
| `uint` | 32 或 64 位 | 取决于平台 |

### `int` 和 `uint` 的大小

`int` 和 `uint` 的大小由编译目标平台决定：

- 32 位平台：4 字节（等价于 `int32`/`uint32`）
- 64 位平台：8 字节（等价于 `int64`/`uint64`）

**最佳实践**：优先使用 `int`，除非有明确的位宽需求（如网络协议、文件格式）。

### 整数溢出

编译期常量溢出会报错，运行时溢出会**回绕**（wrap around）：

```go
const overflow int8 = 128  // 编译错误：constant 128 overflows int8

var x int8 = 127
x = x + 1  // 运行时溢出，回绕为 -128

var y uint8 = 255
y = y + 1  // 回绕为 0
```

Go 不检查运行时整数溢出，开发者需要自行保证运算范围。

### 位运算

Go 支持完整的位运算，并提供二进制、八进制、十六进制字面量：

```go
a := 0b1010  // 二进制，10
b := 0o12    // 八进制，10
c := 0xA     // 十六进制，10

fmt.Println(a & b)   // 按位与
fmt.Println(a | b)   // 按位或
fmt.Println(a ^ b)   // 按位异或
fmt.Println(a << 1)  // 左移
fmt.Println(a >> 1)  // 右移
```

---

## 2.6 浮点数与复数

### 浮点类型

Go 提供 `float32` 和 `float64` 两种浮点类型，遵循 IEEE 754 标准：

```go
var f32 float32 = 3.14
var f64 float64 = 2.718281828459045
```

**最佳实践**：优先使用 `float64`，除非内存受限（如大规模数组）。

### 浮点精度陷阱

浮点数无法精确表示所有小数，二进制表示下 0.1 和 0.2 是无限循环小数：

```go
a := 0.1
b := 0.2
c := a + b
fmt.Println(c == 0.3)  // false！
fmt.Printf("%.20f\n", c)  // 0.30000000000000004441
```

**正确做法**：使用 epsilon 比较：

```go
epsilon := 1e-9
if math.Abs(c - 0.3) < epsilon {
    // 认为相等
}
```

### 特殊浮点值

Go 支持正无穷、负无穷、NaN（Not a Number）：

```go
fmt.Println(math.Inf(1))   // +Inf
fmt.Println(math.Inf(-1))  // -Inf
fmt.Println(math.NaN())    // NaN

nan := math.NaN()
fmt.Println(nan == nan)    // false！NaN 不等于任何值，包括自身
fmt.Println(math.IsNaN(nan))  // true，应使用 IsNaN 检测
```

### 复数

Go 原生支持复数，提供 `complex64` 和 `complex128` 两种类型：

```go
z1 := 1 + 2i
z2 := complex(3, 4)  // 等价于 3 + 4i
fmt.Println(z1 + z2)  // (4+6i)
fmt.Println(real(z1), imag(z1))  // 1, 2
```

复数在科学计算和信号处理中有广泛应用。

---

## 2.7 byte 与 rune

### `byte` 是 `uint8` 的别名

`byte` 用于表示 ASCII 字符或原始字节：

```go
var b byte = 'A'  // 单引号表示字符字面量
fmt.Printf("%d\n", b)  // 65（ASCII 码）
fmt.Printf("%c\n", b)  // A
```

### `rune` 是 `int32` 的别名

`rune` 用于表示 Unicode 码点：

```go
var r rune = '中'
fmt.Printf("%d\n", r)   // 20013（Unicode 码点 U+4E2D）
fmt.Printf("%c\n", r)   // 中
fmt.Printf("U+%04X\n", r)  // U+4E2D
```

### 字符串遍历的陷阱

Go 的字符串是 UTF-8 编码的字节序列，`len()` 返回的是**字节数**，不是字符数：

```go
s := "Go语言"
fmt.Println(len(s))  // 8（Go=2 字节，语=3 字节，言=3 字节）

// 按字节遍历：错误方式
for i := 0; i < len(s); i++ {
    fmt.Printf("%c ", s[i])  // G o è¯ è¨（乱码）
}

// 按 rune 遍历：正确方式
for _, r := range s {
    fmt.Printf("%c ", r)  // G o 语 言
}
```

**转换为 `[]rune` 获取字符数**：

```go
runes := []rune(s)
fmt.Println(len(runes))  // 4
fmt.Printf("%c\n", runes[2])  // 语
```

---

## 2.8 类型转换

Go **不支持隐式类型转换**，所有类型转换必须显式进行：

```go
var i int = 42
var f float64 = float64(i)
var u uint = uint(i)
```

### 浮点转整数：截断小数

```go
var pi float64 = 3.14159
var intPi int = int(pi)  // 3，直接截断小数部分
```

### 有损转换

大类型转小类型可能发生溢出或截断：

```go
var big int64 = 1000
var small int8 = int8(big)  // 截断为低 8 位，结果是 -24
```

### `string` 与 `[]byte`、`[]rune` 转换

```go
s := "Hello"
bytes := []byte(s)   // [72 101 108 108 111]
runes := []rune(s)   // [72 101 108 108 111]

s2 := string(bytes)  // "Hello"
s3 := string(runes)  // "Hello"
```

**注意**：`string` 与 `[]byte` 的转换会复制数据，对于大字符串有性能开销。

### 不同类型不能直接运算

```go
var a int = 10
var b int64 = 20
// c := a + b  // 编译错误：mismatched types int and int64
c := int64(a) + b  // 必须显式转换
```

---

## 2.9 类型定义与类型别名

Go 提供两种自定义类型的方式：

### 类型定义（Type Definition）

创建一个**新类型**，与原类型不兼容：

```go
type Celsius float64
type Fahrenheit float64

var c Celsius = 100.0
// var f float64 = c  // 编译错误：cannot use c (type Celsius) as type float64
var f float64 = float64(c)  // 必须显式转换
```

新类型可以定义自己的方法，实现类型安全。

### 类型别名（Type Alias）

为现有类型创建别名，完全等价：

```go
type MyInt = int  // 注意等号

var a MyInt = 10
var b int = 20
c := a + b  // 可以直接运算，MyInt 和 int 完全等价
```

**预定义的类型别名**：

```go
type byte = uint8
type rune = int32
```

---

## 2.10 空白标识符

`_`（下划线）是 Go 的**空白标识符**，用于忽略不需要的值：

### 忽略函数返回值

```go
x, _ := getValue()  // 只要第一个返回值
_, y := getValue()  // 只要第二个返回值
```

### 空白导入

执行包的 `init` 函数但不使用包名：

```go
import _ "database/sql/driver"
```

### `for range` 忽略索引或值

```go
nums := []int{10, 20, 30}

// 只要值
for _, v := range nums {
    fmt.Println(v)
}

// 只要索引
for i := range nums {
    fmt.Println(i)
}
```

### 类型断言

只检查类型，不使用值：

```go
var i interface{} = 42
_, ok := i.(int)
fmt.Println(ok)  // true
```

---

## 3 个真实报错怎么读

### 报错 1：常量溢出

```
./main.go:10:7: constant 128 overflows int8
```

**原因**：`int8` 范围是 -128 ~ 127，常量 128 超出范围。

**解法**：改用 `int16` 或更大类型，或者用运行时变量（会回绕）。

### 报错 2：类型不匹配

```
./main.go:15:9: invalid operation: a + b (mismatched types int and int64)
```

**原因**：`int` 和 `int64` 是不同类型，不能直接运算。

**解法**：显式转换为同一类型：`int64(a) + b` 或 `a + int(b)`。

### 报错 3：nil map 赋值

```
panic: assignment to entry in nil map
```

**原因**：未初始化的 map 是 `nil`，不能直接赋值。

**解法**：先用 `make` 初始化：`m := make(map[string]int)`，然后再赋值。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `len(s)` 不等于字符数 | `len` 返回字节数，中文占多个字节 | 用 `len([]rune(s))` 获取字符数 |
| `0.1 + 0.2 != 0.3` | 浮点精度问题 | 用 epsilon 比较：`math.Abs(c - 0.3) < 1e-9` |
| `int8` 溢出变负数 | 有符号整数回绕 | 用更大类型或检查边界 |
| `nil map` panic | map 零值是 `nil`，不能直接赋值 | 先 `make(map[K]V)`，再赋值 |
| `NaN == NaN` 是 `false` | NaN 不等于任何值 | 用 `math.IsNaN()` 检测 |
| `int` 和 `int64` 不能混用 | Go 不做隐式转换 | 显式转换：`int64(i)` |
| `:=` 在包级别报错 | 短声明只能在函数内 | 包级别用 `var` |

---

## 练习

### 第 1 题

声明一个 `int8` 变量 `x`，赋值为 100，然后加 30，打印结果。观察是否溢出，如果溢出，解释原因。

::: details 第 1 题参考答案

```go
var x int8 = 100
x = x + 30  // 130 超出 int8 范围（127），回绕为 -126
fmt.Println(x)  // -126
```

**原因**：`int8` 最大值是 127，130 二进制是 `10000010`，有符号解释为 -126。

**更好的做法**：如果预期值可能超过 127，应该用 `int16` 或 `int`。

:::

### 第 2 题

写一个函数，计算两个 `float64` 是否"近似相等"（差距小于 `1e-9`），而不是直接用 `==` 比较。

::: details 第 2 题参考答案

```go
func almostEqual(a, b float64) bool {
    return math.Abs(a - b) < 1e-9
}

fmt.Println(almostEqual(0.1+0.2, 0.3))  // true
fmt.Println(0.1+0.2 == 0.3)             // false
```

**为什么这样写更好**：浮点数受精度限制，直接 `==` 比较会因舍入误差失败。工程中应该用 epsilon 比较。

:::

### 第 3 题

声明一个字符串 `s := "Go语言"`，分别用 `len(s)` 和 `len([]rune(s))` 打印长度，解释差异。

::: details 第 3 题参考答案

```go
s := "Go语言"
fmt.Println(len(s))         // 8（字节数）
fmt.Println(len([]rune(s))) // 4（字符数）
```

**差异原因**：`len(s)` 返回 UTF-8 编码的字节数，中文字符通常占 3 字节。`[]rune(s)` 转换为 Unicode 码点数组，每个字符一个元素。

**工程意义**：遍历字符串时应该用 `for _, r := range s` 按字符遍历，而不是按索引 `s[i]`。

:::

### 第 4 题

用 `iota` 定义一组网络状态常量：`StatusOK=200`, `StatusCreated=201`, `StatusBadRequest=400`, `StatusUnauthorized=401`。

::: details 第 4 题参考答案

```go
const (
    StatusOK = 200 + iota  // 200
    StatusCreated          // 201
    _                      // 202，跳过
    _                      // 203，跳过
    // ... 跳到 400 需要手动设置
)

// 实际更好的做法是分段定义
const (
    StatusOK      = 200
    StatusCreated = 201
)

const (
    StatusBadRequest   = 400
    StatusUnauthorized = 401
)
```

**为什么分段更好**：HTTP 状态码不是连续的，强行用 `iota` 会引入无意义的跳过。分段定义更清晰。

:::

### 第 5 题

定义一个新类型 `UserID int64`，然后尝试把普通 `int64` 赋值给 `UserID`，观察编译错误，并修复。

::: details 第 5 题参考答案

```go
type UserID int64

var id int64 = 12345
// var uid UserID = id  // 编译错误：cannot use id (type int64) as type UserID
var uid UserID = UserID(id)  // 必须显式转换
fmt.Println(uid)
```

**为什么需要转换**：`UserID` 是新类型，与 `int64` 不兼容。这是类型安全的体现，防止把普通整数误用为业务 ID。

**工程意义**：自定义类型可以为 ID、时间戳、金额等业务概念提供类型安全，避免参数传递错误。

:::

### 第 6 题

声明一个 `nil` map 和一个空切片，分别尝试读取和写入，观察哪个会 panic。

::: details 第 6 题参考答案

```go
var m map[string]int
var s []int

// nil map 读取不 panic，返回零值
v := m["key"]
fmt.Println(v)  // 0

// nil map 写入会 panic
// m["key"] = 1  // panic: assignment to entry in nil map

// nil 切片可以 append，不会 panic
s = append(s, 1, 2, 3)
fmt.Println(s)  // [1 2 3]
```

**结论**：`nil` map 读取安全但写入 panic，必须先 `make`。`nil` 切片可以直接 `append`，无需初始化。

**为什么 append 不 panic**：`append` 内部会检查切片是否为 `nil`，如果是就分配新底层数组。

:::

---

## 小结

本章覆盖了 Go 变量与类型系统的核心概念，关键要点：

- **三种声明方式**：`var` 显式、`var` 推导、`:=` 短声明，`:=` 只能在函数内使用。
- **零值保证**：所有变量都有零值，`string` 是空字符串，引用类型是 `nil`。
- **常量与 `iota`**：常量必须编译期确定，`iota` 简化枚举定义，每个 `const` 块重置。
- **整数溢出**：编译期常量溢出报错，运行时溢出回绕，开发者需自行保证范围。
- **浮点精度**：`0.1 + 0.2 != 0.3`，应用 epsilon 比较；`NaN` 不等于任何值，用 `math.IsNaN` 检测。
- **`byte` 与 `rune`**：`byte` 是 `uint8`，`rune` 是 `int32`；`len(s)` 返回字节数，遍历字符用 `for _, r := range s`。
- **显式转换**：Go 不做隐式转换，不同类型不能直接运算，必须显式转换。
- **类型定义 vs 别名**：类型定义创建新类型，类型别名完全等价；`byte` 和 `rune` 是预定义别名。

掌握这些基础后，你将能准确控制数据表示、避免类型陷阱，并写出类型安全的 Go 代码。下一章将学习运算符与格式化输出，探索 `fmt` 包的强大功能与 `go vet` 的格式化检查。
