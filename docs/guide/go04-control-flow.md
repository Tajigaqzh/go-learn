# 第 4 章 · 控制流

前面三章我们学习了变量、类型、运算符和格式化输出——这些是程序的「原料」。但一个真正有用的程序不仅需要数据，还需要**决策**和**循环**：根据条件选择不同的执行路径，或者重复执行某段代码直到满足条件。这就是控制流（control flow）的作用。

Go 的控制流设计非常简洁：只有 `if`、`for`、`switch` 和 `goto` 四种语句，没有 `while`、`do-while`、三元运算符或传统的 `foreach`。但正是这种简洁让 Go 的代码更易读、更统一：你不需要在多种写法之间纠结，几乎所有的循环都能用 `for` 表达，所有的条件判断都能用 `if` 或 `switch` 解决。

本章配套代码在 `internal/chapter/go04_control_flow/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 4.1 if 的基本形式

Go 的 `if` 语句不需要括号包裹条件表达式，但花括号 `{}` 是必须的，即使只有一行代码：

```go
x := 10
if x > 5 {
    fmt.Println("x > 5")
}
```

**输出：**
```
x > 5
```

你可以在同一个 `if` 链中使用 `else if` 和 `else`：

```go
if x > 20 {
    fmt.Println("x > 20")
} else if x > 5 {
    fmt.Println("x <= 20")
} else {
    fmt.Println("x <= 5")
}
```

**输出：**
```
x <= 20
```

Go 禁止在条件表达式外加括号（`if (x > 5)` 会编译错误），也禁止省略花括号。这个强制规范避免了很多常见错误，比如「悬空 else」（dangling else）问题，也让所有 Go 代码的 `if` 看起来都一致。

## 4.2 if 的初始化语句

`if` 可以在条件表达式前执行一条初始化语句，用分号 `;` 隔开。这个初始化语句中声明的变量**只在 `if`-`else` 块内可见**：

```go
if y := x + 5; y > 10 {
    fmt.Printf("y=%d > 10\n", y)
}
// fmt.Println(y) // 编译错误：y 在这里不可见
```

**输出：**
```
y=15 > 10
```

这种写法常用于缩小变量作用域，避免污染外层命名空间。典型场景是调用函数并立即检查错误：

```go
if err := doSomething(); err != nil {
    fmt.Println("失败")
} else {
    fmt.Println("成功")
}
```

**输出：**
```
成功
```

这里 `err` 只在 `if`-`else` 块内有效，函数调用和错误检查紧密关联，代码更紧凑。

## 4.3 for 的经典形式

Go 只有 `for` 一种循环语句，但它有多种形态。最经典的是三段式：

```go
for i := 0; i < 5; i++ {
    fmt.Print(i, " ")
}
```

**输出：**
```
0 1 2 3 4
```

三段式的结构是 `for 初始化; 条件; 后置语句 {}`，和 C/Java 的 `for` 一致。初始化语句只执行一次，条件在每次循环前检查，后置语句在每次循环体执行后运行。

你可以在初始化部分声明多个变量，或者在后置部分执行多个语句（用逗号分隔），但通常不推荐——保持简单更易读：

```go
sum := 0
for i := 1; i <= 10; i++ {
    sum += i
}
fmt.Printf("1+2+...+10 = %d\n", sum)
```

**输出：**
```
1+2+...+10 = 55
```

多变量初始化：

```go
for i, j := 0, 10; i < 5; i, j = i+1, j-1 {
    fmt.Printf("i=%d, j=%d\n", i, j)
}
```

**输出：**
```
i=0, j=10
i=1, j=9
i=2, j=8
i=3, j=7
i=4, j=6
```

## 4.4 for 的条件形式（类似 while）

省略初始化和后置语句，只保留条件，就得到了类似其他语言 `while` 的写法：

```go
n := 1
for n < 100 {
    n *= 2
}
fmt.Printf("第一个 >= 100 的 2 的幂：%d\n", n)
```

**输出：**
```
第一个 >= 100 的 2 的幂：128
```

这种形式常用于「做到满足条件为止」的场景，比如读取输入、等待信号等。

## 4.5 for 的无限循环

省略所有三部分，得到无限循环：

```go
count := 0
for {
    count++
    fmt.Println("循环", count)
    if count >= 3 {
        break
    }
}
fmt.Println("退出无限循环")
```

**输出：**
```
循环 1
循环 2
循环 3
退出无限循环
```

无限循环通常配合 `break` 或 `return` 在循环体内控制退出条件，常见于服务器主循环、事件监听等场景。

## 4.6 range 遍历切片

`range` 是 Go 最常用的遍历方式，可以遍历切片、数组、映射、字符串和 channel。对于切片和数组，`range` 返回两个值：**索引**和**元素值**：

```go
nums := []int{10, 20, 30, 40}
for i, v := range nums {
    fmt.Printf("nums[%d] = %d\n", i, v)
}
```

**输出：**
```
nums[0] = 10
nums[1] = 20
nums[2] = 30
nums[3] = 40
```

如果只要索引，可以省略第二个变量：

```go
for i := range nums {
    fmt.Print(i, " ")
}
```

**输出：**
```
索引：0 1 2 3
```

如果只要值，用空白标识符 `_` 忽略索引：

```go
for _, v := range nums {
    fmt.Print(v, " ")
}
```

**输出：**
```
值：10 20 30 40
```

**重要：Go 1.22 的循环变量语义变化**

在 Go 1.22 之前，`range` 循环变量（如上面的 `i` 和 `v`）在整个循环中**共享同一个变量**，每次迭代只是修改它的值。这导致了一个经典的坑：

```go
// Go 1.21 及之前的行为
var funcs []func()
for i := 0; i < 3; i++ {
    funcs = append(funcs, func() {
        fmt.Println("捕获的 i=", i) // 所有闭包捕获的是同一个 i
    })
}
for _, f := range funcs {
    f() // 输出：3 3 3（循环结束后 i=3）
}
```

从 **Go 1.22** 开始，每次迭代都会创建新的循环变量，闭包捕获的是当次迭代的副本，行为更符合直觉：

```go
// Go 1.22 及之后的行为
for i := 0; i < 3; i++ {
    funcs = append(funcs, func() {
        fmt.Println("捕获的 i=", i) // 每次迭代的 i 是独立的
    })
}
for _, f := range funcs {
    f() // 输出：0 1 2
}
```

**输出（Go 1.22+）：**
```
捕获的 i=0
捕获的 i=1
捕获的 i=2
```

这个变化只影响闭包捕获的行为，不影响循环本身的执行逻辑。如果你的代码需要在 Go 1.21 及之前版本运行，仍然需要显式复制循环变量：

```go
for i := 0; i < 3; i++ {
    i := i // 显式复制（Go 1.21 及之前）
    funcs = append(funcs, func() {
        fmt.Println(i)
    })
}
```

## 4.7 range 遍历 map

遍历 map 时，`range` 返回**键**和**值**。注意 map 的遍历顺序是**随机的**，每次运行可能不同：

```go
ages := map[string]int{"Alice": 25, "Bob": 30, "Carol": 28}
for name, age := range ages {
    fmt.Printf("%s: %d\n", name, age)
}
```

**输出（顺序每次可能不同）：**
```
Carol: 28
Alice: 25
Bob: 30
```

如果只要键：

```go
for name := range ages {
    fmt.Print(name, " ")
}
```

**输出（顺序随机）：**
```
键：Alice Bob Carol
```

如果需要有序遍历，先把键提取到切片，排序后再访问 map：

```go
keys := make([]string, 0, len(ages))
for k := range ages {
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    fmt.Printf("%s: %d\n", k, ages[k])
}
```

## 4.8 range 遍历字符串

遍历字符串时，`range` 返回**字节索引**和**rune 值**（Unicode 码点）：

```go
s := "Hello,世界"
for i, r := range s {
    fmt.Printf("[%d] %c (U+%04X)\n", i, r, r)
}
```

**输出：**
```
[0] H (U+0048)
[1] e (U+0065)
[2] l (U+006C)
[3] l (U+006C)
[4] o (U+006F)
[5] , (U+002C)
[6] 世 (U+4E16)
[9] 界 (U+754C)
```

注意索引不是连续的：`"世"` 占 3 个字节（UTF-8 编码），所以下一个字符 `"界"` 的索引是 9 而不是 7。

`len(s)` 返回的是字节数，不是字符数：

```go
fmt.Printf("len(s)=%d（字节数），rune 个数=%d\n", len(s), utf8.RuneCountInString(s))
```

**输出：**
```
len(s)=12（字节数），rune 个数=8
```

如果需要统计 rune 个数，用 `utf8.RuneCountInString` 或者遍历时计数。

## 4.9 break、continue 和标签

`break` 跳出当前循环，`continue` 跳过本次迭代：

```go
for i := 0; i < 5; i++ {
    fmt.Print(i, " ")
}
// 输出：0 1 2 3 4

