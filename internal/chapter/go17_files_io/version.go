package go17_files_io

// init 与 demo.go 的 init 一起演示「同一个包可以拆成多个文件」。
func init() {}

const (
	// Chapter 是当前章节的编号，文档和测试都引用它。
	Chapter = 17
	// ChapterTitle 是当前章节的标题，Demo 用它打印章节标题栏。
	ChapterTitle = "文件、路径与 IO"
)
