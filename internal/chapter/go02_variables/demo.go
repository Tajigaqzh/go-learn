// Package go02_variables 演示 Go 语言的变量、常量与基本类型。
//
// 涵盖主题：
//   - var 与 := 的声明方式、零值、批量声明
//   - 作用域、遮蔽（shadowing）
//   - const 与 iota、无类型常量
//   - 整数族、溢出、浮点精度、复数
//   - byte 与 rune、类型转换
//   - 类型定义与类型别名、空白标识符
package go02_variables

import (
	"fmt"
	"math"
	"unsafe"
)

// Demo 是第 2 章的入口函数，按小节顺序演示变量与类型的核心概念。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s: %s ==========\n", Chapter, "variables", ChapterTitle)
	fmt.Println("========================================")

	section1_VarDeclarations()
	section2_ZeroValues()
	section3_ScopeAndShadowing()
	section4_ConstAndIota()
	section5_IntegerTypes()
	section6_FloatAndComplex()
	section7_ByteAndRune()
	section8_TypeConversions()
	section9_TypeDefinitionVsAlias()
	section10_BlankIdentifier()

	fmt.Println("========================================")
	fmt.Println("========== 变量、常量与基本类型演示结束 ==========")
	fmt.Println()
}

// --- 2.1 var 与 := 声明 ---
func section1_VarDeclarations() {
	fmt.Println("\n--- 2.1 var 与 := 声明 ---")

	// 方式一：var 显式类型
	var a int = 10
	fmt.Printf("var a int = 10 => a=%d, type=%T\n", a, a)

	// 方式二：var 类型推导
	var b = 20
	fmt.Printf("var b = 20 => b=%d, type=%T\n", b, b)

	// 方式三：短声明（只能在函数内）
	c := 30
	fmt.Printf("c := 30 => c=%d, type=%T\n", c, c)

	// 批量声明
	var x, y, z int = 1, 2, 3
	fmt.Printf("var x, y, z int = 1, 2, 3 => x=%d, y=%d, z=%d\n", x, y, z)

	// 混合类型批量声明
	var name, age, active = "Alice", 25, true
	fmt.Printf("name=%s, age=%d, active=%t\n", name, age, active)

	// 块式声明
	var (
		host = "localhost"
		port = 8080
	)
	fmt.Printf("host=%s, port=%d\n", host, port)

	// := 不能用于包级别，只能在函数内
	// 短声明要求至少有一个新变量
	d := 40
	d, e := 50, 60 // d 被重新赋值，e 是新变量
	fmt.Printf("d=%d, e=%d\n", d, e)
}

// --- 2.2 零值 ---
func section2_ZeroValues() {
	fmt.Println("\n--- 2.2 零值 ---")

	var i int
	var f float64
	var b bool
	var s string
	var p *int
	var slice []int
	var m map[string]int
	var fn func()

	fmt.Printf("int 零值: %d\n", i)
	fmt.Printf("float64 零值: %f\n", f)
	fmt.Printf("bool 零值: %t\n", b)
	fmt.Printf("string 零值: %q (空字符串，不是 nil)\n", s)
	fmt.Printf("*int 零值: %v\n", p)
	fmt.Printf("[]int 零值: %v (nil 切片)\n", slice)
	fmt.Printf("map 零值: %v (nil map)\n", m)
	fmt.Printf("func 零值: %p\n", fn)

	// 零值的可用性：string、slice 可以直接用，map 和指针需要初始化
	s = s + "hello"                // string 零值可以拼接
	slice = append(slice, 1, 2, 3) // nil 切片可以 append
	fmt.Printf("拼接后的 string: %q\n", s)
	fmt.Printf("append 后的 slice: %v\n", slice)

	// nil map 不能赋值，会 panic
	// m["key"] = 1 // panic: assignment to entry in nil map
	m = make(map[string]int)
	m["key"] = 1
	fmt.Printf("初始化后的 map: %v\n", m)
}

