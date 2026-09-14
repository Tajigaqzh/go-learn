# 第 3 章 · 运算符与格式化输出

在第 2 章中，我们学习了 Go 的变量、常量和基本类型。有了数据之后，就需要对数据进行运算和输出。本章将介绍 Go 的运算符体系以及 `fmt` 包的格式化输出功能，这些是编写任何程序都离不开的基础工具。

本章配套代码在 `internal/chapter/go03_operators_fmt/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 3.1 算术运算符

Go 的算术运算符包括 `+`、`-`、`*`、`/`、`%`，以及自增 `++` 和自减 `--`。

```go
a, b := 10, 3
fmt.Println(a + b)  // 13
fmt.Println(a - b)  // 7
fmt.Println(a * b)  // 30
fmt.Println(a / b)  // 3，整数除法截断小数部分
fmt.Println(a % b)  // 1，取余运算
```

**整数除法与取余的符号规则**：

- 整数除法 `/` 向零截断：`10 / 3 = 3`，`-10 / 3 = -3`
- 取余 `%` 的符号跟随被除数：`-10 % 3 = -1`，`10 % -3 = 1`

```go
fmt.Println(-10 / 3)   // -3
fmt.Println(-10 % 3)   // -1（符号跟随 -10）
fmt.Println(10 % -3)   // 1（符号跟随 10）
fmt.Println(-10 % -3)  // -1
```

**浮点除法**：

只要操作数有一个是浮点类型，除法就按浮点数计算：

```go
fmt.Printf("%.4f\n", 10.0 / 3.0)  // 3.3333
```

**自增与自减**：

Go 的 `++` 和 `--` **只能作为语句**，不能作为表达式：

```go
x := 5
x++      // 合法，x 变成 6
// y := x++  // 编译错误：syntax error: unexpected ++
```

而且只支持后缀形式，不支持 `++x`。

## 3.2 比较运算符

Go 的比较运算符包括 `==`、`!=`、`<`、`<=`、`>`、`>=`，返回 `bool` 类型。

```go
fmt.Println(10 == 20)  // false
fmt.Println(10 != 20)  // true
fmt.Println(10 < 20)   // true
```

**字符串比较**按字典序（逐字节比较 UTF-8 编码）：

```go
fmt.Println("apple" < "banana")  // true
```

**可比较类型**：

- 基本类型（数字、字符串、布尔）都可以用 `==` 和 `!=`
- 指针、接口、channel 也可以比较
- 结构体、数组可以比较，但**切片、map、函数不能直接用 `==` 比较**（只能与 `nil` 比较）

```go
var s []int
fmt.Println(s == nil)  // true，切片只能与 nil 比较

