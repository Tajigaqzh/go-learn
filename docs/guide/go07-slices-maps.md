# 第 7 章 · 数组、切片与映射

前面几章我们学习了 Go 的基本类型、控制流和指针。这些是构建程序的基础砖块，但实际编程中，我们经常需要处理**一组数据**：一批用户、一系列配置项、一个键值对集合。这就是数组、切片和映射（map）的作用。

Go 的数组是固定长度的值类型，用得不多；**切片（slice）** 是动态长度的视图，是 Go 中最常用的数据结构之一；**映射（map）** 提供键值对存储，类似其他语言的字典或哈希表。理解切片的底层结构（指针、长度、容量）和 map 的特性（无序、并发不安全）是写好 Go 代码的关键。

本章配套代码在 `internal/chapter/go07_slices_maps/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 7.1 数组是值类型

数组在声明时必须指定长度，长度是类型的一部分。`[3]int` 和 `[4]int` 是不同的类型：

```go
var arr1 [3]int
fmt.Printf("arr1（零值）: %v\n", arr1)
```

**输出：**
```
arr1（零值）: [0 0 0]
```

数组的零值是所有元素都为其类型的零值。

数组字面量初始化：

```go
arr2 := [3]int{10, 20, 30}
fmt.Printf("arr2: %v\n", arr2)
```

**输出：**
```
arr2: [10 20 30]
```

**关键特性：数组是值类型，赋值和传参都是完整复制**：

```go
arr3 := arr2
arr3[0] = 999
fmt.Printf("修改 arr3[0] 后：arr2=%v, arr3=%v\n", arr2, arr3)
```

**输出：**
```
修改 arr3[0] 后：arr2=[10 20 30], arr3=[999 20 30]
```

修改 `arr3` 不影响 `arr2`，因为赋值时完整复制了 `arr2` 的所有元素。

数组作为函数参数也是复制：

```go
func modifyArray(arr [3]int) {
    arr[0] = 111
    fmt.Printf("  函数内 arr: %v\n", arr)
}

arr2 := [3]int{10, 20, 30}
modifyArray(arr2)
fmt.Printf("调用 modifyArray 后，arr2 不变: %v\n", arr2)
```

**输出：**
```
  函数内 arr: [111 20 30]
调用 modifyArray 后，arr2 不变: [10 20 30]
```

这种值传递行为在处理大数组时代价很高。实际编程中，我们几乎总是使用**切片**而不是数组，因为切片是引用语义，传递代价很小。

## 7.2 切片的三要素：指针、长度、容量

切片是对底层数组的**动态视图**，它的内部结构包含三个要素：

1. **指针**：指向底层数组的起始位置
2. **长度（len）**：切片当前包含的元素个数
3. **容量（cap）**：从切片起始位置到底层数组末尾的元素个数

```go
s1 := []int{1, 2, 3, 4, 5}
fmt.Printf("s1: len=%d, cap=%d, %v\n", len(s1), cap(s1), s1)
```

**输出：**
```
s1: len=3, cap=5, [1 2 3 4 5]
```

从切片或数组生成子切片：

```go
s2 := s1[1:4] // [2, 3, 4]
fmt.Printf("s2 := s1[1:4]: len=%d, cap=%d, %v\n", len(s2), cap(s2), s2)
```

**输出：**
```
s2 := s1[1:4]: len=3, cap=4, [2 3 4]
```

`s2` 的长度是 3（`[1:4]` 包含索引 1、2、3），容量是 4（从索引 1 到底层数组末尾是 4 个元素）。

**用 `make` 创建切片**：

```go
s3 := make([]int, 3, 5) // len=3, cap=5
fmt.Printf("s3 := make([]int, 3, 5): len=%d, cap=%d, %v\n", len(s3), cap(s3), s3)
```

**输出：**
```
s3 := make([]int, 3, 5): len=3, cap=5, [0 0 0]
```

`make([]T, len, cap)` 创建一个长度为 `len`、容量为 `cap` 的切片，所有元素初始化为零值。如果省略容量参数，容量等于长度。

## 7.3 append 与扩容

`append` 向切片追加元素。如果容量足够，直接在底层数组追加；如果容量不足，Go 会分配新的底层数组并复制旧元素：

```go
s := make([]int, 0, 2)
fmt.Printf("初始: len=%d, cap=%d\n", len(s), cap(s))