// --- 2.3 作用域与遮蔽 ---
func section3_ScopeAndShadowing() {
	fmt.Println("\n--- 2.3 作用域与遮蔽 ---")

	x := 10
	fmt.Printf("外层 x=%d\n", x)

	{
		// 内层重新声明同名变量，遮蔽外层
		x := 20
		fmt.Printf("内层 x=%d\n", x)

		x = 25 // 修改内层 x
		fmt.Printf("内层修改后 x=%d\n", x)
	}

	fmt.Printf("外层 x 未受影响=%d\n", x)

	// if 初始化语句中的变量作用域仅限 if-else
	if y := 100; y > 50 {
		fmt.Printf("if 块内 y=%d\n", y)
	} else {
		fmt.Printf("else 块内 y=%d\n", y)
	}
	// fmt.Println(y) // 编译错误：undefined: y

	// for 循环变量作用域（Go 1.22 后每次迭代独立）
	for i := 0; i < 3; i++ {
		fmt.Printf("循环 i=%d\n", i)
	}
	// fmt.Println(i) // 编译错误：undefined: i
}

// --- 2.4 const 与 iota ---
func section4_ConstAndIota() {
	fmt.Println("\n--- 2.4 const 与 iota ---")

	// 常量必须在编译期确定，不能是运行时值
	const pi = 3.14159
	const greeting = "Hello"
	fmt.Printf("pi=%f, greeting=%s\n", pi, greeting)

	// 批量常量
	const (
		StatusOK    = 200
		StatusError = 500
	)
	fmt.Printf("StatusOK=%d, StatusError=%d\n", StatusOK, StatusError)

	// iota：常量生成器，从 0 开始递增
	const (
		Sunday    = iota // 0
		Monday           // 1
		Tuesday          // 2
		Wednesday        // 3
	)
	fmt.Printf("Sunday=%d, Monday=%d, Tuesday=%d, Wednesday=%d\n",
		Sunday, Monday, Tuesday, Wednesday)

	// iota 跳过与表达式
	const (
		_  = iota             // 0，跳过
		KB = 1 << (10 * iota) // 1 << 10 = 1024
		MB                    // 1 << 20 = 1048576
		GB                    // 1 << 30 = 1073741824
	)
	fmt.Printf("KB=%d, MB=%d, GB=%d\n", KB, MB, GB)

	// 每个 const 块 iota 重置
	const (
		A = iota // 0
		B        // 1
	)
	const (
		C = iota // 重新从 0 开始
		D        // 1
	)
	fmt.Printf("A=%d, B=%d, C=%d, D=%d\n", A, B, C, D)

	// 无类型常量：高精度，可赋值给兼容类型
	const big = 1e100 // 无类型浮点常量
	// var i int = big // 编译错误：constant 1e+100 overflows int
	var f float64 = big
	fmt.Printf("big 赋给 float64: %e\n", f)

	const small = 1
	var i8 int8 = small   // 无类型整数可赋给 int8
	var i64 int64 = small // 也可赋给 int64
	fmt.Printf("small 赋给 int8=%d, int64=%d\n", i8, i64)
}

