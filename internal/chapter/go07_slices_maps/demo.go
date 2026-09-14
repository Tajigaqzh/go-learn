// Package go07_slices_maps 演示 Go 的数组、切片与映射。
package go07_slices_maps

import "fmt"

// Demo 运行第 7 章的所有示例。
func Demo() {
	fmt.Println("========== go07_slices_maps: 数组、切片与映射 ==========")

	fmt.Println("\n--- 1. 数组是值类型 ---")
	demoArrayValueType()

	fmt.Println("\n--- 2. 切片的三要素：指针、长度、容量 ---")
	demoSliceBasics()

	fmt.Println("\n--- 3. append 与扩容 ---")
	demoAppendAndGrow()

	fmt.Println("\n--- 4. 切片共享底层数组的坑 ---")
	demoSliceSharing()

	fmt.Println("\n--- 5. copy 与安全复制 ---")
	demoCopy()

	fmt.Println("\n--- 6. 三下标切片 ---")
	demoFullSliceExpression()

	fmt.Println("\n--- 7. nil 切片与空切片 ---")
	demoNilVsEmptySlice()

	fmt.Println("\n--- 8. 内存滞留问题 ---")
	demoMemoryRetention()

	fmt.Println("\n--- 9. map 基本操作 ---")
	demoMapBasics()

	fmt.Println("\n--- 10. map 遍历顺序随机 ---")
	demoMapRandomOrder()

	fmt.Println("\n--- 11. map 的零值与 nil map ---")
	demoNilMap()

	fmt.Println("\n--- 12. map key 约束 ---")
	demoMapKeyConstraints()

	fmt.Println("\n--- 13. 并发写 map 会 panic ---")
	demoConcurrentMapPanic()

	fmt.Println("\n========== 数组、切片与映射演示结束 ==========")
}

// demoArrayValueType 演示数组是值类型
func demoArrayValueType() {
	// 数组声明时必须指定长度
	var arr1 [3]int
	fmt.Printf("arr1（零值）: %v\n", arr1)

	// 数组字面量初始化
	arr2 := [3]int{10, 20, 30}
	fmt.Printf("arr2: %v\n", arr2)

	// 数组赋值是完整复制
	arr3 := arr2
	arr3[0] = 999
	fmt.Printf("修改 arr3[0] 后：arr2=%v, arr3=%v\n", arr2, arr3)

	// 数组作为函数参数也是复制
	modifyArray(arr2)
	fmt.Printf("调用 modifyArray 后，arr2 不变: %v\n", arr2)
}

func modifyArray(arr [3]int) {
	arr[0] = 111
	fmt.Printf("  函数内 arr: %v\n", arr)
}

// demoSliceBasics 演示切片的三要素
func demoSliceBasics() {
	// 切片字面量
	s1 := []int{1, 2, 3, 4, 5}
	fmt.Printf("s1: len=%d, cap=%d, %v\n", len(s1), cap(s1), s1)

	// 从数组或切片生成子切片
	s2 := s1[1:4] // [2, 3, 4]
	fmt.Printf("s2 := s1[1:4]: len=%d, cap=%d, %v\n", len(s2), cap(s2), s2)
	// cap(s2) = cap(s1) - 起始索引 = 5 - 1 = 4

	// make 创建切片
	s3 := make([]int, 3, 5) // len=3, cap=5
	fmt.Printf("s3 := make([]int, 3, 5): len=%d, cap=%d, %v\n", len(s3), cap(s3), s3)
}

// demoAppendAndGrow 演示 append 与扩容
func demoAppendAndGrow() {
	s := make([]int, 0, 2)
	fmt.Printf("初始: len=%d, cap=%d\n", len(s), cap(s))

	s = append(s, 1)
	fmt.Printf("append(1): len=%d, cap=%d\n", len(s), cap(s))

	s = append(s, 2)
	fmt.Printf("append(2): len=%d, cap=%d\n", len(s), cap(s))

	// 容量不足时扩容
	s = append(s, 3)
	fmt.Printf("append(3) 触发扩容: len=%d, cap=%d\n", len(s), cap(s))

	// 批量 append
	s = append(s, 4, 5, 6)
	fmt.Printf("append(4,5,6): len=%d, cap=%d, %v\n", len(s), cap(s), s)
}