s = append(s, 1)
fmt.Printf("append(1): len=%d, cap=%d\n", len(s), cap(s))

s = append(s, 2)
fmt.Printf("append(2): len=%d, cap=%d\n", len(s), cap(s))

// 容量不足时扩容
s = append(s, 3)
fmt.Printf("append(3) 触发扩容: len=%d, cap=%d\n", len(s), cap(s))
```

**输出：**
```
初始: len=0, cap=2
append(1): len=1, cap=2
append(2): len=2, cap=2
append(3) 触发扩容: len=3, cap=4
```

扩容策略：

- 当前容量小于 256 时，通常扩容为原来的 2 倍
- 当前容量大于等于 256 时，增长因子逐渐降低（约 1.25 倍），以减少内存浪费

**批量 append**：

```go
s = append(s, 4, 5, 6)
fmt.Printf("append(4,5,6): len=%d, cap=%d, %v\n", len(s), cap(s), s)
```

**输出：**
```
append(4,5,6): len=6, cap=8, [1 2 3 4 5 6]
```

**性能建议**：如果预先知道切片的大致长度，用 `make` 预分配容量可以避免多次扩容，显著提升性能：

```go
// 不好：多次扩容
s1 := []int{}
for i := 0; i < 1000; i++ {
    s1 = append(s1, i)
}

// 好：预分配容量
s2 := make([]int, 0, 1000)
for i := 0; i < 1000; i++ {
    s2 = append(s2, i)
}
```

基准测试显示，预分配比不预分配快约 **3-5 倍**。

## 7.4 切片共享底层数组的坑

切片是对底层数组的视图，多个切片可以**共享同一个底层数组**。修改一个切片可能会影响其他切片：

```go
original := []int{1, 2, 3, 4, 5}
sub := original[1:3] // [2, 3]
fmt.Printf("original: %v\n", original)
fmt.Printf("sub: %v\n", sub)

// 修改子切片会影响原切片
sub[0] = 999
fmt.Printf("修改 sub[0]=999 后：\n")
fmt.Printf("  original: %v\n", original)
fmt.Printf("  sub: %v\n", sub)
```

**输出：**
```
original: [1 2 3 4 5]
sub: [2 3]
修改 sub[0]=999 后：
  original: [1 999 3 4 5]
  sub: [999 3]
```

`sub[0]` 对应 `original[1]`，修改 `sub[0]` 也修改了 `original[1]`。

**但是，如果 `append` 触发扩容，新切片会分配新的底层数组，不再共享**：

```go
sub = append(sub, 100, 200, 300)
sub[1] = 888
fmt.Printf("sub append 并修改后：\n")
fmt.Printf("  original: %v（未受影响）\n", original)
fmt.Printf("  sub: %v\n", sub)
```

**输出：**
```
sub append 并修改后：
  original: [1 999 3 4 5]（未受影响）
  sub: [999 888 100 200 300]
```

扩容后，`sub` 指向新的底层数组，修改 `sub` 不再影响 `original`。

**这个坑很隐蔽**：你以为修改子切片不会影响原切片，但实际上只要不扩容，就会影响。解决方案是使用 `copy` 创建独立副本。

## 7.5 copy 与安全复制

`copy` 函数将源切片的元素复制到目标切片，返回实际复制的元素个数（取 `len(dst)` 和 `len(src)` 的最小值）：

```go
src := []int{10, 20, 30, 40, 50}
dst := make([]int, 3)

