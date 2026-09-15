// Package go06_pointers 演示 Go 语言的指针、值传递与内存基础。
//
// 涵盖主题：
//   - & 取地址与 * 解引用
//   - Go 只有值传递，指针作为"传引用"的替代方案
//   - 用指针修改调用方数据
//   - new 与 make 的区别与使用场景
//   - nil 指针的安全检查与 panic
//   - 指针接收者 vs 值接收者对方法行为的影响
//   - 逃逸分析初窥（-gcflags=-m）
//   - Go 为什么没有指针运算
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go06_pointers/
package go06_pointers

import (
	"fmt"
	"math"
)

// Demo 是第 6 章的入口函数，按小节顺序演示指针与内存的核心概念。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s: %s ==========\n", Chapter, "pointers", ChapterTitle)
	fmt.Println("========================================")

	section1_AddressAndDereference()
	section2_ValuePassing()
	section3_ModifyCallerData()
	section4_NewVsMake()
	section5_NilPointer()
	section6_PointerReceiver()
	section7_EscapeAnalysis()
	section8_NoPointerArithmetic()

	fmt.Println("========================================")
	fmt.Println("========== 指针、值与内存入门演示结束 ==========")
	fmt.Println()
}

// --- 6.1 取地址与解引用 ---
func section1_AddressAndDereference() {
	fmt.Println("\n--- 6.1 取地址与解引用 ---")

	x := 42
	p := &x // & 取地址，p 的类型是 *int

	fmt.Printf("x = %d\n", x)
	fmt.Printf("p = %p（x 的地址）\n", p)
	fmt.Printf("*p = %d（解引用得到 x 的值）\n", *p)

	*p = 100 // 通过指针修改 x
	fmt.Printf("修改 *p 后，x = %d\n", x)

	// 指针的指针
	pp := &p
	fmt.Printf("pp = %p（p 的地址）\n", pp)
	fmt.Printf("*pp = %p（p 的值，即 x 的地址）\n", *pp)
	fmt.Printf("**pp = %d（两次解引用得到 x 的值）\n", **pp)
}

// --- 6.2 Go 只有值传递 ---
func section2_ValuePassing() {
	fmt.Println("\n--- 6.2 Go 只有值传递 ---")

	x := 10
	fmt.Printf("调用前 x = %d\n", x)
	tryModifyValue(x)
	fmt.Printf("调用 tryModifyValue(x) 后 x = %d（未被修改）\n", x)

	tryModifyPointer(&x)
	fmt.Printf("调用 tryModifyPointer(&x) 后 x = %d（通过指针修改成功）\n", x)

	// 切片是引用类型，但底层仍是值传递（拷贝了头部信息）
	s := []int{1, 2, 3}
	fmt.Printf("调用前 s = %v\n", s)
	modifySlice(s)
	fmt.Printf("调用 modifySlice(s) 后 s = %v（底层数组被修改）\n", s)
}

func tryModifyValue(n int) {
	n = 999
	fmt.Printf("  tryModifyValue 内部 n = %d（只是副本）\n", n)
}

func tryModifyPointer(p *int) {
	*p = 999
	fmt.Printf("  tryModifyPointer 内部 *p = %d（修改了原变量）\n", *p)
}

func modifySlice(s []int) {
	if len(s) > 0 {
		s[0] = 100 // 修改底层数组
	}
	fmt.Printf("  modifySlice 内部 s = %v\n", s)
}

// --- 6.3 用指针修改调用方数据 ---
func section3_ModifyCallerData() {
	fmt.Println("\n--- 6.3 用指针修改调用方数据 ---")

	a, b := 3, 5
	fmt.Printf("交换前 a=%d, b=%d\n", a, b)
	Swap(&a, &b)
	fmt.Printf("Swap(&a, &b) 后 a=%d, b=%d\n", a, b)

	// 修改结构体
	counter := Counter{value: 10}
	fmt.Printf("Increment 前 counter.value = %d\n", counter.value)
	Increment(&counter)
	fmt.Printf("Increment(&counter) 后 counter.value = %d\n", counter.value)

	// 工厂函数返回指针
	p := NewCounter(20)
	fmt.Printf("NewCounter(20) 返回的指针：%p，value = %d\n", p, p.value)
}

// Counter 是一个简单的计数器。
type Counter struct {
	value int
}

// Swap 交换两个整数的值。
func Swap(a, b *int) {
	*a, *b = *b, *a
}

// Increment 将计数器的值加 1。
func Increment(c *Counter) {
	if c == nil {
		return
	}
	c.value++
}

// NewCounter 创建一个初始值为 v 的计数器，返回指针。
func NewCounter(v int) *Counter {
	return &Counter{value: v}
}