type Point struct{ X, Y int }
fmt.Println(Point{1, 2} == Point{1, 2})  // true，结构体可以比较
```

## 3.3 逻辑运算符

Go 的逻辑运算符包括 `&&`（逻辑与）、`||`（逻辑或）、`!`（逻辑非）。

```go
fmt.Println(true && false)  // false
fmt.Println(true || false)  // true
fmt.Println(!true)          // false
```

**短路求值**：

`&&` 和 `||` 都是短路运算符：

- `a && b`：如果 `a` 为 `false`，则不会计算 `b`
- `a || b`：如果 `a` 为 `true`，则不会计算 `b`

利用短路特性可以避免空指针或除零错误：

```go
x := 0
if x != 0 && 10/x > 1 {  // x != 0 为 false，右侧不执行
    fmt.Println("ok")
}
```

## 3.4 位运算符

Go 支持完整的位运算：

| 运算符 | 含义 | 示例 |
| --- | --- | --- |
| `&` | 按位与 | `12 & 10 = 8` |
| `\|` | 按位或 | `12 \| 10 = 14` |
| `^` | 按位异或 / 按位取反 | `12 ^ 10 = 6`；`^12 = -13` |
| `<<` | 左移 | `12 << 1 = 24` |
| `>>` | 右移 | `12 >> 1 = 6` |
| `&^` | 位清除 | `12 &^ 10 = 4` |

**按位取反 `^`**：

- 二元：`a ^ b` 是异或
- 一元：`^a` 是按位取反（所有位翻转）

```go
a := 12  // 二进制 1100
fmt.Printf("^%d = %d\n", a, ^a)  // ^12 = -13（补码表示）
```

**位清除 `&^`**：

`a &^ b` 表示「清除 `a` 中 `b` 为 1 的位」：

```go
a := 0b1100  // 12
b := 0b1010  // 10
fmt.Printf("%b\n", a &^ b)  // 0100 (4)，清除了 a 中第 1 和第 3 位
```

**常见位操作技巧**：

```go
n := 42
// 设置第 k 位为 1
n |= (1 << 2)
// 清除第 k 位为 0
n &^= (1 << 3)
// 切换第 k 位
n ^= (1 << 1)
// 检查第 k 位是否为 1
if n & (1 << 3) != 0 {
    fmt.Println("第 3 位是 1")
}
```

## 3.5 运算符优先级

Go 的运算符优先级从高到低：

1. `*`、`/`、`%`、`<<`、`>>`、`&`、`&^`
2. `+`、`-`、`|`、`^`
3. `==`、`!=`、`<`、`<=`、`>`、`>=`
4. `&&`
5. `||`

**易错点**：

- **移位运算符优先级高于加减法**：

```go
a := 10 << 1 + 1  // 实际是 (10 << 1) + 1 = 21，而不是 10 << (1 + 1)
b := 1 << 2 + 3   // 实际是 (1 << 2) + 3 = 7
```

- **比较运算符优先级高于逻辑运算符**：

```go
result := 5 < 3 == false  // 先算 5 < 3 得 false，再算 false == false 得 true
```

**建议**：

- 复杂表达式加括号，提高可读性
- 位运算、移位运算和算术混用时，必须加括号

## 3.6 fmt.Printf 基础

`fmt.Printf` 是 Go 中最常用的格式化输出函数，格式字符串中的「占位符」（verb）决定了如何展示数据。

### 常用占位符

| 占位符 | 含义 | 示例 |
| --- | --- | --- |
| `%v` | 默认格式 | `fmt.Printf("%v", 42)` → `42` |
| `%+v` | 结构体带字段名 | `fmt.Printf("%+v", Point{1,2})` → `{X:1 Y:2}` |
| `%#v` | Go 语法表示 | `fmt.Printf("%#v", []int{1,2})` → `[]int{1, 2}` |
| `%T` | 类型 | `fmt.Printf("%T", 42)` → `int` |
| `%d` | 十进制整数 | `fmt.Printf("%d", 42)` → `42` |
| `%b` | 二进制 | `fmt.Printf("%b", 42)` → `101010` |
| `%o` | 八进制 | `fmt.Printf("%o", 42)` → `52` |
| `%x` | 十六进制小写 | `fmt.Printf("%x", 42)` → `2a` |
| `%X` | 十六进制大写 | `fmt.Printf("%X", 42)` → `2A` |
| `%f` | 浮点数 | `fmt.Printf("%f", 3.14)` → `3.140000` |
| `%.2f` | 保留 2 位小数 | `fmt.Printf("%.2f", 3.14159)` → `3.14` |
| `%e` | 科学计数法小写 | `fmt.Printf("%e", 123456.789)` → `1.234568e+05` |
| `%E` | 科学计数法大写 | `fmt.Printf("%E", 123456.789)` → `1.234568E+05` |
| `%s` | 字符串 | `fmt.Printf("%s", "Hi")` → `Hi` |
| `%q` | 带引号的字符串 | `fmt.Printf("%q", "Hi\n")` → `"Hi\n"` |
| `%c` | 字符（Unicode 码点） | `fmt.Printf("%c", 65)` → `A` |
| `%t` | 布尔值 | `fmt.Printf("%t", true)` → `true` |
| `%p` | 指针地址 | `fmt.Printf("%p", &x)` → `0xc00001a0a8` |
| `%%` | 字面 `%` | `fmt.Printf("%%")` → `%` |

```go
n := 42
fmt.Printf("%v\n", n)   // 42
fmt.Printf("%d\n", n)   // 42
fmt.Printf("%b\n", n)   // 101010
fmt.Printf("%x\n", n)   // 2a
fmt.Printf("%#x\n", n)  // 0x2a（带 0x 前缀）
```

## 3.7 fmt.Printf 标志

在 `%` 和动词之间可以插入标志（flags），调整格式：

| 标志 | 含义 | 示例 |
| --- | --- | --- |
| `+` | 总是输出符号（正数也显示 `+`） | `%+d` → `+42` |
| `-` | 左对齐（默认右对齐） | `%-5d` → `42   ` |
| `0` | 零填充（只对数字有效） | `%05d` → `00042` |
| ` ` （空格） | 正数前留空格（负数显示 `-`） | `% d` → ` 42`，`% d` → `-42` |
| `#` | 备用格式：`%#x` 加 `0x`，`%#o` 加 `0`，`%#v` 用 Go 语法 | `%#x` → `0x2a` |

