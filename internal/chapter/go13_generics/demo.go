// Package go13_generics 演示 Go 的泛型：类型参数、类型约束、comparable、
// 联合类型与 ~、类型推导、泛型函数与泛型类型、约束中的方法集、
// slices/maps/cmp 标准库，以及泛型与接口的取舍。
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go13_generics/
package go13_generics

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
)

// Demo 运行第 13 章的所有示例。
func Demo() {
	fmt.Println("========== go13_generics: 泛型 ==========")

	fmt.Println("\n--- 1. 为什么需要泛型：从重复代码说起 ---")
	demoWhyGenerics()

	fmt.Println("\n--- 2. 类型参数的语法：[T any] ---")
	demoTypeParameters()

	fmt.Println("\n--- 3. 类型约束：comparable 与自定义约束 ---")
	demoConstraints()

	fmt.Println("\n--- 4. 联合类型约束与底层类型 ~T ---")
	demoUnionAndTilde()

	fmt.Println("\n--- 5. 类型推导：什么时候可以省略类型参数 ---")
	demoTypeInference()

	fmt.Println("\n--- 6. 泛型类型：带类型参数的结构体 ---")
	demoGenericTypes()

	fmt.Println("\n--- 7. 约束中的方法集 ---")
	demoMethodSetInConstraints()

	fmt.Println("\n--- 8. 标准库：slices / maps / cmp ---")
	demoStdlibGenerics()

	fmt.Println("\n--- 9. 泛型不能做的事：类型参数化的方法 ---")
	demoGenericsLimitations()

	fmt.Println("\n--- 10. 泛型与接口的取舍 ---")
	demoGenericsVsInterfaces()

	fmt.Println("\n========== 泛型演示结束 ==========")
}

// demoWhyGenerics 展示泛型出现之前的重复代码问题。
func demoWhyGenerics() {
	ints := []int{3, 1, 4, 1, 5}
	strs := []string{"go", "rust", "python"}

	fmt.Printf("MinInt(%v) = %d\n", ints, MinInt(ints))
	fmt.Printf("MinString(%v) = %q\n", strs, MinString(strs))
	fmt.Println("问题：MinInt 和 MinString 逻辑完全一样，但必须写两遍。")
	fmt.Println("泛型前的解法：要么用 interface{} + 反射（慢且失去类型安全），")
	fmt.Println("要么用代码生成（麻烦），要么接受重复（违背 DRY）。")
}

