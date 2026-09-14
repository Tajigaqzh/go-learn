// Package go01_hello 演示环境搭建与第一个 Go 程序的核心概念。
//
// 本章覆盖 go 工具链基础、module 初始化、包与 main、初始化顺序、
// 以及项目目录布局的最佳实践。
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go01_hello/
package go01_hello

import (
	"fmt"
	"os"
	"runtime"
)

// Demo 运行第 1 章的所有演示代码。
func Demo() {
	fmt.Printf("\n========== go01_hello: %s ==========\n", ChapterTitle)

	demo1BasicProgram()
	demo2PackageAndImport()
	demo3Variables()
	demo4GoEnv()
	demo5InitOrder()

	fmt.Println("========== 环境搭建与第一个 Go 程序演示结束 ==========")
	fmt.Println()
}

// --- 1. 最简单的 Go 程序 ---
func demo1BasicProgram() {
	fmt.Println("--- 1. 最简单的 Go 程序 ---")

	// Hello, World! 是学习任何语言的起点
	fmt.Println("Hello, World!")

	// fmt.Println 会自动换行，fmt.Print 不换行
	fmt.Print("Hello, ")
	fmt.Print("Go!\n")

	// fmt.Printf 支持格式化输出
	name := "Go"
	fmt.Printf("欢迎来到 %s 的世界，这是第 %d 章\n", name, Chapter)

	fmt.Println()
}

// --- 2. 包与导入 ---
func demo2PackageAndImport() {
	fmt.Println("--- 2. 包与导入 ---")

	// 每个 Go 文件都属于一个包
	// main 包是程序的入口，必须有 main 函数
	// 其他包通过 import 导入

	fmt.Println("当前包：go01_hello（非 main 包，提供 Demo 函数供 main 包调用）")
	fmt.Println("导入的包：fmt（格式化 IO）、os（操作系统接口）、runtime（运行时信息）")

	// 包名通常是目录名的最后一段，但可以不同
	// 包名应该简短、全小写、单数形式

	fmt.Println()
}

// --- 3. 变量声明与类型 ---
func demo3Variables() {
	fmt.Println("--- 3. 变量声明与零值（第 2 章预告）---")

	// 完整声明：var 名字 类型 = 值
	var message string = "Go 是静态类型语言"
	fmt.Println(message)

	// 类型推导：Go 会根据右侧的值推导类型
	var count = 42
	fmt.Printf("count 的类型是 %T，值是 %d\n", count, count)

	// 短变量声明：只能在函数内使用，最常用
	language := "Go"
	year := 2009
	fmt.Printf("%s 诞生于 %d 年\n", language, year)

	// 零值：声明但不初始化时，变量会被赋予零值
	var num int     // 数字的零值是 0
	var text string // 字符串的零值是 ""
	var flag bool   // 布尔的零值是 false
	fmt.Printf("零值：int=%d, string=\"%s\", bool=%t\n", num, text, flag)

	fmt.Println()
}

// --- 4. go env 与环境信息 ---
func demo4GoEnv() {
	fmt.Println("--- 4. go env 与环境信息 ---")

	// runtime 包提供运行时信息
	fmt.Printf("Go 版本：%s\n", runtime.Version())
	fmt.Printf("操作系统：%s\n", runtime.GOOS)
	fmt.Printf("架构：%s\n", runtime.GOARCH)
	fmt.Printf("CPU 核心数：%d\n", runtime.NumCPU())

	// GOPATH 与 module 模式
	// Go 1.11+ 引入 module，不再强制 GOPATH 布局
	// go mod init 初始化 module 后，可以在任意目录工作
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// 环境变量为空不等于默认值就是空，默认值要问 go env
		gopath = "(环境变量未设置，实际取值以 go env GOPATH 为准)"
	}
	fmt.Printf("GOPATH：%s\n", gopath)

	// GOPROXY 控制依赖下载源
	goproxy := os.Getenv("GOPROXY")
	if goproxy == "" {
		goproxy = "(环境变量未设置，实际取值以 go env GOPROXY 为准)"
	}
	fmt.Printf("GOPROXY：%s\n", goproxy)

	fmt.Println()
}

// --- 5. 初始化顺序 ---

// packageVar 用来演示「包级变量在 init 之前初始化」。
//
// 它的初始化表达式是一次函数调用，所以这里打印的内容落在 main 之前，
// 也就是整个程序输出的最前面——这正是第 5 节要解释的现象。
var packageVar = initPackageVar()

func initPackageVar() string {
	fmt.Println("[启动阶段] 步骤 1：包级变量初始化（按依赖关系与声明顺序）")
	return "package-level"
}

// init 函数在包被导入时自动执行，一个包可以有多个 init。
func init() {
	fmt.Println("[启动阶段] 步骤 2：demo.go 的 init() 执行（同文件内按声明顺序）")
}

func demo5InitOrder() {
	fmt.Println("--- 5. 初始化顺序 ---")

	// 这里不重复打印启动阶段的内容，只解释最前面那几行的来历
	fmt.Println("输出最前面带 [启动阶段] 的几行，就是在 main 之前打印的")
	fmt.Println("完整顺序：导入的包先初始化 → 包级变量 → init → main")
	fmt.Printf("包级变量 packageVar 的值：%s\n", packageVar)
	fmt.Println("想在某段代码必须在 main 之前跑（注册驱动、加载配置）时，就用 init")

	fmt.Println()
}

// GetGreeting 返回一个问候语，演示导出函数（首字母大写）。
func GetGreeting(name string) string {
	if name == "" {
		name = "World"
	}
	return fmt.Sprintf("Hello, %s!", name)
}

// internalHelper 是未导出函数（首字母小写），只能在包内使用。
func internalHelper() string {
	return "internal"
}