```go
fmt.Printf("%+d\n", 42)    // +42
fmt.Printf("%5d\n", 42)    // "   42"（右对齐，宽度 5）
fmt.Printf("%-5d\n", 42)   // "42   "（左对齐）
fmt.Printf("%05d\n", 42)   // 00042
fmt.Printf("%#x\n", 42)    // 0x2a
```

**组合标志**：

```go
fmt.Printf("%+#08x\n", 255)  // +0x0000ff
// + 显示符号，# 加 0x 前缀，08 零填充到宽度 8
```

## 3.8 Printf 宽度和精度

**宽度**：在动词前指定数字，表示最小输出宽度（不足时用空格填充）：

```go
fmt.Printf("%5d\n", 42)    // "   42"（右对齐）
fmt.Printf("%-5d\n", 42)   // "42   "（左对齐）
fmt.Printf("%010d\n", 42)  // 0000000042（零填充）
```

**精度**：用 `.数字` 指定，含义取决于动词：

- 浮点数：小数位数
- 字符串：最大输出字符数

```go
fmt.Printf("%.2f\n", 3.14159)   // 3.14
fmt.Printf("%.5s\n", "Hello")   // Hello（最多 5 个字符）
```

**动态宽度和精度**：

用 `*` 占位，从参数列表获取宽度或精度：

```go
width, precision := 10, 2
fmt.Printf("%*.*f\n", width, precision, 3.14159)
// 输出：      3.14（宽度 10，精度 2）
```

## 3.9 Stringer 接口

如果类型实现了 `fmt.Stringer` 接口（即定义了 `String() string` 方法），`%v` 和 `%s` 会自动调用它：

```go
type Point struct{ X, Y int }

func (p Point) String() string {
    return fmt.Sprintf("Point(%d, %d)", p.X, p.Y)
}

p := Point{3, 4}
fmt.Printf("%v\n", p)   // Point(3, 4)（调用 String 方法）
fmt.Printf("%#v\n", p)  // go03_operators_fmt.Point{X:3, Y:4}（Go 语法，不调用 String）
```

**注意**：

- `%v` 和 `%s` 会调用 `String()` 方法
- `%#v` 不调用，直接输出 Go 语法表示
- `%T` 输出类型名，也不调用

## 3.10 Sprintf 和 Fprintf

除了 `Printf`，`fmt` 包还提供：

- `fmt.Sprintf(format, args...)` → 返回格式化后的字符串
- `fmt.Fprintf(w io.Writer, format, args...)` → 写入 `io.Writer`

```go
s := fmt.Sprintf("圆周率约等于 %.2f", 3.14159)
fmt.Println(s)  // 圆周率约等于 3.14

var buf bytes.Buffer
fmt.Fprintf(&buf, "%d + %d = %d", 1, 2, 3)
fmt.Println(buf.String())  // 1 + 2 = 3
```

**其他常用函数**：

- `fmt.Print(args...)`：无格式，直接拼接输出
- `fmt.Println(args...)`：拼接后加换行
- `fmt.Errorf(format, args...)`：返回格式化的 `error`

```go
fmt.Print("Hello", " ", "World")  // Hello World
fmt.Println("Hello", "World")     // Hello World\n
err := fmt.Errorf("打开文件失败: %s", "file.txt")
```

---

## 真实报错案例

### 案例 1：++ 当表达式用

```go
x := 5
y := x++  // 编译错误
```

**报错**：

```
syntax error: unexpected ++, expecting comma or }
```

**原因**：Go 的 `++` 只能作为语句，不能作为表达式。

**修复**：

```go
x := 5
x++
y := x  // y = 6
```

### 案例 2：切片直接比较

```go
a := []int{1, 2}
b := []int{1, 2}
fmt.Println(a == b)  // 编译错误
```

**报错**：

```
invalid operation: a == b (slice can only be compared to nil)
```

**原因**：切片不能用 `==` 直接比较，只能与 `nil` 比较。

**修复**：

使用 `reflect.DeepEqual` 或手动逐元素比较：

```go
import "reflect"
fmt.Println(reflect.DeepEqual(a, b))  // true
```

### 案例 3：Printf 参数类型不匹配

```go
fmt.Printf("%d", "hello")  // 运行时输出错误
```

