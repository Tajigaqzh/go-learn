// Package go05_functions 演示 Go 语言的函数、闭包与 defer。
//
// 涵盖主题：
//   - 函数签名、多返回值、命名返回值
//   - 变参函数
//   - 函数是一等值（高阶函数）
//   - 闭包与变量捕获
//   - defer 的执行顺序与参数求值时机
//   - defer 配合命名返回值的坑
//   - 递归
//   - init 函数与包初始化
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go05_functions/
package go05_functions

import (
	"fmt"
	"strings"
)

// Demo 是第 5 章的入口函数，按小节顺序演示函数与闭包的核心概念。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s ==========\n", Chapter, "functions")
	fmt.Println("========================================")

	section1_FunctionSignature()
	section2_NamedReturn()
	section3_Variadic()
	section4_FirstClass()
	section5_Closure()
	section6_DeferOrder()
	section7_DeferNamedReturn()
	section8_Recursion()
	section9_InitReview()

	fmt.Println("========================================")
	fmt.Println("========== 函数与闭包演示结束 ==========")
	fmt.Println()
}

// --- 5.1 函数签名与多返回值 ---
func section1_FunctionSignature() {
	fmt.Println("\n--- 5.1 函数签名与多返回值 ---")

	// Go 函数可以返回多个值，这是最常用的错误处理模式
	q, r := Divide(17, 5)
	fmt.Printf("Divide(17, 5) = quotient=%d, remainder=%d\n", q, r)

	// 只关心其中一个返回值时用空白标识符
	q2, _ := Divide(20, 4)
	fmt.Printf("Divide(20, 4) 的商 = %d\n", q2)

	// 多返回值也常用于"成功/失败"模式
	result, ok := SafeDivide(10, 0)
	fmt.Printf("SafeDivide(10, 0): result=%d, ok=%t\n", result, ok)

	result2, ok2 := SafeDivide(10, 2)
	fmt.Printf("SafeDivide(10, 2): result=%d, ok=%t\n", result2, ok2)
}

// Divide 返回商和余数。
func Divide(a, b int) (int, int) {
	return a / b, a % b
}

// SafeDivide 安全除法，除数为 0 时返回 false。
func SafeDivide(a, b int) (int, bool) {
	if b == 0 {
		return 0, false
	}
	return a / b, true
}

// --- 5.2 命名返回值 ---
func section2_NamedReturn() {
	fmt.Println("\n--- 5.2 命名返回值 ---")

	area, perimeter := RectInfo(3, 4)
	fmt.Printf("RectInfo(3, 4): area=%.0f, perimeter=%.0f\n", area, perimeter)

	// 命名返回值在文档中更有表现力，但 naked return 要谨慎使用
	fmt.Println("命名返回值的优势：文档自描述 + 可在 defer 中修改")
}

// RectInfo 返回矩形的面积和周长（命名返回值）。
func RectInfo(width, height float64) (area, perimeter float64) {
	area = width * height
	perimeter = 2 * (width + height)
	return // naked return，等价于 return area, perimeter
}

// --- 5.3 变参函数 ---
func section3_Variadic() {
	fmt.Println("\n--- 5.3 变参函数 ---")

	fmt.Printf("Sum() = %d\n", Sum())
	fmt.Printf("Sum(1, 2, 3) = %d\n", Sum(1, 2, 3))
	fmt.Printf("Sum(10, 20) = %d\n", Sum(10, 20))

	// 展开切片作为变参
	nums := []int{1, 2, 3, 4, 5}
	fmt.Printf("Sum(nums...) = %d\n", Sum(nums...))

	// 变参也可以是字符串
	fmt.Printf("JoinWords: %s\n", JoinWords("Hello", "Go", "World"))
}

// Sum 计算任意个整数的和。
func Sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// JoinWords 用空格连接多个单词。
func JoinWords(words ...string) string {
	return strings.Join(words, " ")
}

// --- 5.4 函数是一等值 ---
func section4_FirstClass() {
	fmt.Println("\n--- 5.4 函数是一等值 ---")

	// 函数可以赋值给变量
	add := func(a, b int) int {
		return a + b
	}
	fmt.Printf("add(2, 3) = %d\n", add(2, 3))

	// 函数可以作为参数
	result := ApplyOperation(10, 5, add)
	fmt.Printf("ApplyOperation(10, 5, add) = %d\n", result)

	// 也可以直接传入匿名函数
	result2 := ApplyOperation(10, 5, func(a, b int) int {
		return a - b
	})
	fmt.Printf("ApplyOperation(10, 5, sub) = %d\n", result2)

	// 函数可以作为返回值
	mul := MakeOperation("mul")
	fmt.Printf("MakeOperation(\"mul\")(4, 5) = %d\n", mul(4, 5))
}

// ApplyOperation 对 a 和 b 应用指定的二元操作。
func ApplyOperation(a, b int, op func(int, int) int) int {
	return op(a, b)
}

// MakeOperation 根据名称返回对应的二元操作函数。
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

