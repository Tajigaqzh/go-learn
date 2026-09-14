package go01_hello

import "fmt"

// init 与 demo.go 里的 init 一起演示「同一个包可以拆成多个文件」。
//
// 同一个包里多个 init 的执行顺序是：先按文件名排序（demo.go 排在 version.go 前面），
// 同一个文件内再按声明顺序执行。所以这行输出排在 demo.go 的 init 之后。
func init() {
	fmt.Println("[启动阶段] 步骤 2：version.go 的 init() 执行（文件名排序在 demo.go 之后）")
}

const (
	// Chapter 是当前章节的编号，文档和测试都引用它。
	Chapter = 1
	// ChapterTitle 是当前章节的标题，Demo 用它打印章节标题栏。
	ChapterTitle = "环境搭建与第一个 Go 程序"
)