for i := 0; i < 10; i++ {
    if i%2 == 0 {
        continue // 跳过偶数
    }
    fmt.Print(i, " ")
}
// 输出：1 3 5 7 9
```

**标签（label）** 可以用于嵌套循环的跳转。标签必须紧贴在 `for`、`switch` 或 `select` 前面：

```go
outer:
for i := 0; i < 3; i++ {
    for j := 0; j < 3; j++ {
        fmt.Printf("(%d,%d) ", i, j)
        if i == 2 && j == 2 {
            break outer // 跳出外层循环
        }
    }
    fmt.Println()
}
fmt.Println("找到 i=2, j=2，跳出外层")
```

**输出：**
```
(0,0) (0,1) (0,2) 
(1,0) (1,1) (1,2) 
(2,0) (2,1) 找到 i=2, j=2，跳出外层
```

没有标签时，`break` 只能跳出最内层循环。标签让你可以直接跳到外层，避免使用额外的标志变量。

**注意：** 标签只能用于 `for`、`switch`、`select`，不能用于 `if`。

## 4.10 switch 的基本形式

`switch` 用于多路分支选择，比一长串 `if-else if` 更清晰：

```go
day := 3
switch day {
case 1:
    fmt.Println("星期一")
case 2:
    fmt.Println("星期二")
case 3:
    fmt.Println("星期三")
default:
    fmt.Println("其他")
}
```

**输出：**
```
星期三
```

Go 的 `switch` 有几个重要特性：

1. **自动 break**：匹配一个 `case` 后自动退出，不会穿透到下一个 `case`（和 C/Java 不同）。
2. **case 可以是表达式**：不限于常量，可以是变量、函数调用或任何返回值。
3. **case 可以有多个值**：用逗号分隔，匹配任意一个即可。

```go
x := 10
switch {
case x < 0:
    fmt.Println("负数")
case x == 0:
    fmt.Println("零")
case x > 0:
    fmt.Println("正数")
}
```

**输出：**
```
正数
```

## 4.11 无表达式 switch

省略 `switch` 后的表达式，等价于 `switch true`，此时每个 `case` 是一个布尔表达式：

```go
x := 42
switch {
case x < 10:
    fmt.Println("x 是一位数")
case x < 100:
    fmt.Println("x 是两位数")
case x < 1000:
    fmt.Println("x 是三位数")
default:
    fmt.Println("x 很大")
}
```

**输出：**
```
x 是两位数
```

这种写法比一长串 `if-else if` 更清晰，尤其当条件表达式较复杂时。

## 4.12 fallthrough

如果你确实需要穿透到下一个 `case`，可以显式使用 `fallthrough`。注意 `fallthrough` 必须是 `case` 块的最后一条语句，且**无条件穿透**到下一个 `case`，不检查下一个 `case` 的条件：

```go
n := 1
switch n {
case 1:
    fmt.Println("case 1")
    fallthrough
case 2:
    fmt.Println("case 2（fallthrough 穿透）")
    fallthrough
case 3:
    fmt.Println("case 3（fallthrough 穿透）")
default:
    fmt.Println("default")
}
```

**输出：**
```
case 1
case 2（fallthrough 穿透）
case 3（fallthrough 穿透）
```

`fallthrough` 在实际代码中很少使用，因为它违背了 Go 设计的初衷（避免意外穿透）。大多数情况下，把多个条件合并到一个 `case` 更清晰。

## 4.13 goto（慎用）

Go 保留了 `goto`，但使用场景非常有限。`goto` 可以跳转到同一函数内的标签，但不能跳过变量声明，也不能跳入内层作用域：

```go
i := 0
loop:
    fmt.Print("i=", i, " ")
    i++
    if i < 3 {
        goto loop
    }