// --- 2.5 整数类型 ---
func section5_IntegerTypes() {
	fmt.Println("\n--- 2.5 整数类型 ---")

	// Go 的整数类型：int8, int16, int32, int64, uint8, uint16, uint32, uint64
	// int 和 uint 的大小取决于平台（32 位或 64 位）
	var i8 int8 = 127
	var u8 uint8 = 255
	var i32 int32 = 2147483647
	var u32 uint32 = 4294967295
	fmt.Printf("int8 最大值=%d, uint8 最大值=%d\n", i8, u8)
	fmt.Printf("int32 最大值=%d, uint32 最大值=%d\n", i32, u32)

	// int 和 uint 的大小
	fmt.Printf("int 字节数=%d, uint 字节数=%d\n",
		unsafe.Sizeof(int(0)), unsafe.Sizeof(uint(0)))

	// 溢出：编译期常量溢出报错，运行时溢出回绕
	// const overflow int8 = 128 // 编译错误：constant 128 overflows int8
	var x int8 = 127
	x = x + 1 // 运行时溢出，回绕为 -128
	fmt.Printf("127 + 1 (int8 溢出) = %d\n", x)

	var y uint8 = 255
	y = y + 1 // 回绕为 0
	fmt.Printf("255 + 1 (uint8 溢出) = %d\n", y)

	// 位运算
	a := 0b1010 // 二进制字面量
	b := 0o12   // 八进制字面量
	c := 0xA    // 十六进制字面量
	fmt.Printf("a=0b1010=%d, b=0o12=%d, c=0xA=%d\n", a, b, c)

	fmt.Printf("a & b = %d\n", a&b)   // 按位与
	fmt.Printf("a | b = %d\n", a|b)   // 按位或
	fmt.Printf("a ^ b = %d\n", a^b)   // 按位异或
	fmt.Printf("a << 1 = %d\n", a<<1) // 左移
	fmt.Printf("a >> 1 = %d\n", a>>1) // 右移
}

// --- 2.6 浮点数与复数 ---
func section6_FloatAndComplex() {
	fmt.Println("\n--- 2.6 浮点数与复数 ---")

	// float32 和 float64
	var f32 float32 = 3.14
	var f64 float64 = 2.718281828459045
	fmt.Printf("float32=%f, float64=%f\n", f32, f64)

	// 浮点精度问题
	a := 0.1
	b := 0.2
	c := a + b
	fmt.Printf("0.1 + 0.2 = %.20f (不完全等于 0.3)\n", c)
	fmt.Printf("0.1 + 0.2 == 0.3 ? %t\n", c == 0.3)

	// 特殊浮点值
	fmt.Printf("正无穷=%f\n", math.Inf(1))
	fmt.Printf("负无穷=%f\n", math.Inf(-1))
	fmt.Printf("NaN=%f\n", math.NaN())
	fmt.Printf("NaN == NaN ? %t (NaN 不等于任何值，包括自身)\n", math.NaN() == math.NaN())

	// 复数：complex64 和 complex128
	var z1 complex128 = 1 + 2i
	var z2 complex128 = complex(3, 4) // 等价于 3 + 4i
	fmt.Printf("z1=%v, z2=%v\n", z1, z2)
	fmt.Printf("z1 + z2 = %v\n", z1+z2)
	fmt.Printf("real(z1)=%f, imag(z1)=%f\n", real(z1), imag(z1))
}

// --- 2.7 byte 与 rune ---
func section7_ByteAndRune() {
	fmt.Println("\n--- 2.7 byte 与 rune ---")

	// byte 是 uint8 的别名，用于表示 ASCII 字符或原始字节
	var b byte = 'A' // 单引号表示字符字面量
	fmt.Printf("byte b='A' => %d (ASCII 码)\n", b)
	fmt.Printf("byte b 作为字符=%c\n", b)

	// rune 是 int32 的别名，用于表示 Unicode 码点
	var r rune = '中'
	fmt.Printf("rune r='中' => %d (Unicode 码点 U+4E2D)\n", r)
	fmt.Printf("rune r 作为字符=%c\n", r)

	// 字符串遍历：[]byte 按字节，[]rune 按字符
	s := "Go语言"
	fmt.Printf("字符串 %q 的长度（字节数）=%d\n", s, len(s))
	fmt.Printf("按字节遍历: ")
	for i := 0; i < len(s); i++ {
		fmt.Printf("%X ", s[i])
	}
	fmt.Println()

	fmt.Printf("按 rune 遍历: ")
	for _, r := range s {
		fmt.Printf("%c(U+%04X) ", r, r)
	}
	fmt.Println()

	// []rune 转换
	runes := []rune(s)
	fmt.Printf("转为 []rune 后长度=%d\n", len(runes))
	fmt.Printf("第 3 个字符=%c\n", runes[2])
}

