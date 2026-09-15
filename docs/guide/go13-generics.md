# 第 13 章：泛型

> **本章配套代码**：`internal/chapter/go13_generics/`  
> **运行方式**：`go run ./cmd/go-learn`

Go 1.18 引入了泛型（Generics），解决了长期以来"为不同类型写重复代码"的痛点。本章从真实的重复代码出发，系统讲解类型参数、类型约束、泛型类型、标准库泛型工具，以及泛型与接口的取舍。

---

## 13.1 为什么需要泛型：从重复代码说起

泛型出现之前，要写一个"找切片最小值"的函数，必须为每种类型写一遍：

```go
func MinInt(s []int) int {
    if len(s) == 0 {
        panic("empty slice")
    }
    min := s[0]
    for _, v := range s[1:] {
        if v < min {
            min = v
        }
    }
    return min
}

func MinString(s []string) string {
    if len(s) == 0 {
        panic("empty slice")
    }
    min := s[0]
    for _, v := range s[1:] {
        if v < min {
            min = v
        }
    }
    return min
}
```

**问题**：`MinInt` 和 `MinString` 的逻辑完全一样，只有类型不同。重复代码违背 DRY 原则，维护成本高。

**泛型前的解法**：
- 用 `interface{}` + 反射：运行时开销大，失去类型安全
- 用代码生成工具：增加构建复杂度
- 接受重复：不优雅

**泛型的解法**：用类型参数写一次，编译器为每种具体类型生成专用代码。

---

## 13.2 类型参数的语法：`[T any]`

泛型函数通过**类型参数列表**（Type Parameter List）来声明类型参数：

```go
func Min[T cmp.Ordered](s []T) T {
    if len(s) == 0 {
        panic("empty slice")
    }
    min := s[0]
    for _, v := range s[1:] {
        if v < min {
            min = v
        }
    }
    return min
}
```

**语法解析**：
- `[T cmp.Ordered]`：类型参数列表
  - `T`：类型参数名（可以任意命名，惯例用单个大写字母）
  - `cmp.Ordered`：类型约束（Constraint），限制 `T` 可以是哪些类型
- `s []T`：参数类型用到了类型参数 `T`
- 返回值 `T`：返回值类型也用到了类型参数 `T`

**调用方式**：

```go
// 显式指定类型参数
result := Min[int]([]int{3, 1, 4, 1, 5})

// 让编译器推导（见 13.5）
result := Min([]int{3, 1, 4, 1, 5})
```

**输出示例**：
```
Min[int]([3 1 4 1 5]) = 1
Min[float64]([3.14 1.41 2.71]) = 1.41
```

---

## 13.3 类型约束：`comparable` 与自定义约束

类型约束（Constraint）是一个接口，规定类型参数必须满足哪些条件。

### 13.3.1 内置约束：`any` 和 `comparable`

- `any`：等价于 `interface{}`，允许任何类型
- `comparable`：要求类型支持 `==` 和 `!=` 运算符

**哪些类型是 `comparable`**：
- 所有基本类型：`int`、`float64`、`string`、`bool`
- 指针类型
- 数组类型（元素是 `comparable`）
- 结构体类型（所有字段都是 `comparable`）

**哪些类型不是 `comparable`**：
- 切片（`[]T`）
- 映射（`map[K]V`）
- 函数类型

**示例：去重函数**：

```go
func Unique[T comparable](s []T) []T {
    seen := make(map[T]bool)
    result := []T{}
    for _, v := range s {
        if !seen[v] {
            seen[v] = true
            result = append(result, v)
        }
    }
    return result
}
```

调用：
```go
ints := Unique([]int{1, 2, 3, 2, 1})    // [1 2 3]
strs := Unique([]string{"go", "rust", "go"}) // ["go" "rust"]
```

### 13.3.2 自定义约束：联合类型

用接口 + 联合类型（Union Type）来列举允许的类型：

```go
type Number interface {
    int | int64 | float64
}

func SumNumbers[T Number](s []T) T {
    var sum T
    for _, v := range s {
        sum += v
    }
    return sum
}
```

**调用**：
```go
SumNumbers([]int{1, 2, 3})       // 6
SumNumbers([]float64{1.1, 2.2, 3.3}) // 6.6
```

**注意**：约束 `int | int64 | float64` 只接受这三种类型，不接受自定义类型（如 `type MyInt int`）。要匹配自定义类型，需要用底层类型约束 `~T`（见 13.4）。

---

## 13.4 联合类型约束与底层类型 `~T`

### 13.4.1 `~T` 匹配底层类型

`~int` 表示"底层类型是 `int` 的所有类型"：