fmt.Println("\ngoto 示例结束")
```

**输出：**
```
i=0 i=1 i=2 
goto 示例结束
```

**为什么要慎用 goto？**

- 容易制造「意大利面条代码」（spaghetti code），降低可读性。
- Go 的 `for`、`break`、`continue` 和标签已经能覆盖绝大多数场景。
- `goto` 的合理使用场景：错误处理中的资源清理（但 `defer` 通常更好）、跳出多层嵌套（但标签的 `break` 更清晰）。

除非你确定 `goto` 是最清晰的写法，否则避免使用。

---

## 5 个真实报错怎么读

### 报错 1：if 条件缺少花括号

```go
if x > 5
    fmt.Println("x > 5")
```

**编译器输出：**
```
./demo.go:10:5: syntax error: unexpected newline, expecting { after if clause
```

**原因：** Go 强制要求花括号 `{}`，即使只有一行代码。

**修复：**
```go
if x > 5 {
    fmt.Println("x > 5")
}
```

### 报错 2：if 条件加了括号

```go
if (x > 5) {
    fmt.Println("x > 5")
}
```

**编译器输出：**
```
./demo.go:10:8: syntax error: unexpected (, expecting {
```

**原因：** Go 禁止在条件表达式外加括号，除非括号是表达式本身的一部分（如 `(a || b) && c`）。

**修复：**
```go
if x > 5 {
    fmt.Println("x > 5")
}
```

### 报错 3：for 的初始化变量遮蔽外层

```go
i := 100
for i := 0; i < 3; i++ {
    fmt.Println(i) // 内层的 i
}
fmt.Println(i) // 外层的 i 仍然是 100
```

这不是编译错误，但容易混淆。`for` 内的 `i` 遮蔽了外层的 `i`，两者是不同的变量。如果你想修改外层的 `i`，应该：

```go
i := 100
for i = 0; i < 3; i++ { // 不用 :=，直接赋值
    fmt.Println(i)
}
fmt.Println(i) // 现在是 3
```

### 报错 4：range 遍历 nil map 会 panic

```go
var m map[string]int // nil map
for k, v := range m { // 不会 panic
    fmt.Println(k, v)
}
m["key"] = 1 // panic: assignment to entry in nil map
```

**运行时输出：**
```
panic: assignment to entry in nil map
```

**原因：** `range` 遍历 nil map 是安全的（不执行任何迭代），但**写入** nil map 会 panic。

**修复：** 初始化 map：
```go
m := make(map[string]int)
m["key"] = 1
```

### 报错 5：goto 跳过变量声明

```go
goto skip
x := 10
skip:
fmt.Println(x)
```

**编译器输出：**
```
./demo.go:10:2: goto skip jumps over declaration of x at ./demo.go:11:2
```

**原因：** `goto` 不能跳过变量声明，因为这会导致未初始化的变量被使用。

**修复：** 在 `goto` 之前声明变量：
```go
var x int
goto skip
x = 10
skip:
fmt.Println(x) // 输出 0（零值）
```

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `if` 后少花括号编译错误 | Go 强制要求 `{}`，即使只有一行 | 加花括号：`if cond { ... }` |
| `if (cond)` 编译错误 | Go 禁止条件表达式外加括号 | 去掉括号：`if cond { ... }` |
| `range` 闭包捕获的变量都是最后一个值 | Go 1.21 及之前循环变量共享 | Go 1.22+ 自动修复；1.21 需显式复制：`v := v` |
| `range` 遍历 map 顺序每次不同 | map 遍历顺序随机 | 提取键到切片，排序后遍历 |
| `range` 字符串索引不连续 | 索引是字节偏移，多字节字符跳过中间字节 | 理解 UTF-8 编码，或转为 `[]rune` |
| 写 nil map 会 panic | nil map 不能写，只能读和遍历 | 用 `make(map[K]V)` 初始化 |
| `break` 只跳出最内层循环 | 默认行为 | 用标签 `break outer` 跳出外层 |
| `fallthrough` 无条件穿透 | 不检查下一个 `case` 条件 | 合并 `case` 或重新设计逻辑 |
| `goto` 跳过变量声明编译错误 | 跳过声明会导致未初始化变量 | 在 `goto` 前声明变量 |

---

## 练习

### 第 1 题

写一个函数 `fizzBuzz(n int)`,打印 1 到 n 的数字，但：
- 能被 3 整除的数字打印 "Fizz"
- 能被 5 整除的数字打印 "Buzz"
- 能被 15 整除的数字打印 "FizzBuzz"
- 其他打印数字本身

示例：`fizzBuzz(15)` 应打印：
```
1 2 Fizz 4 Buzz Fizz 7 8 Fizz Buzz 11 Fizz 13 14 FizzBuzz
```

::: details 第 1 题参考答案
```go
func fizzBuzz(n int) {
    for i := 1; i <= n; i++ {
        switch {
        case i%15 == 0:
            fmt.Print("FizzBuzz ")
        case i%3 == 0:
            fmt.Print("Fizz ")
        case i%5 == 0:
            fmt.Print("Buzz ")
        default:
            fmt.Print(i, " ")
        }
    }
    fmt.Println()
}
```

**为什么这样写更好：**
- 用无表达式 `switch` 清晰表达优先级：先检查 15，再检查 3 和 5。
- 如果用 `if-else if` 链也可以，但 `switch` 的意图更明确。
- 注意 `case i%15 == 0` 必须放在最前面，否则会被 `case i%3 == 0` 或 `case i%5 == 0` 提前匹配。
:::

### 第 2 题

用 `for` 和 `range` 两种方式反转切片 `[]int{1, 2, 3, 4, 5}`，打印结果。

::: details 第 2 题参考答案
**方法一：双指针（经典 for）**
```go
nums := []int{1, 2, 3, 4, 5}
for i, j := 0, len(nums)-1; i < j; i, j = i+1, j-1 {
    nums[i], nums[j] = nums[j], nums[i]
}
fmt.Println(nums) // [5 4 3 2 1]
```

**方法二：倒序遍历（range + 索引）**
```go
nums := []int{1, 2, 3, 4, 5}
reversed := make([]int, len(nums))
for i, v := range nums {
    reversed[len(nums)-1-i] = v
}
fmt.Println(reversed) // [5 4 3 2 1]
```

**为什么方法一更好：** 原地反转，不分配新切片，空间复杂度 O(1)。方法二更直观但需要额外空间。
:::

### 第 3 题

写一个函数 `findPrime(n int) int`，返回第 n 个质数（质数从 2 开始，第 1 个质数是 2，第 2 个是 3）。

::: details 第 3 题参考答案
```go
func findPrime(n int) int {
    count := 0
    num := 2
    for {
        if isPrime(num) {
            count++
            if count == n {
                return num
            }
        }
        num++
    }
}

func isPrime(n int) bool {
    if n < 2 {
        return false
    }
    for i := 2; i*i <= n; i++ {
        if n%i == 0 {
            return false
        }
    }
    return true
}
```

**示例：**
```go
fmt.Println(findPrime(1))  // 2
fmt.Println(findPrime(10)) // 29
```

**为什么这样写：**
- 用无限循环 `for {}` 配合 `return` 更清晰，避免复杂的条件判断。
- `isPrime` 只检查到 `sqrt(n)`，优化了性能。
:::

### 第 4 题

用 `range` 遍历字符串 `"Go语言"`，统计 ASCII 字符和非 ASCII 字符的数量。

::: details 第 4 题参考答案
```go
s := "Go语言"
ascii, nonASCII := 0, 0
for _, r := range s {
    if r < 128 {
        ascii++
    } else {
        nonASCII++
    }
}
fmt.Printf("ASCII: %d, 非ASCII: %d\n", ascii, nonASCII)
// 输出：ASCII: 2, 非ASCII: 2
```

**为什么这样写：** `range` 遍历字符串返回 rune，直接比较 Unicode 码点即可。ASCII 范围是 0-127。
:::

### 第 5 题

写一个函数 `sumEvenInMatrix(matrix [][]int) int`，用标签和 `break` 提前退出嵌套循环，计算二维切片中所有偶数的和，但遇到负数就停止。

示例：
```go
matrix := [][]int{
    {1, 2, 3},
    {4, 5, 6},
    {-1, 8, 9}, // 遇到 -1 停止
}
// 应返回 2 + 4 + 6 = 12
```

::: details 第 5 题参考答案
```go
func sumEvenInMatrix(matrix [][]int) int {
    sum := 0
outer:
    for _, row := range matrix {
        for _, val := range row {
            if val < 0 {
                break outer // 遇到负数跳出外层循环
            }
            if val%2 == 0 {
                sum += val
            }
        }
    }
    return sum
}
```

**为什么用标签：** 没有标签时，`break` 只能跳出内层循环，外层会继续。标签让你可以直接退出整个嵌套结构。
:::

### 第 6 题

Go 1.22 的循环变量语义变化对下面代码有什么影响？解释并修复（如果需要）。

```go
var funcs []func()
for i := 0; i < 3; i++ {
    funcs = append(funcs, func() {
        fmt.Println(i)
    })
}
for _, f := range funcs {
    f()
}
```

::: details 第 6 题参考答案
**Go 1.21 及之前：** 输出 `3 3 3`，因为所有闭包捕获的是同一个 `i`，循环结束后 `i=3`。

**Go 1.22 及之后：** 输出 `0 1 2`，因为每次迭代创建新的 `i`，闭包捕获的是当次迭代的副本。

**如果需要兼容 Go 1.21，修复方法：**
```go
for i := 0; i < 3; i++ {
    i := i // 显式复制
    funcs = append(funcs, func() {
        fmt.Println(i)
    })
}
```

**为什么这个变化重要：** 这是 Go 语言设计中的一个「历史包袱」，Go 1.22 的修复让行为更符合直觉，避免了大量新手错误。
:::

---

## 小结

- Go 只有 `if`、`for`、`switch`、`goto` 四种控制流语句，设计简洁统一。
- `if` 和 `for` 的初始化语句可以缩小变量作用域，让代码更紧凑。
- `for` 是唯一的循环语句，有四种形态：经典三段式、条件式、无限循环、`range` 遍历。
- `range` 可以遍历切片、数组、map、字符串和 channel，返回索引/键和值。
- Go 1.22 改变了循环变量语义，每次迭代创建新变量，闭包捕获行为更符合直觉。
- `switch` 自动 `break`，无需显式写 `break`；用 `fallthrough` 可以穿透到下一个 `case`。
- 标签配合 `break` 可以跳出嵌套循环，避免使用额外的标志变量。
- `goto` 在 Go 中很少使用，除非确定是最清晰的写法，否则避免。

下一章我们将学习**函数与闭包**：如何定义函数、多返回值、变参、闭包捕获、`defer` 的执行顺序，以及为什么 Go 没有函数重载。