**输出**：

```
%!d(string=hello)
```

**原因**：`%d` 期望整数，但传入字符串，`fmt` 会输出错误提示。

**修复**：

- 检查占位符与参数类型是否匹配
- 使用 `go vet` 可以在编译时检查出这类错误：

```bash
$ go vet your_file.go
Printf format %d has arg "hello" of wrong type string
```

### 案例 4：整数溢出

```go
var a int8 = 127
a++
fmt.Println(a)  // -128
```

**现象**：`int8` 的最大值是 127，加 1 后溢出变成 -128（补码表示）。

**原因**：Go 的整数溢出会回绕，不会报错或 panic。

**避免方法**：

- 使用更大的类型（`int`、`int64`）
- 关键场景下手动检查边界
- 使用 `math` 包的常量检查：

```go
import "math"
if a == math.MaxInt8 {
    fmt.Println("即将溢出")
}
```

### 案例 5：运算符优先级错误

```go
mask := 1 << 2 + 3  // 程序员以为是 1 << (2 + 3) = 32
fmt.Println(mask)   // 实际输出 7
```

**原因**：Go 的移位运算符优先级**高于加减法**，所以实际是 `(1 << 2) + 3 = 4 + 3 = 7`。

**修复**：加括号明确意图：

```go
mask := 1 << (2 + 3)  // 32
```

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `y := x++` 报错 | `++` 只能是语句，不能是表达式 | 分两行：`x++; y := x` |
| 切片 `a == b` 报错 | 切片只能与 `nil` 比较 | 用 `reflect.DeepEqual` 或手动遍历 |
| `fmt.Printf("%d", "str")` 输出 `%!d(string=str)` | 类型不匹配 | 检查占位符，运行 `go vet` 提前发现 |
| `int8` 溢出后变成负数 | Go 整数溢出会回绕 | 用更大类型或手动检查边界 |
| `1 << 2 + 3` 结果是 7 而不是 32 | 移位优先级高于加法 | 加括号：`1 << (2 + 3)` |
| `%v` 输出指针地址而不是值 | 传入的是指针 | 传值或用 `%+v` / `%#v` 查看结构 |
| `%s` 输出二进制乱码 | 字节切片被当字符串 | 用 `%q` 或 `%x` 输出 |

---

## 练习

### 第 1 题

判断下面代码的输出：

```go
a := 15
b := 4
fmt.Println(a / b)
fmt.Println(a % b)
fmt.Println(float64(a) / float64(b))
```

::: details 第 1 题参考答案

输出：

```
3
3
3.75
```

**解析**：

- `a / b` 是整数除法，结果 `3`（向零截断）
- `a % b` 取余，`15 % 4 = 3`
- 转成 `float64` 后是浮点除法，结果 `3.75`

**要点**：整数除法会丢失小数部分，如果需要精确结果，先转浮点再除。

:::

### 第 2 题

下面代码能否编译通过？如果不能，如何修复？

```go
x := 10
if x > 5 && expensive() {
    fmt.Println("ok")
}
```

假设 `expensive()` 是一个耗时函数，返回 `bool`。

::: details 第 2 题参考答案

**能编译通过，但需要注意短路求值**。

如果 `x > 5` 为 `true`，则 `expensive()` 会被调用；如果为 `false`，则 `expensive()` 不会执行。

如果你希望总是调用 `expensive()`，应该先单独执行：

```go
result := expensive()
if x > 5 && result {
    fmt.Println("ok")
}
```

**要点**：利用短路特性可以提高性能，但要注意副作用函数的执行时机。

:::

### 第 3 题

用位运算实现：判断一个整数 `n` 是否是 2 的幂（即 `n = 2^k`，k ≥ 0）。

::: details 第 3 题参考答案

```go
func isPowerOfTwo(n int) bool {
    return n > 0 && (n & (n - 1)) == 0
}
```

**原理**：

- 2 的幂的二进制只有一个 `1`：`1`、`10`、`100`、`1000`...
- `n - 1` 会把最右边的 `1` 变成 `0`，后面全变成 `1`
- `n & (n - 1)` 会把唯一的 `1` 清除，结果为 `0`

例如 `n = 8`（二进制 `1000`）：

- `n - 1 = 7`（`0111`）
- `8 & 7 = 0`

如果 `n` 不是 2 的幂（如 `n = 6`，`110`）：

- `n - 1 = 5`（`101`）
- `6 & 5 = 4`（非 0）

:::

### 第 4 题