```go
type IntegerLike interface {
    ~int | ~int64 | ~uint
}

func SumIntegers[T IntegerLike](s []T) T {
    var sum T
    for _, v := range s {
        sum += v
    }
    return sum
}
```

**测试**：
```go
type MyInt int

plainInts := []int{1, 2, 3}
myInts := []MyInt{10, 20, 30}

SumIntegers(plainInts) // 6
SumIntegers(myInts)    // 60
```

**对比**：
- `int | int64`：只接受 `int` 和 `int64` 本身
- `~int | ~int64`：接受底层类型是 `int` 或 `int64` 的所有类型（包括 `type MyInt int`）

### 13.4.2 混用基本类型和自定义类型

约束可以混合使用基本类型和自定义类型：

```go
type StringLike interface {
    int | ~string
}

func ToString[T StringLike](v T) string {
    switch val := any(v).(type) {
    case int:
        return fmt.Sprintf("%d", val)
    case string:
        return val
    default:
        // 底层类型是 string 但不是 string 本身
        return fmt.Sprint(v)
    }
}
```

**调用**：
```go
type MyString string

ToString(42)                // "42"
ToString(MyString("hello")) // "hello"
```

---

## 13.5 类型推导：什么时候可以省略类型参数

编译器可以从**函数参数**推导出类型参数，这时可以省略类型参数：

```go
// 显式指定类型参数
result := Min[int]([]int{3, 1, 4})

// 编译器从 []int 推导出 T = int
result := Min([]int{3, 1, 4})
```

**推导规则**：
1. 编译器从实参的类型推导类型参数
2. 如果参数是泛型类型，会递归推导
3. **返回值类型不参与推导**
4. 如果推导失败或有歧义，必须显式指定类型参数

**何时必须显式指定**：
- 函数没有参数：`func Create[T any]() T`
- 类型参数只在返回值中出现
- 推导有歧义（例如多个类型参数，只有部分能推导）

---

## 13.6 泛型类型：带类型参数的结构体

结构体也可以有类型参数，成为泛型类型：

```go
type Stack[T any] struct {
    items []T
}

func NewStack[T any]() *Stack[T] {
    return &Stack[T]{items: []T{}}
}

func (s *Stack[T]) Push(v T) {
    s.items = append(s.items, v)
}

func (s *Stack[T]) Pop() (T, bool) {
    if len(s.items) == 0 {
        var zero T
        return zero, false
    }
    idx := len(s.items) - 1
    v := s.items[idx]
    s.items = s.items[:idx]
    return v, true
}
```

**使用示例**：

```go
// 创建 int 类型的栈
intStack := NewStack[int]()
intStack.Push(10)
intStack.Push(20)
v, ok := intStack.Pop() // v = 20, ok = true

// 创建 string 类型的栈
strStack := NewStack[string]()
strStack.Push("go")
strStack.Push("rust")
v, ok := strStack.Pop() // v = "rust", ok = true
```

**要点**：
- 泛型类型实例化时必须指定类型参数：`NewStack[int]()`
- 方法接收者要带类型参数：`func (s *Stack[T]) Push(v T)`
- 不能在方法上再加类型参数（见 13.9）

---

## 13.7 约束中的方法集

约束不仅可以用联合类型，还可以要求类型参数实现某些方法：

```go
type HasAge interface {
    GetAge() int
}

func Oldest[T HasAge](items []T) T {
    if len(items) == 0 {
        panic("empty slice")
    }
    oldest := items[0]
    for _, v := range items[1:] {
        if v.GetAge() > oldest.GetAge() {
            oldest = v
        }
    }
    return oldest
}
```

**使用示例**：

```go
type Person struct {
    Name string
    Age  int
}

func (p Person) GetAge() int {
    return p.Age
}

people := []Person{
    {"Alice", 30},
    {"Bob", 25},
    {"Charlie", 35},
}

oldest := Oldest(people) // {Name:Charlie Age:35}
```

**要点**：
- 约束 `HasAge` 要求类型参数必须有 `GetAge() int` 方法
- 函数体内可以调用 `v.GetAge()`
- 这是泛型与接口的结合：约束是接口，类型参数是具体类型

---

## 13.8 标准库：`slices` / `maps` / `cmp`

Go 1.21 新增了三个泛型工具包：

### 13.8.1 `slices` 包