// --- 2.8 类型转换 ---
func section8_TypeConversions() {
	fmt.Println("\n--- 2.8 类型转换 ---")

	// Go 要求显式类型转换，不支持隐式转换
	var i int = 42
	var f float64 = float64(i)
	var u uint = uint(i)
	fmt.Printf("int=%d, float64=%f, uint=%d\n", i, f, u)

	// 浮点转整数：截断小数部分
	var pi float64 = 3.14159
	var intPi int = int(pi)
	fmt.Printf("float64(3.14159) 转 int = %d\n", intPi)

	// 有损转换：大类型转小类型可能溢出或截断
	var big int64 = 1000
	var small int8 = int8(big) // big 超出 int8 范围，发生截断
	fmt.Printf("int64(1000) 转 int8 = %d (发生截断)\n", small)

	// string 与 []byte、[]rune 转换
	s := "Hello"
	bytes := []byte(s)
	runes := []rune(s)
	fmt.Printf("string=%q, []byte=%v, []rune=%v\n", s, bytes, runes)

	s2 := string(bytes)
	s3 := string(runes)
	fmt.Printf("[]byte 转 string=%q, []rune 转 string=%q\n", s2, s3)

	// 不同类型不能直接运算
	var a int = 10
	var b int64 = 20
	// c := a + b // 编译错误：invalid operation: a + b (mismatched types int and int64)
	c := int64(a) + b
	fmt.Printf("int(10) + int64(20) = %d\n", c)
}

// --- 2.9 类型定义与类型别名 ---

// 类型定义：创建新类型
type Celsius float64
type Fahrenheit float64

// 类型别名：为现有类型取别名
type MyInt = int

func section9_TypeDefinitionVsAlias() {
	fmt.Println("\n--- 2.9 类型定义与类型别名 ---")

	// 类型定义：Celsius 和 float64 是不同类型
	var c Celsius = 100.0
	// var f float64 = c // 编译错误：cannot use c (type Celsius) as type float64
	var f float64 = float64(c) // 需要显式转换
	fmt.Printf("Celsius=%f, float64=%f\n", c, f)

	// 类型别名：MyInt 和 int 完全相同
	var a MyInt = 10
	var b int = 20
	sum := a + b // 可以直接运算，MyInt 和 int 完全等价
	fmt.Printf("MyInt(10) + int(20) = %d\n", sum)

	// byte 和 rune 是预定义的类型别名
	// type byte = uint8
	// type rune = int32
	var x byte = 255
	var y uint8 = x // 无需转换
	fmt.Printf("byte=%d, uint8=%d\n", x, y)
}

// --- 2.10 空白标识符 ---
func section10_BlankIdentifier() {
	fmt.Println("\n--- 2.10 空白标识符 ---")

	// _ 用于忽略不需要的返回值或变量
	x, _ := getValue() // 忽略第二个返回值
	fmt.Printf("只要第一个返回值 x=%d\n", x)

	_, y := getValue() // 忽略第一个返回值
	fmt.Printf("只要第二个返回值 y=%d\n", y)

	// 空白导入：执行包的 init 函数但不使用包名
	// import _ "database/sql/driver"

	// 类型断言：只检查类型，不使用值
	var i interface{} = 42
	_, ok := i.(int)
	fmt.Printf("i 是 int 类型吗？%t\n", ok)

	// for range 忽略索引或值
	nums := []int{10, 20, 30}
	fmt.Print("只要值: ")
	for _, v := range nums {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	fmt.Print("只要索引: ")
	for i := range nums {
		fmt.Printf("%d ", i)
	}
	fmt.Println()
}

func getValue() (int, int) {
	return 42, 100
}