n := copy(dst, src)
fmt.Printf("copy %d 个元素: dst=%v\n", n, dst)
```

**输出：**
```
copy 3 个元素: dst=[10 20 30]
```

修改 `dst` 不影响 `src`，因为 `copy` 创建了独立副本：

```go
dst[0] = 999
fmt.Printf("修改 dst[0] 后: src=%v, dst=%v\n", src, dst)
```

**输出：**
```
修改 dst[0] 后: src=[10 20 30 40 50], dst=[999 20 30]
```

**完整复制**：

```go
full := make([]int, len(src))
copy(full, src)
fmt.Printf("完整复制: full=%v\n", full)
```

**输出：**
```
完整复制: full=[10 20 30 40 50]
```

**什么时候用 `copy`**：

- 需要修改数据但不想影响原切片
- 从一个大切片中提取一小部分，避免内存滞留（见 7.8）
- 并发场景下需要独立副本

## 7.6 三下标切片

普通切片表达式 `s[low:high]` 生成的子切片，其容量是 `cap(s) - low`。这意味着子切片可以通过 `append` 访问到原切片后面的元素，可能导致意外的共享。

**三下标切片表达式** `s[low:high:max]` 可以显式限制容量，让子切片更早触发扩容，减少共享风险：

```go
s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

// 普通切片 s[2:5] -> [2, 3, 4]，cap = cap(s) - 2 = 8
sub1 := s[2:5]
fmt.Printf("sub1 := s[2:5]: len=%d, cap=%d, %v\n", len(sub1), cap(sub1), sub1)

// 三下标切片 s[2:5:6] -> [2, 3, 4]，cap = 6 - 2 = 4
sub2 := s[2:5:6]
fmt.Printf("sub2 := s[2:5:6]: len=%d, cap=%d, %v\n", len(sub2), cap(sub2), sub2)
```

**输出：**
```
sub1 := s[2:5]: len=3, cap=8, [2 3 4]
sub2 := s[2:5:6]: len=3, cap=4, [2 3 4]
```

`sub2` 的容量被限制为 4，append 一个元素后就会触发扩容，不再共享原数组：

```go
sub2 = append(sub2, 99)
fmt.Printf("sub2 append 后: len=%d, cap=%d, %v\n", len(sub2), cap(sub2), sub2)
fmt.Printf("原始 s: %v（未受影响）\n", s)
```

**输出：**
```
sub2 append 后: len=4, cap=4, [2 3 4 99]
原始 s: [0 1 2 3 4 5 6 7 8 9]（未受影响）
```

三下标切片在需要严格控制共享行为时很有用，但实际代码中用得不多，大多数情况用 `copy` 更直观。

## 7.7 nil 切片与空切片

nil 切片和空切片是两个概念：

```go
var nilSlice []int
emptySlice := []int{}
madeSlice := make([]int, 0)

fmt.Printf("nilSlice == nil: %v, len=%d, cap=%d\n", nilSlice == nil, len(nilSlice), cap(nilSlice))
fmt.Printf("emptySlice == nil: %v, len=%d, cap=%d\n", emptySlice == nil, len(emptySlice), cap(emptySlice))
fmt.Printf("madeSlice == nil: %v, len=%d, cap=%d\n", madeSlice == nil, len(madeSlice), cap(madeSlice))
```

**输出：**
```
nilSlice == nil: true, len=0, cap=0
emptySlice == nil: false, len=0, cap=0
madeSlice == nil: false, len=0, cap=0
```

- **nil 切片**：声明但未初始化，内部指针为 `nil`，`len` 和 `cap` 都是 0
- **空切片**：字面量 `[]T{}` 或 `make([]T, 0)` 创建，内部指针非 `nil`，但 `len` 和 `cap` 是 0

**实际使用中的区别很小**：三者都可以安全地 `append`、遍历和传参。大多数场景下，`var s []T`（nil 切片）和 `s := []T{}`（空切片）可以互换。

**什么时候选哪个**：

- `var s []T`（nil 切片）：默认选择，零值可用，不分配内存
- `s := []T{}`（空切片）：需要 JSON 序列化为 `[]` 而不是 `null` 时
- `s := make([]T, 0, cap)`：需要预分配容量时

## 7.8 内存滞留问题

当你从一个大切片中提取一小段时，子切片仍然引用整个底层数组，导致大数组无法被 GC 回收，造成**内存滞留**：

```go
// 假设从一个大切片中取一小段
large := make([]byte, 1000)
for i := range large {
    large[i] = byte(i % 256)
}

