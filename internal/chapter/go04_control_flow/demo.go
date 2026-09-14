// Package go04_control_flow 演示 Go 的控制流：if、for、range、switch、break/continue、goto
package go04_control_flow

import (
	"fmt"
)

// Demo 是本章的入口函数，展示 Go 的所有控制流结构
func Demo() {
	fmt.Println("\n========================================")
	fmt.Println("========== go04_control_flow ==========")
	fmt.Println("========================================")

	section1_IfBasics()
	section2_IfWithInit()
	section3_ForClassic()
	section4_ForConditionOnly()
	section5_ForInfinite()
	section6_RangeSlice()
	section7_RangeMap()
	section8_RangeString()
	section9_BreakContinueLabel()
	section10_Switch()
	section11_SwitchNoExpr()
	section12_Fallthrough()
	section13_Goto()

	fmt.Println("========================================")
	fmt.Println("========== 控制流演示结束 ==========")
	fmt.Println("========================================")
}

// section1_IfBasics 演示 if 的基本用法
func section1_IfBasics() {
	fmt.Println("\n--- 4.1 if 的基本形式 ---")

	x := 10
	if x > 5 {
		fmt.Println("x > 5")
	}

	if x > 20 {
		fmt.Println("x > 20")
	} else {
		fmt.Println("x <= 20")
	}

	if x < 0 {
		fmt.Println("负数")
	} else if x == 0 {
		fmt.Println("零")
	} else {
		fmt.Println("正数")
	}

	// Go 没有三元运算符，必须用 if
	result := ""
	if x%2 == 0 {
		result = "偶数"
	} else {
		result = "奇数"
	}
	fmt.Printf("x=%d 是%s\n", x, result)
}

// section2_IfWithInit 演示 if 的初始化语句
func section2_IfWithInit() {
	fmt.Println("\n--- 4.2 if 的初始化语句 ---")

	// 在 if 中声明的变量只在 if/else 块内可见
	if y := compute(); y > 10 {
		fmt.Printf("y=%d > 10\n", y)
	} else {
		fmt.Printf("y=%d <= 10\n", y)
	}
	// fmt.Println(y) // 编译错误：y 未定义

	// 常见模式：错误处理
	if err := doSomething(); err != nil {
		fmt.Printf("错误：%v\n", err)
	} else {
		fmt.Println("成功")
	}
}

func compute() int {
	return 15
}

func doSomething() error {
	return nil // 模拟成功
}