// demoSliceSharing 演示切片共享底层数组的坑
func demoSliceSharing() {
	original := []int{1, 2, 3, 4, 5}
	sub := original[1:3] // [2, 3]
	fmt.Printf("original: %v\n", original)
	fmt.Printf("sub: %v\n", sub)

	// 修改子切片会影响原切片
	sub[0] = 999
	fmt.Printf("修改 sub[0]=999 后：\n")
	fmt.Printf("  original: %v\n", original)
	fmt.Printf("  sub: %v\n", sub)

	// append 可能触发扩容，此时不再共享
	sub = append(sub, 100, 200, 300)
	sub[1] = 888
	fmt.Printf("sub append 并修改后：\n")
	fmt.Printf("  original: %v（未受影响）\n", original)
	fmt.Printf("  sub: %v\n", sub)
}

// demoCopy 演示 copy 与安全复制
func demoCopy() {
	src := []int{10, 20, 30, 40, 50}
	dst := make([]int, 3)

	// copy 返回实际复制的元素个数（取 len(dst) 和 len(src) 的最小值）
	n := copy(dst, src)
	fmt.Printf("copy %d 个元素: dst=%v\n", n, dst)

	// 修改 dst 不影响 src
	dst[0] = 999
	fmt.Printf("修改 dst[0] 后: src=%v, dst=%v\n", src, dst)

	// 完整复制
	full := make([]int, len(src))
	copy(full, src)
	fmt.Printf("完整复制: full=%v\n", full)
}

// demoFullSliceExpression 演示三下标切片
func demoFullSliceExpression() {
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	fmt.Printf("原始切片: len=%d, cap=%d\n", len(s), cap(s))

	// 普通切片 s[2:5] -> [2, 3, 4]，cap = cap(s) - 2 = 8
	sub1 := s[2:5]
	fmt.Printf("sub1 := s[2:5]: len=%d, cap=%d, %v\n", len(sub1), cap(sub1), sub1)

	// 三下标切片 s[2:5:6] -> [2, 3, 4]，cap = 6 - 2 = 4
	sub2 := s[2:5:6]
	fmt.Printf("sub2 := s[2:5:6]: len=%d, cap=%d, %v\n", len(sub2), cap(sub2), sub2)

	// sub2 的容量受限，append 更容易触发扩容，减少共享风险
	sub2 = append(sub2, 99)
	fmt.Printf("sub2 append 后: len=%d, cap=%d, %v\n", len(sub2), cap(sub2), sub2)
	fmt.Printf("原始 s: %v（未受影响）\n", s)
}

// demoNilVsEmptySlice 演示 nil 切片与空切片
func demoNilVsEmptySlice() {
	var nilSlice []int
	emptySlice := []int{}
	madeSlice := make([]int, 0)

	fmt.Printf("nilSlice == nil: %v, len=%d, cap=%d\n", nilSlice == nil, len(nilSlice), cap(nilSlice))
	fmt.Printf("emptySlice == nil: %v, len=%d, cap=%d\n", emptySlice == nil, len(emptySlice), cap(emptySlice))
	fmt.Printf("madeSlice == nil: %v, len=%d, cap=%d\n", madeSlice == nil, len(madeSlice), cap(madeSlice))

	// 三者都可以安全地 append
	nilSlice = append(nilSlice, 1)
	emptySlice = append(emptySlice, 2)
	madeSlice = append(madeSlice, 3)
	fmt.Printf("append 后: nilSlice=%v, emptySlice=%v, madeSlice=%v\n", nilSlice, emptySlice, madeSlice)
}

// demoMemoryRetention 演示内存滞留问题
func demoMemoryRetention() {
	// 假设从一个大切片中取一小段
	large := make([]byte, 1000)
	for i := range large {
		large[i] = byte(i % 256)
	}

	// 只需要前 10 个字节
	small := large[:10]
	fmt.Printf("small: len=%d, cap=%d\n", len(small), cap(small))
	fmt.Println("注意：small 的底层数组仍然是 1000 字节，造成内存滞留")

	// 正确做法：用 copy 创建独立副本
	smallCopy := make([]byte, 10)
	copy(smallCopy, large[:10])
	fmt.Printf("smallCopy: len=%d, cap=%d\n", len(smallCopy), cap(smallCopy))
	fmt.Println("smallCopy 不再引用大数组，large 可以被 GC 回收")
}