常用函数：
- `Sort[S ~[]E, E cmp.Ordered](s S)`：排序
- `SortFunc[S ~[]E, E any](s S, cmp func(a, b E) int)`：自定义比较函数排序
- `Contains[S ~[]E, E comparable](s S, v E) bool`：是否包含
- `Index[S ~[]E, E comparable](s S, v E) int`：查找索引
- `Max[S ~[]E, E cmp.Ordered](s S) E`：最大值
- `Min[S ~[]E, E cmp.Ordered](s S) E`：最小值
- `Clone[S ~[]E, E any](s S) S`：克隆
- `Equal[S ~[]E, E comparable](s1, s2 S) bool`：比较相等
- `Delete[S ~[]E, E any](s S, i, j int) S`：删除区间

**示例**：

```go
ints := []int{3, 1, 4, 1, 5, 9}
slices.Sort(ints)                // [1 1 3 4 5 9]
fmt.Println(slices.Max(ints))    // 9
fmt.Println(slices.Contains(ints, 5)) // true

// 自定义排序：按字符串长度排序
strs := []string{"go", "rust", "python", "java"}
slices.SortFunc(strs, func(a, b string) int {
    return len(a) - len(b)
})
// [go rust java python]
```

### 13.8.2 `maps` 包

常用函数：
- `Clone[M ~map[K]V, K comparable, V any](m M) M`：克隆
- `Copy[M1, M2 ~map[K]V, K comparable, V any](dst M1, src M2)`：复制
- `Equal[M1, M2 ~map[K]V, K, V comparable](m1 M1, m2 M2) bool`：比较相等
- `DeleteFunc[M ~map[K]V, K comparable, V any](m M, del func(K, V) bool)`：删除符合条件的键值对

**示例**：

```go
m1 := map[string]int{"a": 1, "b": 2}
m2 := map[string]int{"b": 2, "c": 3}

fmt.Println(maps.Equal(m1, m2)) // false

m3 := maps.Clone(m1)
fmt.Println(m3) // map[a:1 b:2]
```

### 13.8.3 `cmp` 包

常用函数：
- `Compare[T cmp.Ordered](x, y T) int`：返回 -1 / 0 / +1
- `Less[T cmp.Ordered](x, y T) bool`：`x < y`
- `Or[T comparable](vals ...T) T`：返回第一个非零值

**示例**：

```go
fmt.Println(cmp.Compare(10, 20))  // -1
fmt.Println(cmp.Or(0, 42, 100))   // 42（第一个非零值）
```

---

## 13.9 泛型不能做的事：类型参数化的方法

Go 泛型有以下限制：

### 13.9.1 方法不能有类型参数

```go
type Foo struct{}

// 编译错误：方法不能有类型参数
func (f Foo) Bar[T any](v T) {}
```

**原因**：方法调用的语法是 `f.Bar()`，没地方放类型参数 `[T]`。

**解法**：
- 把类型参数移到类型上：`type Foo[T any] struct {}`
- 或使用顶层泛型函数：`func Bar[T any](f Foo, v T)`

### 13.9.2 不能在约束中访问类型参数的字段

```go
// 编译错误：约束不能包含字段
type HasName interface {
    Name string
}
```

**原因**：约束只能是方法集，不能是字段集。

**解法**：在约束里加 `GetName()` 方法。

### 13.9.3 不能对类型参数做类型断言或类型 switch（直接）

```go
func F[T any](v T) {
    // 编译错误：类型参数不能直接做类型 switch
    switch v.(type) {}
}
```

**解法**：先转成 `any` 再断言：

```go
func F[T any](v T) {
    switch any(v).(type) {
    case int:
        fmt.Println("int")
    case string:
        fmt.Println("string")
    }
}
```

---

## 13.10 泛型与接口的取舍

什么时候用泛型，什么时候用接口？

### 13.10.1 用泛型的场景

1. **容器类型**：`Stack[T]`、`Queue[T]`、`Cache[K, V]`
2. **算法与数据结构**：`Min[T]`、`Max[T]`、`Sort[T]`、`BinarySearch[T]`
3. **需要保留具体类型**：避免 `interface{}` 装箱
4. **编译期类型安全 + 零开销抽象**：编译器为每种类型生成专用代码，无运行时开销

### 13.10.2 用接口的场景

1. **运行时多态**：不同类型的集合、插件系统
2. **依赖注入与可测试性**：`Repository`、`Logger`、`HTTPClient`
3. **标准库协议**：`io.Reader`、`io.Writer`、`error`、`fmt.Stringer`
4. **接口比泛型更简洁时**：不需要类型参数列表

### 13.10.3 经验法则

- **泛型擅长「对不同类型做同样的事」**：容器、算法、工具函数
- **接口擅长「对同一类型做不同的事」**：抽象行为、多态、依赖注入
- **不确定时优先接口**：只有遇到重复代码或装箱开销才考虑泛型