// section3_ForClassic 演示经典三段式 for 循环
func section3_ForClassic() {
	fmt.Println("\n--- 4.3 for 的经典形式 ---")

	// 标准三段式
	for i := 0; i < 5; i++ {
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	// 初始化和后置语句可以省略分号
	sum := 0
	for i := 1; i <= 10; i++ {
		sum += i
	}
	fmt.Printf("1+2+...+10 = %d\n", sum)

	// 多变量初始化
	for i, j := 0, 10; i < j; i, j = i+1, j-1 {
		fmt.Printf("i=%d, j=%d\n", i, j)
	}
}

// section4_ForConditionOnly 演示只有条件的 for（类似 while）
func section4_ForConditionOnly() {
	fmt.Println("\n--- 4.4 for 的条件形式（类似 while）---")

	n := 1
	for n < 100 {
		n *= 2
	}
	fmt.Printf("第一个 >= 100 的 2 的幂：%d\n", n)
}

// section5_ForInfinite 演示无限循环
func section5_ForInfinite() {
	fmt.Println("\n--- 4.5 for 的无限循环 ---")

	count := 0
	for {
		count++
		if count > 3 {
			break
		}
		fmt.Printf("循环 %d\n", count)
	}
	fmt.Println("退出无限循环")
}

// section6_RangeSlice 演示 range 遍历切片
func section6_RangeSlice() {
	fmt.Println("\n--- 4.6 range 遍历切片 ---")

	nums := []int{10, 20, 30, 40}

	// 同时获取索引和值
	for i, v := range nums {
		fmt.Printf("nums[%d] = %d\n", i, v)
	}

	// 只要索引
	fmt.Print("索引：")
	for i := range nums {
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	// 只要值（用空白标识符忽略索引）
	fmt.Print("值：")
	for _, v := range nums {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// Go 1.22+ 循环变量语义：每次迭代都是新变量
	// 下面的代码在 1.22+ 中是安全的
	var funcs []func()
	for i := range 3 {
		funcs = append(funcs, func() {
			fmt.Printf("捕获的 i=%d\n", i)
		})
	}
	for _, f := range funcs {
		f()
	}
}

// section7_RangeMap 演示 range 遍历 map
func section7_RangeMap() {
	fmt.Println("\n--- 4.7 range 遍历 map ---")

	m := map[string]int{
		"Alice": 25,
		"Bob":   30,
		"Carol": 28,
	}

	// 遍历 key 和 value（顺序随机）
	fmt.Println("map 遍历（顺序随机）：")
	for k, v := range m {
		fmt.Printf("%s: %d\n", k, v)
	}

	// 只要 key
	fmt.Print("键：")
	for k := range m {
		fmt.Printf("%s ", k)
	}
	fmt.Println()
}

// section8_RangeString 演示 range 遍历字符串
func section8_RangeString() {
	fmt.Println("\n--- 4.8 range 遍历字符串 ---")

	s := "Hello,世界"

	// range 遍历字符串时，索引是字节位置，值是 rune
	for i, r := range s {
		fmt.Printf("[%d] %c (U+%04X)\n", i, r, r)
	}

	// 字节长度 vs rune 个数
	fmt.Printf("len(s)=%d（字节数），rune 个数=%d\n", len(s), len([]rune(s)))
}

// section9_BreakContinueLabel 演示 break、continue 和标签
func section9_BreakContinueLabel() {
	fmt.Println("\n--- 4.9 break、continue 和标签 ---")

	// break 跳出循环
	for i := 0; i < 10; i++ {
		if i == 5 {
			break
		}
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	// continue 跳过本次迭代
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			continue
		}
		fmt.Printf("%d ", i)
	}
	fmt.Println()

	// 标签：跳出外层循环
	fmt.Println("标签跳出嵌套循环：")
Outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i*j == 4 {
				fmt.Printf("找到 i=%d, j=%d，跳出外层\n", i, j)
				break Outer
			}
			fmt.Printf("(%d,%d) ", i, j)
		}
		fmt.Println()
	}
}

// section10_Switch 演示 switch 的基本用法
func section10_Switch() {
	fmt.Println("\n--- 4.10 switch 的基本形式 ---")

	day := 3
	switch day {
	case 1:
		fmt.Println("星期一")
	case 2:
		fmt.Println("星期二")
	case 3:
		fmt.Println("星期三")
	case 4, 5:
		// 多个 case 值
		fmt.Println("星期四或星期五")
	default:
		fmt.Println("周末")
	}

	// switch 可以带初始化语句
	switch n := compute(); {
	case n < 0:
		fmt.Println("负数")
	case n == 0:
		fmt.Println("零")
	default:
		fmt.Println("正数")
	}
}

// section11_SwitchNoExpr 演示无表达式 switch（类似 if-else 链）
func section11_SwitchNoExpr() {
	fmt.Println("\n--- 4.11 无表达式 switch ---")

	x := 42
	switch {
	case x < 0:
		fmt.Println("x 是负数")
	case x < 10:
		fmt.Println("x 是个位数")
	case x < 100:
		fmt.Println("x 是两位数")
	default:
		fmt.Println("x >= 100")
	}
}

// section12_Fallthrough 演示 fallthrough
func section12_Fallthrough() {
	fmt.Println("\n--- 4.12 fallthrough ---")

	// Go 的 switch 默认自动 break，不需要显式写
	// fallthrough 强制执行下一个 case
	v := 1
	switch v {
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
}

// section13_Goto 演示 goto（慎用）
func section13_Goto() {
	fmt.Println("\n--- 4.13 goto（慎用）---")

	i := 0
loop:
	if i < 3 {
		fmt.Printf("i=%d ", i)
		i++
		goto loop
	}
	fmt.Println("\ngoto 示例结束")

	// goto 不能跳过变量声明
	// goto skip
	// x := 10 // 编译错误
	// skip:
	//   fmt.Println(x)
}