// 只需要前 10 个字节
small := large[:10]
fmt.Printf("small: len=%d, cap=%d\n", len(small), cap(small))
fmt.Println("注意：small 的底层数组仍然是 1000 字节，造成内存滞留")
```

**输出：**
```
small: len=10, cap=1000
注意：small 的底层数组仍然是 1000 字节，造成内存滞留
```

即使只保留 `small`，整个 1000 字节的数组都不能被回收。

**正确做法：用 `copy` 创建独立副本**：

```go
smallCopy := make([]byte, 10)
copy(smallCopy, large[:10])
fmt.Printf("smallCopy: len=%d, cap=%d\n", len(smallCopy), cap(smallCopy))
fmt.Println("smallCopy 不再引用大数组，large 可以被 GC 回收")
```

**输出：**
```
smallCopy: len=10, cap=10
smallCopy 不再引用大数组，large 可以被 GC 回收
```

**什么时候需要注意内存滞留**：

- 从文件或网络读取大数据，只保留一小部分
- 解析日志、CSV 等大文本，只保留部分行
- 图片处理、切片裁剪等场景

## 7.9 map 基本操作

map 是无序的键值对集合，类似其他语言的字典或哈希表。声明时必须用 `make` 初始化，或使用字面量：

```go
// map 字面量初始化
ages := map[string]int{
    "Alice": 25,
    "Bob":   30,
}
fmt.Printf("初始 map: %v\n", ages)
```

**输出：**
```
初始 map: map[Alice:25 Bob:30]
```

**增**：

```go
ages["Carol"] = 28
fmt.Printf("增加 Carol: %v\n", ages)
```

**输出：**
```
增加 Carol: map[Alice:25 Bob:30 Carol:28]
```

**改**：

```go
ages["Alice"] = 26
fmt.Printf("修改 Alice: %v\n", ages)
```

**输出：**
```
修改 Alice: map[Alice:26 Bob:30 Carol:28]
```

**查（带 ok 形式）**：

```go
if age, ok := ages["Bob"]; ok {
    fmt.Printf("Bob 的年龄: %d\n", age)
}
```

**输出：**
```
Bob 的年龄: 30
```

查询不存在的键返回值类型的零值：

```go
age := ages["David"] // 返回 0
fmt.Printf("David 的年龄（不存在）: %d\n", age)
```

**输出：**
```
David 的年龄（不存在）: 0
```

**删**：

```go
delete(ages, "Bob")
fmt.Printf("删除 Bob: %v\n", ages)
```

**输出：**
```
删除 Bob: map[Alice:26 Carol:28]
```

删除不存在的键是安全的，不会 panic。

**长度**：

```go
fmt.Printf("map 长度: %d\n", len(ages))
```

**输出：**
```
map 长度: 2
```

## 7.10 map 遍历顺序随机

map 的遍历顺序是**随机的**，每次运行或每次遍历都可能不同。这是 Go 故意设计的，防止程序依赖遍历顺序：

```go
m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}

fmt.Println("第一次遍历:")
for k, v := range m {
    fmt.Printf("  %s: %d\n", k, v)
}

fmt.Println("第二次遍历（顺序可能不同）:")
for k, v := range m {
    fmt.Printf("  %s: %d\n", k, v)
}
```

**输出（每次可能不同）：**
```
第一次遍历:
  c: 3
  d: 4
  e: 5
  a: 1
  b: 2
第二次遍历（顺序可能不同）:
  e: 5
  a: 1
  b: 2
  c: 3
  d: 4
