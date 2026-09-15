// Package go03_operators_fmt 演示 Go 语言的运算符与格式化输出。
//
// 涵盖主题：
//   - 算术、比较、逻辑、位运算符
//   - 运算符优先级与结合性
//   - fmt.Printf 格式化动词与标志
//   - fmt.Sprintf、fmt.Fprintf 系列函数
//   - Stringer 接口与自定义格式化
//   - fmt.Scan 系列输入函数
package go03_operators_fmt

import (
	"fmt"
	"math"
	"strings"
)

// Demo 是第 3 章的入口函数，按小节顺序演示运算符与格式化输出。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s: %s ==========\n", Chapter, "operators_fmt", ChapterTitle)
	fmt.Println("========================================")

	section1_ArithmeticOperators()
	section2_ComparisonOperators()
	section3_LogicalOperators()
	section4_BitwiseOperators()
	section5_OperatorPrecedence()
	section6_PrintfBasics()
	section7_PrintfFlags()
	section8_PrintfWidth()
	section9_StringerInterface()
	section10_SprintfFprintf()

	fmt.Println("========================================")
	fmt.Println("========== 运算符与格式化输出演示结束 ==========")
	fmt.Println()
}

// --- 3.1 算术运算符 ---
func section1_ArithmeticOperators() {
	fmt.Println("\n--- 3.1 算术运算符 ---")

	a, b := 10, 3
	fmt.Printf("%d + %d = %d\n", a, b, a+b)
	fmt.Printf("%d - %d = %d\n", a, b, a-b)
	fmt.Printf("%d * %d = %d\n", a, b, a*b)
	fmt.Printf("%d / %d = %d (整数除法，截断)\n", a, b, a/b)
	fmt.Printf("%d %% %d = %d (取余)\n", a, b, a%b)

	// 浮点除法
	fa, fb := 10.0, 3.0
	fmt.Printf("%.2f / %.2f = %.4f (浮点除法)\n", fa, fb, fa/fb)

	// 负数取余：结果符号与被除数相同
	fmt.Printf("%d %% %d = %d\n", -10, 3, -10%3)   // -1
	fmt.Printf("%d %% %d = %d\n", 10, -3, 10%-3)   // 1
	fmt.Printf("%d %% %d = %d\n", -10, -3, -10%-3) // -1

	// 自增自减
	x := 5
	x++ // 只有后置，不支持 ++x
	fmt.Printf("x++ 后 x = %d\n", x)
	x--
	fmt.Printf("x-- 后 x = %d\n", x)

	// Go 不支持 ++x、--x，也不支持 x = y++ 这种表达式
	// y := x++  // 编译错误：syntax error: unexpected ++
}

// --- 3.2 比较运算符 ---
func section2_ComparisonOperators() {
	fmt.Println("\n--- 3.2 比较运算符 ---")

	a, b := 10, 20
	fmt.Printf("%d == %d: %t\n", a, b, a == b)
	fmt.Printf("%d != %d: %t\n", a, b, a != b)
	fmt.Printf("%d < %d: %t\n", a, b, a < b)
	fmt.Printf("%d <= %d: %t\n", a, b, a <= b)
	fmt.Printf("%d > %d: %t\n", a, b, a > b)
	fmt.Printf("%d >= %d: %t\n", a, b, a >= b)

	// 字符串比较：按字典序（UTF-8 字节序）
	s1, s2 := "apple", "banana"
	fmt.Printf("%q < %q: %t\n", s1, s2, s1 < s2)

	// 切片和 map 不能直接用 == 比较（除了与 nil 比较）
	var slice []int
	fmt.Printf("slice == nil: %t\n", slice == nil)
	// slice1 := []int{1, 2}
	// slice2 := []int{1, 2}
	// fmt.Println(slice1 == slice2)  // 编译错误：invalid operation

	// 结构体可以比较（如果所有字段都可比较）
	type Point struct{ X, Y int }
	p1 := Point{1, 2}
	p2 := Point{1, 2}
	fmt.Printf("Point{1,2} == Point{1,2}: %t\n", p1 == p2)
}

// --- 3.3 逻辑运算符 ---
func section3_LogicalOperators() {
	fmt.Println("\n--- 3.3 逻辑运算符 ---")

	t, f := true, false
	fmt.Printf("true && false = %t\n", t && f)
	fmt.Printf("true || false = %t\n", t || f)
	fmt.Printf("!true = %t\n", !t)

	// 短路求值：&& 左侧为 false 时不执行右侧，|| 左侧为 true 时不执行右侧
	fmt.Println("\n短路求值演示:")
	x := 0
	if x != 0 && 10/x > 1 {
		fmt.Println("不会执行到这里")
	} else {
		fmt.Println("x != 0 为 false，右侧 10/x 未执行，避免了除零")
	}

	y := 10
	if y > 5 || expensive() {
		fmt.Println("y > 5 为 true，右侧 expensive() 未执行")
	}
}