// --- 6.4 new 与 make ---
func section4_NewVsMake() {
	fmt.Println("\n--- 6.4 new 与 make ---")

	// new(T) 分配零值内存，返回 *T
	pi := new(int)
	fmt.Printf("new(int) 返回 *int = %p，*pi = %d（零值）\n", pi, *pi)
	*pi = 42
	fmt.Printf("*pi = %d\n", *pi)

	// new 也可以用于结构体
	pc := new(Counter)
	fmt.Printf("new(Counter) 返回 *Counter，value = %d（零值）\n", pc.value)

	// make 只用于 slice、map、channel
	s := make([]int, 3, 5)
	fmt.Printf("make([]int, 3, 5) = %v，len=%d，cap=%d\n", s, len(s), cap(s))

	m := make(map[string]int, 10)
	m["key"] = 42
	fmt.Printf("make(map[string]int, 10) 后 m = %v\n", m)

	// ch := make(chan int, 2) // channel 在后续章节详细讲
}

// --- 6.5 nil 指针 ---
func section5_NilPointer() {
	fmt.Println("\n--- 6.5 nil 指针 ---")

	var p *int
	fmt.Printf("未初始化的指针 p = %v\n", p)

	// nil 指针可以比较
	if p == nil {
		fmt.Println("p 是 nil，安全")
	}

	// 解引用 nil 指针会 panic
	// fmt.Println(*p) // panic: runtime error: invalid memory address or nil pointer dereference

	// 正确做法：使用前检查
	safeDereference(p)

	q := 100
	safeDereference(&q)
}

func safeDereference(p *int) {
	if p == nil {
		fmt.Println("safeDereference: 收到 nil 指针，跳过")
		return
	}
	fmt.Printf("safeDereference: *p = %d\n", *p)
}

// --- 6.6 指针接收者 vs 值接收者 ---
func section6_PointerReceiver() {
	fmt.Println("\n--- 6.6 指针接收者 vs 值接收者 ---")

	v := Vector{X: 3, Y: 4}
	fmt.Printf("原始向量 v = %v\n", v)

	// 值接收者：不修改原变量
	fmt.Printf("v.Length() = %.2f（值接收者）\n", v.Length())
	fmt.Printf("调用后 v = %v（未变）\n", v)

	// 指针接收者：修改原变量
	v.Scale(2)
	fmt.Printf("v.Scale(2) 后 v = %v（指针接收者修改了原变量）\n", v)

	// 自动取地址：编译器会在可寻址的值上自动插入 &
	v2 := Vector{1, 1}
	v2.Scale(10) // 等价于 (&v2).Scale(10)
	fmt.Printf("v2.Scale(10) 后 v2 = %v（自动取地址）\n", v2)

	// 但临时值不可寻址
	// Vector{1,2}.Scale(10) // 编译错误：cannot call pointer method on Vector literal
}

// Vector 表示二维向量。
type Vector struct {
	X, Y float64
}

// Length 计算向量的长度（值接收者，不修改原向量）。
func (v Vector) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

// Scale 将向量缩放 factor 倍（指针接收者，修改原向量）。
func (v *Vector) Scale(factor float64) {
	if v == nil {
		return
	}
	v.X *= factor
	v.Y *= factor
}

// --- 6.7 逃逸分析初窥 ---
func section7_EscapeAnalysis() {
	fmt.Println("\n--- 6.7 逃逸分析初窥 ---")

	fmt.Println("逃逸分析（escape analysis）是编译器决定变量分配在栈上还是堆上的过程。")
	fmt.Println("栈分配：函数返回后自动回收，效率高。")
	fmt.Println("堆分配（逃逸）：由 GC 管理，生命周期更长。")

	p := allocateOnHeap()
	fmt.Printf("allocateOnHeap 返回的指针：%p，value = %d\n", p, *p)
	fmt.Println("这个指针指向的 int 逃逸到了堆上，因为函数返回后仍然被使用。")

	local := allocateOnStack()
	fmt.Printf("allocateOnStack 返回值：%d（局部变量，通常在栈上）\n", local)
}

func allocateOnHeap() *int {
	x := 42
	return &x // x 逃逸到堆上
}

func allocateOnStack() int {
	x := 42
	return x // 返回值拷贝，x 留在栈上
}

// --- 6.8 为什么没有指针运算 ---
func section8_NoPointerArithmetic() {
	fmt.Println("\n--- 6.8 为什么没有指针运算 ---")

	arr := [3]int{10, 20, 30}
	p := &arr[0]
	fmt.Printf("arr = %v\n", arr)
	fmt.Printf("&arr[0] = %p，*p = %d\n", p, *p)

	// Go 不支持 p++ 或 p+1
	// p++ // 编译错误：invalid operation: p++ (non-numeric type *int)

	// 安全的替代方式：用索引
	for i := range arr {
		fmt.Printf("arr[%d] = %d（地址 %p）\n", i, arr[i], &arr[i])
	}

	fmt.Println("Go 禁止指针运算，防止越界访问和内存破坏。")
	fmt.Println("需要遍历数组时，使用索引或 range，而不是指针算术。")
}
