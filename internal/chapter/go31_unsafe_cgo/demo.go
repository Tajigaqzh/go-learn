// Package go31_unsafe_cgo 演示 unsafe、cgo 与代码生成。
//
// 本章讲解底层能力：unsafe 包的三个函数与指针转换规则、cgo 类型映射与内存管理、
// //export 反向导出、构建约束、//go:generate 与 //go:embed。
//
// 注意：cgo 示例仅展示接口设计与规则，不实际引入 C 编译器依赖。
package go31_unsafe_cgo

import (
	"fmt"
	"unsafe"
)

// Demo 是第 31 章的统一入口。
func Demo() {
	fmt.Println("\n========== go31_unsafe_cgo: unsafe、cgo 与代码生成 ==========")

	// 31.1 unsafe 包的三个函数
	demo31_1()

	// 31.2 unsafe.Pointer 与 uintptr 转换规则
	demo31_2()

	// 31.3 runtime.KeepAlive 的作用
	demo31_3()

	// 31.4 cgo 类型映射概览
	demo31_4()

	// 31.5 C.CString 与内存管理
	demo31_5()

	// 31.6 //export 反向导出
	demo31_6()

	// 31.7 cgo 的代价与构建约束
	demo31_7()

	// 31.8 //go:generate 与 //go:embed
	demo31_8()

	fmt.Println("\n========== unsafe、cgo 与代码生成演示结束 ==========")
}

// demo31_1 演示 unsafe.Sizeof / Offsetof / Alignof。
func demo31_1() {
	fmt.Println("\n--- 31.1 unsafe 包的三个函数 ---")

	type Example struct {
		a bool   // 1 字节
		b int32  // 4 字节
		c int64  // 8 字节
		d string // 16 字节（指针 + 长度）
	}

	var e Example
	fmt.Printf("Sizeof(Example) = %d 字节\n", unsafe.Sizeof(e))
	fmt.Printf("Sizeof(e.a) = %d, Offsetof(e.a) = %d, Alignof(e.a) = %d\n",
		unsafe.Sizeof(e.a), unsafe.Offsetof(e.a), unsafe.Alignof(e.a))
	fmt.Printf("Sizeof(e.b) = %d, Offsetof(e.b) = %d, Alignof(e.b) = %d\n",
		unsafe.Sizeof(e.b), unsafe.Offsetof(e.b), unsafe.Alignof(e.b))
	fmt.Printf("Sizeof(e.c) = %d, Offsetof(e.c) = %d, Alignof(e.c) = %d\n",
		unsafe.Sizeof(e.c), unsafe.Offsetof(e.c), unsafe.Alignof(e.c))
	fmt.Printf("Sizeof(e.d) = %d, Offsetof(e.d) = %d, Alignof(e.d) = %d\n",
		unsafe.Sizeof(e.d), unsafe.Offsetof(e.d), unsafe.Alignof(e.d))

	fmt.Println()
	fmt.Println("说明：")
	fmt.Println("  - Sizeof 返回类型大小（字节）")
	fmt.Println("  - Offsetof 返回字段在结构体中的偏移量")
	fmt.Println("  - Alignof 返回类型的对齐要求")
	fmt.Println("  - 结构体有内存对齐，实际大小 >= 各字段大小之和")
}

// demo31_2 演示 unsafe.Pointer 与 uintptr 的转换规则。
func demo31_2() {
	fmt.Println("\n--- 31.2 unsafe.Pointer 与 uintptr 转换规则 ---")

	fmt.Println("unsafe.Pointer 是通用指针类型，可以指向任意类型")
	fmt.Println("uintptr 是整数类型，可以参与算术运算")
	fmt.Println()

	fmt.Println("合法的转换模式：")
	fmt.Println("  1. *T -> unsafe.Pointer -> *U（类型转换）")
	fmt.Println("  2. unsafe.Pointer -> uintptr（临时计算偏移）")
	fmt.Println("  3. uintptr -> unsafe.Pointer（重建指针，必须一行完成）")
	fmt.Println()

	type Point struct {
		x, y int32
	}
	p := Point{x: 10, y: 20}

	// 合法：通过 unsafe.Pointer 访问字段
	fmt.Println("示例 1：通过偏移访问字段 y")
	py := (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&p)) + unsafe.Offsetof(p.y)))
	fmt.Printf("  p.y = %d, *py = %d\n", p.y, *py)
	*py = 30
	fmt.Printf("  修改后 p.y = %d\n", p.y)
	fmt.Println()

	fmt.Println("注意事项：")
	fmt.Println("  - uintptr 不是指针，GC 不会追踪它")
	fmt.Println("  - uintptr -> unsafe.Pointer 转换必须在一个表达式里完成")
	fmt.Println("  - 不要把 uintptr 存到变量里再转回 unsafe.Pointer（对象可能被 GC 移动）")
	fmt.Println("  - 使用 runtime.KeepAlive 保证对象在使用期间不被回收")
}