func expensive() bool {
	fmt.Println("  expensive() 被调用了")
	return true
}

// --- 3.4 位运算符 ---
func section4_BitwiseOperators() {
	fmt.Println("\n--- 3.4 位运算符 ---")

	a, b := 0b1100, 0b1010 // 12, 10
	fmt.Printf("a = %04b (%d), b = %04b (%d)\n", a, a, b, b)
	fmt.Printf("a & b  = %04b (%d) (按位与)\n", a&b, a&b)
	fmt.Printf("a | b  = %04b (%d) (按位或)\n", a|b, a|b)
	fmt.Printf("a ^ b  = %04b (%d) (按位异或)\n", a^b, a^b)
	fmt.Printf("^a     = %b (%d) (按位取反)\n", ^a, ^a)
	fmt.Printf("a << 1 = %04b (%d) (左移)\n", a<<1, a<<1)
	fmt.Printf("a >> 1 = %04b (%d) (右移)\n", a>>1, a>>1)

	// &^ 是位清除运算符：a &^ b = a & (^b)
	fmt.Printf("a &^ b = %04b (%d) (位清除，清除 a 中 b 为 1 的位)\n", a&^b, a&^b)

	// 常见位操作技巧
	fmt.Println("\n常见位操作:")
	n := 42
	fmt.Printf("n = %d, 二进制 %08b\n", n, n)
	fmt.Printf("设置第 2 位: %d (二进制 %08b)\n", n|1<<2, n|1<<2)
	fmt.Printf("清除第 3 位: %d (二进制 %08b)\n", n&^(1<<3), n&^(1<<3))
	fmt.Printf("切换第 1 位: %d (二进制 %08b)\n", n^(1<<1), n^(1<<1))
	fmt.Printf("检查第 3 位: %t\n", n&(1<<3) != 0)
}

// --- 3.5 运算符优先级 ---
func section5_OperatorPrecedence() {
	fmt.Println("\n--- 3.5 运算符优先级 ---")

	// Go 运算符优先级（从高到低）：
	// 1. * / % << >> & &^
	// 2. + - | ^
	// 3. == != < <= > >=
	// 4. &&
	// 5. ||

	a := 2 + 3*4 // 先乘后加
	fmt.Printf("2 + 3 * 4 = %d (先乘后加)\n", a)

	b := 10<<1 + 1 // 先移位后加
	// 移位运算符优先级高于加减法
	// 实际是 (10 << 1) + 1 = 21
	fmt.Printf("10 << 1 + 1 = %d (实际是 (10 << 1) + 1)\n", b)

	c := 5 < 3 == false // 先比较后判等
	fmt.Printf("5 < 3 == false: %t (先算 5 < 3 = false，再算 false == false = true)\n", c)

	// 位运算优先级易错
	d := 1<<2 + 3 // (1 << 2) + 3 = 7，移位优先级高于加法
	fmt.Printf("1 << 2 + 3 = %d (是 (1 << 2) + 3 = 7)\n", d)

	// 最佳实践：不确定时加括号
	e := (2 + 3) * 4
	fmt.Printf("(2 + 3) * 4 = %d (括号优先)\n", e)
}

// --- 3.6 fmt.Printf 基础 ---
func section6_PrintfBasics() {
	fmt.Println("\n--- 3.6 fmt.Printf 基础 ---")

	// 常用格式化动词
	fmt.Printf("%%v (默认格式): %v\n", 42)
	fmt.Printf("%%d (十进制): %d\n", 42)
	fmt.Printf("%%b (二进制): %b\n", 42)
	fmt.Printf("%%o (八进制): %o\n", 42)
	fmt.Printf("%%x (十六进制小写): %x\n", 42)
	fmt.Printf("%%X (十六进制大写): %X\n", 42)

	// 浮点数
	fmt.Printf("%%f (浮点): %f\n", 3.14159)
	fmt.Printf("%%.2f (保留 2 位小数): %.2f\n", 3.14159)
	fmt.Printf("%%e (科学计数法小写): %e\n", 123456.789)
	fmt.Printf("%%E (科学计数法大写): %E\n", 123456.789)
	fmt.Printf("%%g (自动选择 %%f 或 %%e): %g\n", 123456.789)

	// 字符串和字符
	fmt.Printf("%%s (字符串): %s\n", "Hello")
	fmt.Printf("%%q (带引号的字符串): %q\n", "Hello\nWorld")
	fmt.Printf("%%c (字符): %c\n", 'A')

	// 布尔值
	fmt.Printf("%%t (布尔值): %t\n", true)

	// 指针
	x := 42
	fmt.Printf("%%p (指针地址): %p\n", &x)

	// %T 打印类型
	fmt.Printf("%%T (类型): %T\n", 42)
	fmt.Printf("%%T (类型): %T\n", "hello")
	fmt.Printf("%%T (类型): %T\n", 3.14)
}