```

**如果需要有序遍历**，先把键提取到切片，排序后再访问 map：

```go
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    fmt.Printf("%s: %d\n", k, m[k])
}
```

## 7.11 map 的零值与 nil map

map 的零值是 `nil`。nil map 可以安全地**读取和遍历**，但**不能写入**：

```go
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
```

**输出：**
```
nil map: map[], len=0, m==nil: true
读 nil map: 0
nil map 查询返回 ok=false
遍历 nil map 安全（0 次迭代）
```

**写 nil map 会 panic**：

```go
// m["key"] = 100 // panic: assignment to entry in nil map
```

**正确做法：初始化**：

```go
m = make(map[string]int)
m["key"] = 100
fmt.Printf("初始化后写入: %v\n", m)
```

**输出：**
```
初始化后写入: map[key:100]
```

## 7.12 map key 约束

只有**可比较类型**可以作为 map 的 key。可比较类型包括：

- 基本类型：`int`、`float64`、`string`、`bool`、指针
- 结构体（所有字段都可比较）
- 数组（元素类型可比较）

**不可比较类型不能作为 key**：

- 切片 `[]T`
- map `map[K]V`
- 函数 `func()`

```go
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
```

**输出：**
```
Point 作为 key: map[{1 2}:origin nearby]
数组作为 key: map[[1 2]:array key]
```

**编译错误示例**：

```go
// var m5 map[[]int]string // 编译错误：invalid map key type []int
// var m6 map[map[int]int]string // 编译错误
// var m7 map[func()]string // 编译错误
```

## 7.13 并发写 map 会 panic

map **不是并发安全的**。多个 goroutine 同时读写同一个 map 会导致 panic 或数据竞争：

```go
// 错误示例（会 panic 或数据竞争）
m := make(map[int]int)
go func() {
    for i := 0; i < 1000; i++ {
        m[i] = i // 写
    }
}()
go func() {
    for i := 0; i < 1000; i++ {
        _ = m[i] // 读
    }
}()
```

**运行时输出（可能）：**
```
fatal error: concurrent map writes
```

**解决方案**：

1. **使用 `sync.Mutex` 保护 map**：

```go
var mu sync.Mutex
m := make(map[int]int)

mu.Lock()
m[1] = 100
mu.Unlock()