**示例对比**：

| 场景 | 选择 | 原因 |
|---|---|---|
| 写一个通用的最小值函数 | 泛型 `Min[T cmp.Ordered](s []T) T` | 逻辑相同，类型不同 |
| 写一个日志接口 | 接口 `type Logger interface { Log(string) }` | 不同实现（文件、网络、控制台），但行为相同 |
| 写一个通用的 LRU 缓存 | 泛型 `Cache[K comparable, V any]` | 容器类型，需要保留具体类型 |
| 写一个 HTTP 中间件系统 | 接口 `type Middleware interface { Handle(http.Handler) http.Handler }` | 运行时多态，链式调用 |

---

## 3 个真实报错怎么读

下面三条都来自实际编译，行号对应各自的最小示例。

**报错 1：类型实参不满足约束**

```go
type Number interface {
	~int | ~float64
}
func Sum[T Number](a, b T) T { return a + b }

func use() {
	_ = Sum[string]("a", "b") // string 不在 Number 的类型集里
}
```

```text
scratch_gencheck\main.go:18:10: string does not satisfy Number (string missing in ~int | ~float64)
```

`Sum[string]` 在调用点就被拦住，`does not satisfy` 后面的括号直接告诉你缺什么：`string missing in ~int | ~float64`。约束错误几乎都是这类「类型集不相交」问题，修复线索就藏在这半句里。

**报错 2：对类型参数做类型断言**

```go
func as[T any](v T) {
	_ = v.(string) // T 是 any，编译期无法确定动态类型
}
```

```text
scratch_gencheck\main.go:23:6: invalid operation: cannot use type assertion on type parameter value v (variable of type T constrained by any)
```

`T` 约束是 `any`，编译器没法在它身上直接断言。要断言得先把 `v` 转成 `any`（写成 `any(v).(string)`），或者改用类型 switch。这与 13.9.3 讲的「不能对类型参数做类型断言」是同一个限制。

**报错 3：`~` 用在命名类型上**

```go
type Bad string
type S interface {
	~Bad // ~ 只能直接跟在基本类型后面
}
```

```text
scratch_gencheck\main.go:29:2: invalid use of ~ (underlying type of Bad is string)
```

`~T` 里的 `T` 必须是**基本类型本身**，不能是 `type Bad string` 这样的命名类型。想表达「底层类型是 string 的所有类型」，写 `~string` 即可——它天然包含 `Bad`；反过来 `~Bad` 是错的。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
|---|---|---|
| `type MyInt int` 不匹配约束 `int \| int64` | 约束只匹配列出的类型本身，不匹配自定义类型 | 用 `~int \| ~int64` 匹配底层类型 |
| `cannot use v (variable of type T) in type switch` | 类型参数不能直接做类型 switch | 先转成 `any`：`switch any(v).(type)` |
| `method cannot have type parameters` | 方法不能有类型参数 | 把类型参数移到类型上或用顶层函数 |
| `cannot infer T` | 编译器无法推导类型参数 | 显式指定类型参数：`Func[int](...)` |
| `type parameter T does not implement comparable` | 约束不满足 | 检查约束是否包含所需操作（如 `comparable`、`cmp.Ordered`） |

---

## 练习

**第 1 题**：实现一个泛型函数 `Map[T, U any](s []T, f func(T) U) []U`，把切片 `s` 的每个元素用函数 `f` 转换成新的切片。

::: details 第 1 题参考答案

```go
func Map[T, U any](s []T, f func(T) U) []U {
    result := make([]U, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

// 测试
ints := []int{1, 2, 3}
strs := Map(ints, func(x int) string {
    return fmt.Sprintf("num_%d", x)
})
// strs = ["num_1", "num_2", "num_3"]
```

**为什么这样写**：
- 用两个类型参数 `T` 和 `U`，允许输入输出类型不同
- 用 `make([]U, len(s))` 预分配，避免多次扩容

:::

**第 2 题**：实现一个泛型函数 `Filter[T any](s []T, keep func(T) bool) []T`，返回满足条件的元素。

::: details 第 2 题参考答案

```go
func Filter[T any](s []T, keep func(T) bool) []T {
    result := []T{}
    for _, v := range s {
        if keep(v) {
            result = append(result, v)
        }
    }
    return result
}

// 测试
nums := []int{1, 2, 3, 4, 5, 6}
evens := Filter(nums, func(x int) bool {
    return x%2 == 0
})
// evens = [2, 4, 6]
```

**为什么这样写**：
- 不预分配 `result`，因为不知道最终长度
- 如果性能敏感，可以预分配 `make([]T, 0, len(s))`