// demo31_3 演示 runtime.KeepAlive 的作用。
func demo31_3() {
	fmt.Println("\n--- 31.3 runtime.KeepAlive 的作用 ---")

	fmt.Println("问题场景：")
	fmt.Println("  当你通过 uintptr 访问对象内存时，Go 可能会认为原对象已无引用，")
	fmt.Println("  从而在 GC 时移动或回收它，导致 uintptr 指向无效内存。")
	fmt.Println()

	fmt.Println("解决方案：")
	fmt.Println("  在使用 uintptr 后立即调用 runtime.KeepAlive(obj)，")
	fmt.Println("  告诉 GC 该对象在此之前必须保持有效。")
	fmt.Println()

	fmt.Println("示例：")
	fmt.Println("  // 错误写法（对象可能被 GC）")
	fmt.Println("  ptr := uintptr(unsafe.Pointer(&obj))")
	fmt.Println("  // ... 使用 ptr ...")
	fmt.Println()
	fmt.Println("  // 正确写法")
	fmt.Println("  ptr := uintptr(unsafe.Pointer(&obj))")
	fmt.Println("  // ... 使用 ptr ...")
	fmt.Println("  runtime.KeepAlive(&obj)  // 保证 obj 在此之前不被回收")
}

// demo31_4 演示 cgo 类型映射概览（伪代码）。
func demo31_4() {
	fmt.Println("\n--- 31.4 cgo 类型映射概览 ---")

	fmt.Println("cgo 提供 C 与 Go 类型的互操作，常见映射：")
	fmt.Println()
	fmt.Println("| C 类型        | Go 类型           | 说明                     |")
	fmt.Println("| ------------- | ----------------- | ------------------------ |")
	fmt.Println("| char          | C.char            | 1 字节整数               |")
	fmt.Println("| int           | C.int             | 平台相关（通常 4 字节）  |")
	fmt.Println("| long          | C.long            | 平台相关（32/64 位）     |")
	fmt.Println("| float         | C.float           | 4 字节浮点               |")
	fmt.Println("| double        | C.double          | 8 字节浮点               |")
	fmt.Println("| void*         | unsafe.Pointer    | 通用指针                 |")
	fmt.Println("| char*         | *C.char           | C 字符串（NUL 结尾）     |")
	fmt.Println()

	fmt.Println("字符串转换：")
	fmt.Println("  - Go string -> C char*: 用 C.CString(s)，返回 *C.char")
	fmt.Println("  - C char* -> Go string: 用 C.GoString(cs)")
	fmt.Println("  - C.CString 分配的内存必须手动释放：C.free(unsafe.Pointer(cs))")
	fmt.Println()

	fmt.Println("切片传递：")
	fmt.Println("  - Go 切片不能直接传给 C，需要传指针 + 长度")
	fmt.Println("  - 示例：C.process(&data[0], C.int(len(data)))")
	fmt.Println()

	fmt.Println("注意：本章不实际引入 cgo（避免 C 编译器依赖），仅展示接口规则")
}

// demo31_5 演示 C.CString 与内存管理（伪代码）。
func demo31_5() {
	fmt.Println("\n--- 31.5 C.CString 与内存管理 ---")

	fmt.Println("C.CString 的内存模型：")
	fmt.Println("  1. C.CString(s) 分配 C 堆内存，复制 Go 字符串，追加 NUL")
	fmt.Println("  2. 返回 *C.char 指针，Go GC 不管理这块内存")
	fmt.Println("  3. 使用完毕后必须调用 C.free(unsafe.Pointer(cs)) 释放")
	fmt.Println()

	fmt.Println("示例（伪代码）：")
	fmt.Println("  // import \"C\"")
	fmt.Println("  s := \"hello\"")
	fmt.Println("  cs := C.CString(s)         // 分配 C 内存")
	fmt.Println("  defer C.free(unsafe.Pointer(cs))  // 确保释放")
	fmt.Println("  C.some_c_function(cs)      // 调用 C 函数")
	fmt.Println()

	fmt.Println("常见错误：")
	fmt.Println("  - 忘记 C.free -> 内存泄漏")
	fmt.Println("  - 使用已释放的 cs -> 野指针")
	fmt.Println("  - 把 Go 字符串指针直接传给 C -> 不安全（GC 可能移动）")
}