写出下面代码的输出：

```go
fmt.Printf("|%5d|\n", 42)
fmt.Printf("|%-5d|\n", 42)
fmt.Printf("|%05d|\n", 42)
```

::: details 第 4 题参考答案

输出：

```
|   42|
|42   |
|00042|
```

**解析**：

- `%5d`：宽度 5，右对齐，空格填充
- `%-5d`：宽度 5，**左对齐**，空格填充
- `%05d`：宽度 5，右对齐，**零填充**

**要点**：默认右对齐，`-` 标志改为左对齐，`0` 标志用零填充（只对数字有效）。

:::

### 第 5 题

定义一个 `Temperature` 类型，实现 `Stringer` 接口，输出时自动加上 `°C` 后缀。

```go
type Temperature float64

// TODO: 实现 String() 方法

func main() {
    t := Temperature(36.5)
    fmt.Println(t)  // 期望输出：36.5°C
}
```

::: details 第 5 题参考答案

```go
func (t Temperature) String() string {
    return fmt.Sprintf("%.1f°C", t)
}
```

**完整示例**：

```go
package main

import "fmt"

type Temperature float64

func (t Temperature) String() string {
    return fmt.Sprintf("%.1f°C", t)
}

func main() {
    t := Temperature(36.5)
    fmt.Println(t)          // 36.5°C
    fmt.Printf("%v\n", t)   // 36.5°C
    fmt.Printf("%#v\n", t)  // main.Temperature(36.5)（Go 语法，不调用 String）
}
```

**要点**：实现 `String()` 方法后，`%v` 和 `%s` 会自动调用；`%#v` 不调用，输出 Go 语法表示。

:::

### 第 6 题

使用 `fmt.Errorf` 和 `%w` 包装错误，保留原始错误信息。

```go
package main

import (
    "errors"
    "fmt"
)

func readFile() error {
    return errors.New("file not found")
}

func processFile() error {
    err := readFile()
    if err != nil {
        // TODO: 包装错误并返回
    }
    return nil
}

func main() {
    err := processFile()
    if err != nil {
        fmt.Println(err)
        // TODO: 用 errors.Is 或 errors.As 检查原始错误
    }
}
```

::: details 第 6 题参考答案

```go
func processFile() error {
    err := readFile()
    if err != nil {
        return fmt.Errorf("处理文件失败: %w", err)
    }
    return nil
}

func main() {
    err := processFile()
    if err != nil {
        fmt.Println(err)  // 处理文件失败: file not found

        // 检查是否包含特定错误
        fileNotFoundErr := errors.New("file not found")
        if errors.Is(err, fileNotFoundErr) {
            fmt.Println("确实是文件未找到错误")
        }

        // 或者用 Unwrap 获取原始错误
        fmt.Println("原始错误:", errors.Unwrap(err))
    }
}
```

**要点**：

- `%w` 会包装原始错误，可以通过 `errors.Is` 或 `errors.Unwrap` 访问
- `%v` 只是拼接错误信息字符串，丢失了原始错误对象

:::

---

## 小结

- Go 的算术运算符支持 `+`、`-`、`*`、`/`、`%`，整数除法向零截断，取余符号跟随被除数
- `++` 和 `--` 只能作为语句，不能作为表达式；不支持前缀形式
- 比较运算符适用于基本类型、结构体、数组；**切片、map、函数不能直接用 `==` 比较**
- 逻辑运算符 `&&` 和 `||` 支持短路求值，可以避免无效计算和空指针错误
- 位运算符包括 `&`、`|`、`^`、`<<`、`>>`、`&^`，常用于标志位、权限控制、性能优化
- 运算符优先级：移位 > 加减 > 比较 > 逻辑；复杂表达式建议加括号
- `fmt.Printf` 的占位符包括 `%v`、`%d`、`%f`、`%s`、`%t`、`%p`、`%T` 等，配合标志、宽度、精度灵活控制输出格式
- 实现 `Stringer` 接口（`String() string`）后，`%v` 和 `%s` 会自动调用，方便自定义类型输出
- `fmt.Sprintf` 返回格式化字符串，`fmt.Fprintf` 写入 `io.Writer`，`fmt.Errorf` 返回格式化错误
- 使用 `go vet` 可以在编译时检查 `Printf` 类型不匹配问题

下一章将介绍 Go 的控制流：`if`、`for`、`switch`、`defer`，以及错误处理的基本模式。

<!-- CONTINUE_MARKER_03 -->