// demoMapBasics 演示 map 基本操作
func demoMapBasics() {
	// map 字面量初始化
	ages := map[string]int{
		"Alice": 25,
		"Bob":   30,
	}
	fmt.Printf("初始 map: %v\n", ages)

	// 增
	ages["Carol"] = 28
	fmt.Printf("增加 Carol: %v\n", ages)

	// 改
	ages["Alice"] = 26
	fmt.Printf("修改 Alice: %v\n", ages)

	// 查（带 ok 形式）
	if age, ok := ages["Bob"]; ok {
		fmt.Printf("Bob 的年龄: %d\n", age)
	}

	// 查不存在的键
	age := ages["David"] // 返回零值
	fmt.Printf("David 的年龄（不存在）: %d\n", age)

	// 删
	delete(ages, "Bob")
	fmt.Printf("删除 Bob: %v\n", ages)

	// 删除不存在的键是安全的
	delete(ages, "NonExist")

	// len 返回键值对数量
	fmt.Printf("map 长度: %d\n", len(ages))
}

// demoMapRandomOrder 演示 map 遍历顺序随机
func demoMapRandomOrder() {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}

	fmt.Println("第一次遍历:")
	for k, v := range m {
		fmt.Printf("  %s: %d\n", k, v)
	}

	fmt.Println("第二次遍历（顺序可能不同）:")
	for k, v := range m {
		fmt.Printf("  %s: %d\n", k, v)
	}

	fmt.Println("注意：每次运行或遍历，顺序都可能不同")
}

// demoNilMap 演示 nil map
func demoNilMap() {
	var m map[string]int
	fmt.Printf("nil map: %v, len=%d, m==nil: %v\n", m, len(m), m == nil)

	// 读 nil map 返回零值，不会 panic
	v := m["key"]
	fmt.Printf("读 nil map: %d\n", v)

	// 查询 nil map
	if _, ok := m["key"]; ok {
		fmt.Println("找到了")
	} else {
		fmt.Println("nil map 查询返回 ok=false")
	}

	// 遍历 nil map 安全
	for k, v := range m {
		fmt.Printf("%s: %d\n", k, v) // 不会执行
	}
	fmt.Println("遍历 nil map 安全（0 次迭代）")

	// 写 nil map 会 panic（这里用注释演示）
	// m["key"] = 100 // panic: assignment to entry in nil map

	// 正确做法：初始化
	m = make(map[string]int)
	m["key"] = 100
	fmt.Printf("初始化后写入: %v\n", m)
}

// demoMapKeyConstraints 演示 map key 约束
func demoMapKeyConstraints() {
	// 可比较类型可以作为 key
	m1 := make(map[int]string)
	m1[42] = "answer"

	m2 := make(map[string]int)
	m2["key"] = 1

	// 结构体作为 key（如果所有字段都可比较）
	type Point struct{ X, Y int }
	m3 := make(map[Point]string)
	m3[Point{1, 2}] = "origin nearby"
	fmt.Printf("Point 作为 key: %v\n", m3)

	// 数组可以作为 key
	m4 := make(map[[2]int]string)
	m4[[2]int{1, 2}] = "array key"
	fmt.Printf("数组作为 key: %v\n", m4)

	// 切片、map、函数不能作为 key（编译错误）
	// var m5 map[[]int]string // 编译错误：invalid map key type []int
	// var m6 map[map[int]int]string // 编译错误
	// var m7 map[func()]string // 编译错误

	fmt.Println("只有可比较类型可以作为 map key")
}

// demoConcurrentMapPanic 演示并发写 map 会 panic
func demoConcurrentMapPanic() {
	// 注意：这个示例不实际触发 panic，只演示概念
	fmt.Println("并发读写 map 是不安全的，会导致 panic 或数据竞争")
	fmt.Println("解决方案：")
	fmt.Println("  1. 使用 sync.Mutex 保护 map")
	fmt.Println("  2. 使用 sync.Map（适合读多写少场景）")
	fmt.Println("  3. 使用 channel 串行化访问")

	// 正确示例：使用 make 初始化
	m := make(map[string]int)
	m["safe"] = 1
	fmt.Printf("单协程访问是安全的: %v\n", m)
}