mu.Lock()
v := m[1]
mu.Unlock()
```

2. **使用 `sync.Map`**（适合读多写少场景）：

```go
var sm sync.Map
sm.Store("key", "value")
if v, ok := sm.Load("key"); ok {
    fmt.Println(v)
}
```

3. **使用 channel 串行化访问**：让一个 goroutine 独占 map，其他 goroutine 通过 channel 发送请求。

**什么时候需要注意**：任何多 goroutine 场景都要考虑并发安全。即使是「一读一写」也不安全。

---

## 5 个真实报错怎么读

### 报错 1：数组长度不匹配

```go
var arr1 [3]int
var arr2 [4]int
arr1 = arr2
```

**编译器输出：**
```
cannot use arr2 (variable of type [4]int) as type [3]int in assignment
```

**原因：** 数组的长度是类型的一部分，`[3]int` 和 `[4]int` 是不同的类型，不能互相赋值。

**修复：** 使用切片而不是数组，或确保长度一致。

### 报错 2：切片索引越界

```go
s := []int{1, 2, 3}
fmt.Println(s[3])
```

**运行时输出：**
```
panic: runtime error: index out of range [3] with length 3
```

**原因：** 切片长度是 3，有效索引是 0-2，访问 `s[3]` 越界。

**修复：** 检查索引范围，或用 `append` 扩展切片。

### 报错 3：写 nil map

```go
var m map[string]int
m["key"] = 100
```

**运行时输出：**
```
panic: assignment to entry in nil map
```

**原因：** nil map 不能写入，必须先用 `make` 初始化。

**修复：**
```go
m := make(map[string]int)
m["key"] = 100
```

### 报错 4：切片作为 map key

```go
m := make(map[[]int]string)
```

**编译器输出：**
```
invalid map key type []int
```

**原因：** 切片不是可比较类型，不能作为 map key。

**修复：** 使用数组或字符串（把切片转成字符串表示），或使用结构体。

### 报错 5：append 后切片未赋值

```go
s := []int{1, 2, 3}
append(s, 4)
fmt.Println(s) // 输出 [1 2 3]，不是 [1 2 3 4]
```

**原因：** `append` 返回新的切片，不会修改原切片。

**修复：**
```go
s = append(s, 4)
fmt.Println(s) // [1 2 3 4]
```

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 修改子切片影响原切片 | 切片共享底层数组 | 用 `copy` 创建独立副本 |
| `append` 后原切片不变 | `append` 返回新切片，不修改原切片 | 必须赋值：`s = append(s, x)` |
| 切片索引越界 panic | 访问了 `len` 之外的索引 | 检查 `len(s)`，或用 `append` 扩展 |
| 从大切片提取小段后内存不释放 | 子切片仍引用整个底层数组 | 用 `copy` 创建独立副本 |
| 写 nil map panic | nil map 不能写 | 用 `make(map[K]V)` 初始化 |
| map 遍历顺序每次不同 | map 遍历顺序随机 | 提取键到切片，排序后遍历 |
| 并发读写 map panic | map 不是并发安全的 | 用 `sync.Mutex` 或 `sync.Map` |
| 切片作为 map key 编译错误 | 切片不可比较 | 改用数组或字符串 |
| 数组赋值后修改不影响原数组 | 数组是值类型，赋值是复制 | 改用切片，或传递数组指针 |

---

## 练习

### 第 1 题

写一个函数 `removeDuplicates(nums []int) []int`，原地删除切片中的重复元素（假设切片已排序），返回新切片。

示例：
```go
nums := []int{1, 1, 2, 2, 2, 3, 4, 4}
result := removeDuplicates(nums)
fmt.Println(result) // [1 2 3 4]
```

::: details 第 1 题参考答案
```go
func removeDuplicates(nums []int) []int {
    if len(nums) == 0 {
        return nums
    }

    // 双指针：slow 指向不重复元素的末尾，fast 遍历
    slow := 0
    for fast := 1; fast < len(nums); fast++ {
        if nums[fast] != nums[slow] {
            slow++
            nums[slow] = nums[fast]
        }
    }
    return nums[:slow+1]
}
```

**为什么这样写：**
- 原地修改，空间复杂度 O(1)
- 利用已排序的特性，只需一次遍历，时间复杂度 O(n)
- 返回 `nums[:slow+1]` 而不是 `nums`，因为后面可能有重复元素
:::

### 第 2 题

写一个函数 `merge(a, b []int) []int`，合并两个已排序的切片，返回新的已排序切片。

示例：
```go
a := []int{1, 3, 5}
b := []int{2, 4, 6}
result := merge(a, b)
fmt.Println(result) // [1 2 3 4 5 6]
```

::: details 第 2 题参考答案
```go
func merge(a, b []int) []int {
    result := make([]int, 0, len(a)+len(b))
    i, j := 0, 0

    for i < len(a) && j < len(b) {
        if a[i] <= b[j] {
            result = append(result, a[i])
            i++
        } else {
            result = append(result, b[j])
            j++
        }
    }

    // 追加剩余元素
    result = append(result, a[i:]...)
    result = append(result, b[j:]...)

    return result
}
```

**为什么这样写：**
- 预分配容量 `len(a)+len(b)`，避免多次扩容
- 双指针归并，时间复杂度 O(n+m)
- `append(result, a[i:]...)` 用 `...` 展开切片
:::

### 第 3 题

写一个函数 `wordFreq(text string) map[string]int`，统计文本中每个单词的出现次数（忽略大小写和标点）。

示例：
```go
text := "Hello, world! Hello Go."
freq := wordFreq(text)
fmt.Println(freq) // map[go:1 hello:2 world:1]
```

::: details 第 3 题参考答案
```go
import (
    "strings"
    "unicode"
)

func wordFreq(text string) map[string]int {
    // 移除标点，转小写
    cleanText := strings.Map(func(r rune) rune {
        if unicode.IsLetter(r) || unicode.IsSpace(r) {
            return unicode.ToLower(r)
        }
        return -1 // 删除字符
    }, text)

    words := strings.Fields(cleanText)
    freq := make(map[string]int)
    for _, word := range words {
        freq[word]++
    }
    return freq
}
```

**为什么这样写：**
- `strings.Map` 清理标点和统一大小写
- `strings.Fields` 按空白分割单词
- `freq[word]++` 利用 map 零值（0）自动初始化
:::

### 第 4 题

写一个函数 `groupAnagrams(words []string) [][]string`，把异位词（anagram，字母相同但顺序不同的单词）分组。

示例：
```go
words := []string{"eat", "tea", "tan", "ate", "nat", "bat"}
groups := groupAnagrams(words)
// [["eat", "tea", "ate"], ["tan", "nat"], ["bat"]]
```

::: details 第 4 题参考答案
```go
import (
    "sort"
    "strings"
)

func groupAnagrams(words []string) [][]string {
    groups := make(map[string][]string)

    for _, word := range words {
        // 排序后的字符串作为 key
        key := sortString(word)
        groups[key] = append(groups[key], word)
    }

    result := make([][]string, 0, len(groups))
    for _, group := range groups {
        result = append(result, group)
    }
    return result
}