// demo31_6 演示 //export 反向导出（伪代码）。
func demo31_6() {
	fmt.Println("\n--- 31.6 //export 反向导出 ---")

	fmt.Println("//export 用于把 Go 函数导出给 C 调用（动态库、回调）")
	fmt.Println()

	fmt.Println("示例（伪代码）：")
	fmt.Println("  // export Add")
	fmt.Println("  func Add(a, b C.int) C.int {")
	fmt.Println("      return a + b")
	fmt.Println("  }")
	fmt.Println()
	fmt.Println("编译成动态库：")
	fmt.Println("  go build -buildmode=c-shared -o libadd.so add.go")
	fmt.Println()
	fmt.Println("C 代码调用：")
	fmt.Println("  // 自动生成的 libadd.h 包含 Add 声明")
	fmt.Println("  #include \"libadd.h\"")
	fmt.Println("  int result = Add(1, 2);")
	fmt.Println()

	fmt.Println("规则：")
	fmt.Println("  - 导出函数的参数和返回值必须是 C 类型（C.int、*C.char 等）")
	fmt.Println("  - 不能导出泛型函数")
	fmt.Println("  - panic 不能穿透到 C，必须 recover")
	fmt.Println("  - 生成的头文件与 .so/.dll 一起分发")
}

// demo31_7 演示 cgo 的代价与构建约束。
func demo31_7() {
	fmt.Println("\n--- 31.7 cgo 的代价与构建约束 ---")

	fmt.Println("cgo 的代价：")
	fmt.Println("  1. 编译慢（需要 C 编译器，交叉编译复杂）")
	fmt.Println("  2. 调用开销大（Go <-> C 切换，每次调用约 100ns）")
	fmt.Println("  3. 二进制体积大（静态链接 C 库）")
	fmt.Println("  4. 丧失纯 Go 优势（不能 CGO_ENABLED=0 交叉编译）")
	fmt.Println()

	fmt.Println("构建约束（build tags）：")
	fmt.Println("  // +build cgo")
	fmt.Println("  此文件只在 CGO_ENABLED=1 时编译")
	fmt.Println()
	fmt.Println("  // +build !cgo")
	fmt.Println("  此文件只在 CGO_ENABLED=0 时编译（纯 Go fallback）")
	fmt.Println()

	fmt.Println("禁用 cgo：")
	fmt.Println("  CGO_ENABLED=0 go build")
	fmt.Println("  适用场景：")
	fmt.Println("    - 交叉编译（如在 x86 上编译 ARM 二进制）")
	fmt.Println("    - Docker 多阶段构建（避免 C 运行时依赖）")
	fmt.Println("    - 纯静态二进制")
	fmt.Println()

	fmt.Println("示例：go-sqlite3 库依赖 cgo，禁用后需换用纯 Go 的 modernc.org/sqlite")
}

// demo31_8 演示 //go:generate 与 //go:embed。
func demo31_8() {
	fmt.Println("\n--- 31.8 //go:generate 与 //go:embed ---")

	fmt.Println("//go:generate 用于代码生成：")
	fmt.Println("  在源文件顶部写：")
	fmt.Println("    //go:generate stringer -type=Status")
	fmt.Println("  然后运行：")
	fmt.Println("    go generate ./...")
	fmt.Println("  自动生成 status_string.go")
	fmt.Println()

	fmt.Println("常用生成工具：")
	fmt.Println("  - stringer: 为枚举生成 String() 方法")
	fmt.Println("  - mockgen: 生成 mock 接口（gomock）")
	fmt.Println("  - protoc-gen-go: 从 .proto 生成 Go 代码")
	fmt.Println("  - wire: 依赖注入代码生成")
	fmt.Println()

	fmt.Println("//go:embed 用于嵌入静态资源（Go 1.16+）：")
	fmt.Println("  //go:embed templates/*.html")
	fmt.Println("  var templates embed.FS")
	fmt.Println()
	fmt.Println("  优点：")
	fmt.Println("    - 编译时打包到二进制，无需外部文件")
	fmt.Println("    - 支持单个文件、目录、glob 匹配")
	fmt.Println("  限制：")
	fmt.Println("    - 只能嵌入同包或子目录文件")
	fmt.Println("    - 不能嵌入 . 或 _ 开头的文件（除非显式指定）")
}