// --- 3.7 fmt.Printf 标志 ---
func section7_PrintfFlags() {
	fmt.Println("\n--- 3.7 fmt.Printf 标志 ---")

	// + 强制显示正负号
	fmt.Printf("%%+d: %+d, %+d\n", 42, -42)

	// - 左对齐（默认右对齐）
	fmt.Printf("%%5d 右对齐: |%5d|\n", 42)
	fmt.Printf("%%-5d 左对齐: |%-5d|\n", 42)

	// 0 用零填充（只对数字）
	fmt.Printf("%%05d: %05d\n", 42)
	fmt.Printf("%%08.2f: %08.2f\n", 3.14)

	// 空格：正数前加空格，负数前加负号（用于对齐）
	fmt.Printf("%% d: |% d|, |% d|\n", 42, -42)

	// # 备用格式
	fmt.Printf("%%#x (带 0x 前缀): %#x\n", 42)
	fmt.Printf("%%#o (带 0 前缀): %#o\n", 42)
	fmt.Printf("%%#v (Go 语法表示): %#v\n", []int{1, 2, 3})

	// 组合使用
	fmt.Printf("%%+#08x: %+#08x\n", 255)
}

// --- 3.8 Printf 宽度和精度 ---
func section8_PrintfWidth() {
	fmt.Println("\n--- 3.8 Printf 宽度和精度 ---")

	// 宽度：指定最小字段宽度
	fmt.Printf("%%5d: |%5d|\n", 42)
	fmt.Printf("%%5d: |%5d| (不足 5 位右对齐)\n", 1)

	// 精度：对浮点数表示小数位数，对字符串表示最大字符数
	fmt.Printf("%%.2f: %.2f\n", 3.14159)
	fmt.Printf("%%.5s: %.5s (只取前 5 个字符)\n", "Hello, World")

	// 宽度和精度可以用 * 从参数获取
	width, precision := 10, 2
	fmt.Printf("%%*.*f: |%*.*f|\n", width, precision, 3.14159)

	// 对齐和填充
	fmt.Printf("|%%10s| 右对齐: |%10s|\n", "Go")
	fmt.Printf("|%%-10s| 左对齐: |%-10s|\n", "Go")
	fmt.Printf("|%%010d| 零填充: |%010d|\n", 42)
}

// --- 3.9 Stringer 接口 ---

// Point 是一个简单的点结构体
type Point struct {
	X, Y int
}

// String 实现 fmt.Stringer 接口，自定义格式化输出
func (p Point) String() string {
	return fmt.Sprintf("Point(%d, %d)", p.X, p.Y)
}

func section9_StringerInterface() {
	fmt.Println("\n--- 3.9 Stringer 接口 ---")

	p := Point{3, 4}
	fmt.Printf("%%v: %v (调用 String 方法)\n", p)
	fmt.Printf("%%s: %s (调用 String 方法)\n", p)
	fmt.Printf("%%#v: %#v (Go 语法表示，不调用 String)\n", p)

	// 错误的 String 实现会导致无限递归
	// type BadPoint struct{ X, Y int }
	// func (p BadPoint) String() string {
	//     return fmt.Sprintf("%v", p)  // 无限递归！
	// }
	// 正确做法：
	// return fmt.Sprintf("BadPoint{X:%d, Y:%d}", p.X, p.Y)
}

// --- 3.10 Sprintf 和 Fprintf ---
func section10_SprintfFprintf() {
	fmt.Println("\n--- 3.10 Sprintf 和 Fprintf ---")

	// Sprintf 返回格式化后的字符串
	s := fmt.Sprintf("圆周率约等于 %.2f", math.Pi)
	fmt.Printf("Sprintf 结果: %s\n", s)

	// Fprintf 写入 io.Writer
	var buf strings.Builder
	fmt.Fprintf(&buf, "写入 buf: %d + %d = %d\n", 1, 2, 3)
	fmt.Printf("Fprintf 结果: %s", buf.String())

	// Print 系列不格式化，Println 自动换行
	fmt.Print("Print 不换行 ")
	fmt.Print("拼接输出\n")
	fmt.Println("Println 自动换行")

	// Errorf 创建包含格式化信息的 error
	err := fmt.Errorf("打开文件失败: %s", "file.txt")
	fmt.Printf("Errorf 结果: %v\n", err)
}