:::

**第 3 题**：实现一个泛型类型 `Pair[T, U any]`，表示两个可能不同类型的值。实现方法 `Swap() Pair[U, T]` 交换两个值。

::: details 第 3 题参考答案

```go
type Pair[T, U any] struct {
    First  T
    Second U
}

func (p Pair[T, U]) Swap() Pair[U, T] {
    return Pair[U, T]{
        First:  p.Second,
        Second: p.First,
    }
}

// 测试
p := Pair[int, string]{First: 42, Second: "hello"}
swapped := p.Swap() // Pair[string, int]{First: "hello", Second: 42}
```

**为什么这样写**：
- 返回 `Pair[U, T]`（类型参数交换）而不是 `Pair[T, U]`
- 方法接收者是 `Pair[T, U]`，不是指针，因为只读操作

:::

**第 4 题**：实现一个泛型函数 `Keys[M ~map[K]V, K comparable, V any](m M) []K`，返回映射的所有键（顺序不定）。

::: details 第 4 题参考答案

```go
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
    keys := make([]K, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

// 测试
m := map[string]int{"a": 1, "b": 2, "c": 3}
keys := Keys(m) // ["a", "b", "c"]（顺序不定）
```

**为什么这样写**：
- 用 `~map[K]V` 约束，允许自定义映射类型
- 预分配 `make([]K, 0, len(m))`，避免多次扩容
- 顺序不定，因为 map 遍历是随机的

:::

**第 5 题**：写一个泛型函数 `GroupBy[T any, K comparable](s []T, key func(T) K) map[K][]T`，按 `key` 函数的返回值分组。

::: details 第 5 题参考答案

```go
func GroupBy[T any, K comparable](s []T, key func(T) K) map[K][]T {
    result := make(map[K][]T)
    for _, v := range s {
        k := key(v)
        result[k] = append(result[k], v)
    }
    return result
}

// 测试
type Person struct {
    Name string
    Age  int
}

people := []Person{
    {"Alice", 30},
    {"Bob", 25},
    {"Charlie", 30},
}

byAge := GroupBy(people, func(p Person) int {
    return p.Age
})
// byAge = map[25:[{Bob 25}] 30:[{Alice 30} {Charlie 30}]]
```

**为什么这样写**：
- `K` 必须是 `comparable`，因为要作为 map 的键
- `map[K][]T` 的值是切片，第一次 append 时会自动初始化为空切片

:::

**第 6 题**：实现一个泛型函数 `Reduce[T, U any](s []T, init U, f func(U, T) U) U`，累积计算（类似 JavaScript 的 `Array.reduce`）。

::: details 第 6 题参考答案

```go
func Reduce[T, U any](s []T, init U, f func(U, T) U) U {
    acc := init
    for _, v := range s {
        acc = f(acc, v)
    }
    return acc
}

// 测试：求和
nums := []int{1, 2, 3, 4, 5}
sum := Reduce(nums, 0, func(acc, x int) int {
    return acc + x
})
// sum = 15

// 测试：字符串拼接
words := []string{"hello", "world"}
sentence := Reduce(words, "", func(acc, s string) string {
    if acc == "" {
        return s
    }
    return acc + " " + s
})
// sentence = "hello world"
```

**为什么这样写**：
- 用两个类型参数 `T`（元素类型）和 `U`（累积器类型），允许输入输出类型不同
- `init` 是初始值，`f` 是累积函数

:::

---

## 小结

- **泛型解决重复代码问题**：用类型参数写一次，编译器为每种具体类型生成专用代码
- **类型参数语法**：`func F[T Constraint](v T) T`，约束限制类型参数的操作
- **内置约束**：`any`（任意类型）、`comparable`（可比较）、`cmp.Ordered`（可排序）
- **自定义约束**：用 `interface + 联合类型` 或 `interface + 方法集`
- **底层类型约束 `~T`**：匹配底层类型是 `T` 的所有类型（包括自定义类型）
- **类型推导**：编译器可以从参数推导类型参数，但返回值不参与推导
- **泛型类型**：`type Stack[T any] struct { items []T }`，实例化时指定类型参数
- **标准库泛型**：`slices`（排序、查找、去重）、`maps`（克隆、比较）、`cmp`（比较、取非零值）
- **泛型限制**：方法不能有类型参数，约束不能包含字段，类型参数不能直接做类型断言
- **泛型与接口取舍**：泛型擅长「对不同类型做同样的事」，接口擅长「对同一类型做不同的事」

下一章我们将学习**反射**（Reflection），在运行时检查和操作类型。