// MinInt 返回 int 切片的最小值（泛型前的写法）。
func MinInt(s []int) int {
	if len(s) == 0 {
		panic("empty slice")
	}
	m := s[0]
	for _, v := range s[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// MinString 返回 string 切片的最小值（泛型前的写法）。
func MinString(s []string) string {
	if len(s) == 0 {
		panic("empty slice")
	}
	m := s[0]
	for _, v := range s[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// demoTypeParameters 演示类型参数的基本语法。
func demoTypeParameters() {
	ints := []int{3, 1, 4, 1, 5}
	floats := []float64{3.14, 1.41, 2.71}

	fmt.Printf("Min[int](%v) = %d\n", ints, Min[int](ints))
	fmt.Printf("Min[float64](%v) = %.2f\n", floats, Min[float64](floats))
	fmt.Println("语法：func Min[T cmp.Ordered](s []T) T")
	fmt.Println("  [T cmp.Ordered] 是类型参数列表，T 是类型参数名，cmp.Ordered 是约束")
	fmt.Println("  s []T 和返回值 T 都用到了这个类型参数")
	fmt.Println("  调用时用 Min[int](...) 指定具体类型，或让编译器推导（见 5）")
}

// Min 返回可比较类型切片的最小值（泛型写法）。
func Min[T cmp.Ordered](s []T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// demoConstraints 演示 comparable 约束和自定义约束。
func demoConstraints() {
	ints := []int{1, 2, 3, 2, 1}
	strs := []string{"go", "rust", "go"}

	fmt.Printf("Unique[int](%v) = %v\n", ints, Unique(ints))
	fmt.Printf("Unique[string](%v) = %v\n", strs, Unique(strs))
	fmt.Println("comparable 约束：类型参数可以用 == 和 != 比较")
	fmt.Println("  所有基本类型、指针、数组、结构体（字段全可比）都是 comparable")
	fmt.Println("  切片、map、函数不是 comparable")

	fmt.Printf("\nSumNumbers[int](%v) = %d\n", ints, SumNumbers(ints))
	floats := []float64{1.1, 2.2, 3.3}
	fmt.Printf("SumNumbers[float64](%v) = %.1f\n", floats, SumNumbers(floats))
	fmt.Println("自定义约束 Number：用 interface + 联合类型列举允许的类型")
}

// Unique 返回去重后的切片（元素顺序保留首次出现）。
func Unique[T comparable](s []T) []T {
	seen := make(map[T]bool)
	var result []T
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// Number 是一个自定义约束，限制为整数和浮点数。
type Number interface {
	int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		float32 | float64
}

// SumNumbers 返回数字切片的和。
func SumNumbers[T Number](s []T) T {
	var sum T
	for _, v := range s {
		sum += v
	}
	return sum
}

// demoUnionAndTilde 演示联合类型与底层类型匹配符 ~。
func demoUnionAndTilde() {
	type MyInt int
	type MyString string

	plainInts := []int{1, 2, 3}
	myInts := []MyInt{10, 20, 30}

	fmt.Printf("SumIntegers[int](%v) = %d\n", plainInts, SumIntegers(plainInts))
	fmt.Printf("SumIntegers[MyInt](%v) = %d\n", myInts, SumIntegers(myInts))
	fmt.Println("~int 匹配底层类型是 int 的所有类型（int、MyInt、type Foo int 等）")
	fmt.Println("没有 ~ 的话，约束 int | int64 不能匹配 MyInt，因为 MyInt 不等于 int")

	fmt.Printf("\nToString[int](%d) = %q\n", 42, ToString(42))
	fmt.Printf("ToString[MyString](%q) = %q\n", MyString("hello"), ToString(MyString("hello")))
	fmt.Println("联合类型约束可以混用基本类型和自定义类型（~int | MyString）")
}

// IntegerLike 约束底层类型是整数的所有类型。
type IntegerLike interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// SumIntegers 返回整数类切片的和。
func SumIntegers[T IntegerLike](s []T) T {
	var sum T
	for _, v := range s {
		sum += v
	}
	return sum
}

// StringLike 允许 int（转成字符串）或底层类型是 string 的类型。
type StringLike interface {
	int | ~string
}

// ToString 把 int 或 string-like 转成 string。
func ToString[T StringLike](v T) string {
	switch val := any(v).(type) {
	case int:
		return fmt.Sprintf("%d", val)
	case string:
		return val
	default:
		// 底层类型是 string 但不是 string 本身（如 type MyString string）
		// 需要先转成 string 类型
		return fmt.Sprint(v)
	}
}

// demoTypeInference 演示类型推导的规则。
func demoTypeInference() {
	ints := []int{1, 2, 3}

	// 显式指定类型参数
	r1 := Min[int](ints)
	fmt.Printf("显式：Min[int](...) = %d\n", r1)

	// 编译器从参数推导出 T = int
	r2 := Min(ints)
	fmt.Printf("推导：Min(...)       = %d\n", r2)

	// 返回值类型不参与推导，所以下面这行会报错：
	// var m float64 = Min(ints)  // 编译错误：推导出 T=int，但赋值给 float64

	fmt.Println("推导规则：")
	fmt.Println("  1. 编译器从实参的类型推导类型参数")
	fmt.Println("  2. 如果参数是泛型类型，会递归推导")
	fmt.Println("  3. 返回值类型不参与推导")
	fmt.Println("  4. 如果推导失败或有歧义，必须显式指定类型参数")
}

// demoGenericTypes 演示泛型结构体和方法。
func demoGenericTypes() {
	intStack := NewStack[int]()
	intStack.Push(10)
	intStack.Push(20)
	v1, ok1 := intStack.Pop()
	fmt.Printf("intStack.Pop() = %d, %t\n", v1, ok1)
	v2, ok2 := intStack.Pop()
	fmt.Printf("intStack.Pop() = %d, %t\n", v2, ok2)
	v3, ok3 := intStack.Pop()
	fmt.Printf("intStack.Pop() = %d, %t\n", v3, ok3)

	strStack := NewStack[string]()
	strStack.Push("go")
	strStack.Push("rust")
	v4, ok4 := strStack.Pop()
	fmt.Printf("strStack.Pop() = %q, %t\n", v4, ok4)

	fmt.Println("泛型类型的语法：type Stack[T any] struct { items []T }")
	fmt.Println("  实例化时必须指定类型参数：NewStack[int]()")
	fmt.Println("  方法接收者要带类型参数：func (s *Stack[T]) Push(v T)")
}

// Stack 是一个泛型栈。
type Stack[T any] struct {
	items []T
}

// NewStack 创建一个空栈。
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

// Push 压入一个元素。
func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

// Pop 弹出一个元素，如果栈空返回 false。
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	last := len(s.items) - 1
	v := s.items[last]
	s.items = s.items[:last]
	return v, true
}

// demoMethodSetInConstraints 演示约束中带方法的接口。
func demoMethodSetInConstraints() {
	people := []Person{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 35},
	}

	fmt.Printf("Oldest[Person](%v) = %+v\n", people, Oldest(people))
	fmt.Println("约束 HasAge 要求类型参数必须有 GetAge() int 方法")
	fmt.Println("  这样就可以在函数体里调用 v.GetAge()")
}

// HasAge 约束类型参数必须实现 GetAge 方法。
type HasAge interface {
	GetAge() int
}

// Oldest 返回年龄最大的元素。
func Oldest[T HasAge](s []T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	oldest := s[0]
	for _, v := range s[1:] {
		if v.GetAge() > oldest.GetAge() {
			oldest = v
		}
	}
	return oldest
}

// GetAge 实现 HasAge 接口。
func (p Person) GetAge() int {
	return p.Age
}

type Person struct {
	Name string
	Age  int
}

// demoStdlibGenerics 演示标准库中的泛型函数。
func demoStdlibGenerics() {
	ints := []int{3, 1, 4, 1, 5, 9}
	strs := []string{"go", "rust", "python", "java"}

	fmt.Printf("原始：ints = %v\n", ints)
	slices.Sort(ints)
	fmt.Printf("slices.Sort 后：%v\n", ints)
	fmt.Printf("slices.Max(ints) = %d\n", slices.Max(ints))
	fmt.Printf("slices.Contains(ints, 5) = %t\n", slices.Contains(ints, 5))

	fmt.Printf("\n原始：strs = %v\n", strs)
	slices.SortFunc(strs, func(a, b string) int {
		return cmp.Compare(len(a), len(b))
	})
	fmt.Printf("按长度排序：%v\n", strs)

	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 2, "c": 3}
	fmt.Printf("\nm1 = %v，m2 = %v\n", m1, m2)
	fmt.Printf("maps.Equal(m1, m2) = %t\n", maps.Equal(m1, m2))
	m3 := maps.Clone(m1)
	fmt.Printf("maps.Clone(m1) = %v\n", m3)

	fmt.Println("\ncmp.Or(0, 42, 100) =", cmp.Or(0, 42, 100), "（返回第一个非零值）")
	fmt.Println("cmp.Compare(10, 20) =", cmp.Compare(10, 20), "（-1 / 0 / +1）")

	fmt.Println("\n标准库泛型总结：")
	fmt.Println("  slices: Sort / SortFunc / Contains / Index / Max / Min / Clone / Equal / Delete ...")
	fmt.Println("  maps: Clone / Copy / Equal / DeleteFunc ...")
	fmt.Println("  cmp: Compare / Or / Less")
}

// demoGenericsLimitations 演示泛型的限制。
func demoGenericsLimitations() {
	fmt.Println("泛型的限制 1：方法不能有类型参数")
	fmt.Println("  type Foo struct{}")
	fmt.Println("  func (f Foo) Bar[T any](v T) {} // 编译错误")
	fmt.Println("  原因：方法调用的语法是 f.Bar()，没地方放类型参数")
	fmt.Println("  解法：把类型参数移到类型上 type Foo[T any]，或用顶层泛型函数")

	fmt.Println("\n泛型的限制 2：不能在约束中访问类型参数的字段")
	fmt.Println("  type HasName interface { Name string } // 编译错误")
	fmt.Println("  只能约束方法，不能约束字段")
	fmt.Println("  解法：在约束里加 GetName() 方法")

	fmt.Println("\n泛型的限制 3：不能对类型参数做类型断言或类型 switch（直接）")
	fmt.Println("  func F[T any](v T) { switch v.(type) {} } // 编译错误")
	fmt.Println("  解法：先转成 any 再断言：switch any(v).(type)")
}

// demoGenericsVsInterfaces 对比泛型和接口的适用场景。
func demoGenericsVsInterfaces() {
	fmt.Println("什么时候用泛型，什么时候用接口？")
	fmt.Println("\n用泛型的场景：")
	fmt.Println("  1. 容器类型（Stack / Queue / Cache / Map）")
	fmt.Println("  2. 算法与数据结构（Min / Max / Sort / BinarySearch）")
	fmt.Println("  3. 需要保留具体类型（避免 interface{} 装箱）")
	fmt.Println("  4. 编译期类型安全 + 零开销抽象")

	fmt.Println("\n用接口的场景：")
	fmt.Println("  1. 运行时多态（不同类型的集合、插件系统）")
	fmt.Println("  2. 依赖注入与可测试性（Repository / Logger / Client）")
	fmt.Println("  3. 标准库协议（io.Reader / io.Writer / error / fmt.Stringer）")
	fmt.Println("  4. 接口比泛型更简洁时（不需要类型参数列表）")

	fmt.Println("\n经验法则：")
	fmt.Println("  - 泛型擅长「对不同类型做同样的事」")
	fmt.Println("  - 接口擅长「对同一类型做不同的事」")
	fmt.Println("  - 不确定时优先接口，只有遇到重复代码或装箱开销才考虑泛型")
}
