// Command go-learn 依次运行各章的演示代码。
//
// 用法：
//
//	go run ./cmd/go-learn
//
// 每写完一章，就在下面的 import 区和 main 里按章节顺序登记该章的 Demo 函数。
// 章节包放在 internal/chapter/ 下，命名形如 goNN_主题。
package main

import (
	"go-learn/internal/chapter/go01_hello"
	"go-learn/internal/chapter/go02_variables"
)

func main() {
	// 章节注册区：新增章节时在这里按顺序追加 goNN_主题.Demo()。
	go01_hello.Demo()
	go02_variables.Demo()
}