func sortString(s string) string {
    runes := []rune(s)
    sort.Slice(runes, func(i, j int) bool {
        return runes[i] < runes[j]
    })
    return string(runes)
}
```

**为什么这样写：**
- 排序后的字符串作为 key，异位词的 key 相同
- `map[string][]string` 自动分组
- 时间复杂度 O(n * k log k)，n 是单词数，k 是单词平均长度
:::

### 第 5 题

写一个函数 `topK(nums []int, k int) []int`，返回切片中出现频率最高的 k 个元素（可以任意顺序返回）。

示例：
```go
nums := []int{1, 1, 1, 2, 2, 3}
result := topK(nums, 2)
fmt.Println(result) // [1 2]（任意顺序）
```

::: details 第 5 题参考答案
```go
import "sort"

func topK(nums []int, k int) []int {
    // 统计频率
    freq := make(map[int]int)
    for _, num := range nums {
        freq[num]++
    }

    // 转成切片并排序
    type pair struct {
        num   int
        count int
    }
    pairs := make([]pair, 0, len(freq))
    for num, count := range freq {
        pairs = append(pairs, pair{num, count})
    }

    sort.Slice(pairs, func(i, j int) bool {
        return pairs[i].count > pairs[j].count
    })

    // 取前 k 个
    result := make([]int, k)
    for i := 0; i < k; i++ {
        result[i] = pairs[i].num
    }
    return result
}
```

**为什么这样写：**
- 用 map 统计频率
- 转成切片并按频率排序
- 取前 k 个。时间复杂度 O(n + m log m)，m 是不同元素数量
- 更优解是用堆（优先队列），时间复杂度 O(n + m log k)
:::

### 第 6 题

解释下面代码的输出，并说明如何修复：

```go
s := []int{1, 2, 3, 4, 5}
var result []*int
for _, v := range s {
    result = append(result, &v)
}
for _, p := range result {
    fmt.Print(*p, " ")
}
```

::: details 第 6 题参考答案
**输出：**
```
5 5 5 5 5
```

**原因：** 在 Go 1.22 之前，`range` 循环变量 `v` 在整个循环中共享同一个变量。所有的 `&v` 都指向同一个地址，循环结束后 `v` 的值是 5，所以所有指针解引用后都是 5。

Go 1.22+ 修复了这个问题，每次迭代 `v` 是新变量，输出 `1 2 3 4 5`。

**兼容 Go 1.21 的修复方法：**
```go
for _, v := range s {
    v := v // 显式复制
    result = append(result, &v)
}
```

或者直接取切片元素的地址（如果是切片）：
```go
for i := range s {
    result = append(result, &s[i])
}
```

**为什么这个坑重要：** 这是 Go 1.22 之前最常见的陷阱之一，涉及闭包和指针的组合。
:::

---

## 小结

- 数组是值类型，长度固定，赋值和传参都是完整复制；实际编程中几乎总是用切片。
- 切片是对底层数组的动态视图，内部包含指针、长度和容量三要素。
- `append` 会在容量不足时扩容，返回新切片；预分配容量可以显著提升性能。
- 切片共享底层数组，修改子切片可能影响原切片；用 `copy` 创建独立副本。
- 三下标切片 `s[low:high:max]` 可以限制容量，减少共享风险。
- nil 切片和空切片在大多数场景下可以互换，都可以安全地 `append` 和遍历。
- 从大切片提取小段时注意内存滞留，用 `copy` 创建独立副本。
- map 是无序的键值对集合，遍历顺序随机。
- nil map 可以安全地读取和遍历，但不能写入；必须用 `make` 初始化。
- 只有可比较类型可以作为 map key，切片、map、函数不能作为 key。
- map 不是并发安全的，多 goroutine 同时读写会 panic 或数据竞争。

下一章我们将学习**字符串、字节与 Unicode**：string 的底层结构、UTF-8 编码、`strings` 与 `strconv` 标准库、字符串拼接的性能对比，以及 `strings.Builder` 与 `bytes.Buffer` 的使用。