// --- 5.5 闭包与捕获 ---
func section5_Closure() {
	fmt.Println("\n--- 5.5 闭包与捕获 ---")

	// 闭包：函数 + 其引用的外部变量环境
	double := MakeMultiplier(2)
	triple := MakeMultiplier(3)

	fmt.Printf("double(5) = %d\n", double(5))
	fmt.Printf("triple(5) = %d\n", triple(5))

	// 闭包共享变量的陷阱（Go 1.22 之前循环变量问题已修复）
	funcs := make([]func() int, 3)
	for i := range 3 {
		funcs[i] = func() int {
			return i * i // i 是每次迭代独立的（Go 1.22+）
		}
	}
	fmt.Print("闭包捕获循环变量: ")
	for _, f := range funcs {
		fmt.Printf("%d ", f())
	}
	fmt.Println()

	// 闭包修改外部变量
	counter := MakeCounter()
	fmt.Printf("counter() = %d\n", counter())
	fmt.Printf("counter() = %d\n", counter())
	fmt.Printf("counter() = %d\n", counter())
}

// MakeMultiplier 返回一个将输入乘以 factor 的函数。
func MakeMultiplier(factor int) func(int) int {
	return func(x int) int {
		return x * factor
	}
}

// MakeCounter 返回一个每次调用自增的计数器函数。
func MakeCounter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

// --- 5.6 defer 的执行顺序与参数求值 ---
func section6_DeferOrder() {
	fmt.Println("\n--- 5.6 defer 的执行顺序与参数求值 ---")

	fmt.Println("defer 演示开始")
	defer fmt.Println("defer 1: 最先注册，最后执行")
	defer fmt.Println("defer 2: 第二注册，第二执行")
	defer fmt.Println("defer 3: 最后注册，最先执行")
	fmt.Println("defer 演示结束（普通语句先执行）")

	// defer 的参数在注册时求值
	x := 1
	defer fmt.Printf("defer 中 x 的值（注册时求值）: %d\n", x)
	x = 100
	fmt.Printf("x 被修改为 %d，但 defer 的参数已经确定\n", x)
}

// --- 5.7 defer 配合命名返回值的坑 ---
func section7_DeferNamedReturn() {
	fmt.Println("\n--- 5.7 defer 配合命名返回值的坑 ---")

	fmt.Printf("deferWithNamedReturn() = %d\n", deferWithNamedReturn())
	fmt.Printf("deferWithoutNamedReturn() = %d\n", deferWithoutNamedReturn())
	fmt.Printf("deferWithParam(5) = %d\n", deferWithParam(5))
}

// deferWithNamedReturn 演示 defer 修改命名返回值。
func deferWithNamedReturn() (result int) {
	result = 1
	defer func() {
		result++ // 修改命名返回值
	}()
	return result // 实际返回 2
}

// deferWithoutNamedReturn 演示 defer 无法修改非命名返回值。
func deferWithoutNamedReturn() int {
	result := 1
	defer func() {
		result++ // 修改的是局部变量，不影响返回值
	}()
	return result // 返回 1
}

// deferWithParam 演示 defer 匿名函数的参数求值时机。
func deferWithParam(n int) int {
	defer func(x int) {
		fmt.Printf("defer 中 x = %d（注册时的值）\n", x)
	}(n)
	n = 100
	return n
}

// --- 5.8 递归 ---
func section8_Recursion() {
	fmt.Println("\n--- 5.8 递归 ---")

	fmt.Printf("Factorial(5) = %d\n", Factorial(5))
	fmt.Printf("Fibonacci(10) = %d\n", Fibonacci(10))

	// 尾递归优化：Go 编译器不做尾递归优化，但可以用循环改写
	fmt.Printf("FactorialLoop(5) = %d（循环版，无栈溢出风险）\n", FactorialLoop(5))
}

// Factorial 计算阶乘（递归实现）。
func Factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * Factorial(n-1)
}

// Fibonacci 计算第 n 个斐波那契数（递归实现，效率低）。
func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return Fibonacci(n-1) + Fibonacci(n-2)
}

// FactorialLoop 计算阶乘（循环实现，更高效）。
func FactorialLoop(n int) int {
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}

// --- 5.9 init 回顾 ---

// initCounter 用于演示 init 的执行时机。
var initCounter = initFunc()

func initFunc() int {
	fmt.Println("\n--- 5.9 init 回顾 ---")
	fmt.Println("[init] initFunc() 在包初始化时执行，早于 main 和 Demo()")
	return 42
}

func init() {
	fmt.Println("[init] go05_functions 的 init() 执行")
}

func section9_InitReview() {
	fmt.Println("init 函数在包被导入时自动执行，一个包可以有多个 init。")
	fmt.Printf("包级变量 initCounter 的值：%d\n", initCounter)
	fmt.Println("init 的典型用途：注册驱动、验证配置、预计算常量表。")
	fmt.Println("注意：init 顺序由依赖关系和文件名字母顺序决定，不要编写依赖特定 init 顺序的代码。")
}
