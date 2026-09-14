# 第 15 章 · 标准库精讲（一）：时间、数学与排序

经过前面章节对 Go 语言核心语法的系统学习，你已经具备了编写完整程序的能力。但要写出高效、可靠的工程代码，还需要熟练掌握标准库——Go 的强大很大程度上来自于它设计精良的标准库。本章聚焦于日常开发中最常用的三个领域：时间处理、数学计算与排序操作。

我们将深入讲解 `time` 包的时区陷阱与单调时钟、`math` 和 `math/rand/v2` 的常用能力、`sort` 与 `slices` 包在现代 Go 中的排序实践，以及 Go 1.21+ 引入的 `maps`、`min`/`max`、`clear` 等便利工具。掌握这些内容后，你将能避免时间处理中的常见 bug，写出更简洁高效的排序和集合操作代码。

本章配套代码在 `internal/chapter/go15_stdlib_time_sort/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 15.1 time.Time 基础与创建

`time.Time` 是 Go 中表示时间点的核心类型，包含墙钟时间（wall clock）和单调时钟（monotonic clock）两部分信息。

### 获取当前时间

```go
now := time.Now()
fmt.Println(now.Year(), now.Month(), now.Day())
fmt.Println(now.Hour(), now.Minute(), now.Second())
fmt.Println(now.Weekday(), now.YearDay())
```

### 创建指定时间

```go
t := time.Date(2024, time.September, 14, 10, 30, 0, 0, time.UTC)
```

### Unix 时间戳

```go
now.Unix()       // 秒
now.UnixMilli()  // 毫秒
now.UnixNano()   // 纳秒
```

### 实测输出

```text
time.Now() = 2026-09-14 13:30:17.5547843 +0800 CST m=+0.002634601
  年=2026, 月=9, 日=14
  时=13, 分=30, 秒=17
  星期=Monday, 年中第257天
time.Date(...) = 2024-09-14 10:30:00 +0000 UTC
now.Unix() = 1789363817（秒）
```

**注意**：`time.Now()` 的输出中包含 `m=+0.002634601`，这就是单调时钟读数，保证了即使系统时间被调整，计时仍然准确。

---

## 15.2 Duration 与时间点运算

`time.Duration` 是纳秒级的有符号整数，表示两个时间点之间的时间间隔。

### 常用常量

```go
time.Nanosecond  = 1
time.Microsecond = 1000
time.Millisecond = 1000000
time.Second      = 1000000000
time.Minute      = 60 * time.Second
time.Hour        = 60 * time.Minute
```

### Duration 的创建与转换

```go
d := 2*time.Hour + 30*time.Minute + 15*time.Second
fmt.Println(d.Hours())    // 2.5
fmt.Println(d.Minutes())  // 150
fmt.Println(d.Seconds())  // 9015
```

### 时间点运算

```go
now := time.Now()
later := now.Add(2 * time.Hour)
diff := later.Sub(now)     // 2h0m0s
now.Before(later)          // true
now.Equal(later)           // false
```

### 实测输出

```text
Duration = 2h30m15s
  小时=2.50, 分钟=150, 秒=9015
now + 2h = 2026-09-14 15:30:17...
later - now = 2h0m0s
now.Before(later) = true
```

---

## 15.3 Layout 格式化与解析

Go 的时间格式化采用**参考时间**（reference time）作为 layout：

> **Mon Jan 2 15:04:05 MST 2006**

记忆口诀：1 2 3 4 5 6 7 8（月份=1，日=2，时=3，分=4，秒=5，年=6，时区=7，...）

### 格式化

```go
t := time.Date(2024, time.September, 14, 10, 30, 0, 0, time.UTC)

fmt.Println(t.Format(time.RFC3339))              // 2024-09-14T10:30:00Z
fmt.Println(t.Format("2006-01-02 15:04:05"))     // 2024-09-14 10:30:00
fmt.Println(t.Format("2006年01月02日 15时04分"))  // 2024年09月14日 10时30分
```

### 解析

```go
// Parse：无时区信息时默认 UTC
t, err := time.Parse("2006-01-02 15:04:05", "2024-09-14 10:30:00")

// ParseInLocation：指定时区
t, err := time.ParseInLocation("2006-01-02 15:04:05", "2024-09-14 10:30:00", shanghai)
```

### 实测输出

```text
RFC3339: 2024-09-14T10:30:00Z
自定义: 2024-09-14 10:30:00
中文: 2024年09月14日 10时30分
解析结果: 2024-09-14 10:30:00 +0000 UTC
带时区解析: 2024-09-14 10:30:00 +0800 CST
```

**常见坑**：不要用其他语言的时间格式（如 `"YYYY-MM-DD"`），Go 的 layout 必须用参考时间。

---

## 15.4 时区与 Location 陷阱

时区处理是时间操作中最容易出错的地方。

### 加载时区

```go
sh, err := time.LoadLocation("Asia/Shanghai")
if err != nil {
    sh = time.FixedZone("CST", 8*60*60)
}
```

### 同一时刻的不同时区表示

```go
utc := time.Date(2024, time.September, 14, 10, 0, 0, 0, time.UTC)
local := utc.In(sh)
fmt.Println(utc.Equal(local)) // true，同一时刻
```

### Parse vs ParseInLocation

```go
// Parse：无时区信息时默认 UTC
t, _ := time.Parse("2006-01-02 15:04:05", "2024-09-14 10:00:00")
fmt.Println(t.Location()) // UTC

// ParseInLocation：指定本地时区
t2, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-09-14 10:00:00", sh)
fmt.Println(t2.Location()) // Asia/Shanghai
```

### 实测输出

```text
UTC:  2024-09-14 10:00:00 +0000 UTC
上海: 2024-09-14 18:00:00 +0800 CST
两者是同一时刻: true
Parse 结果（无时区用 UTC）: 2024-09-14 10:00:00 +0000 UTC, Location=UTC
ParseInLocation 结果: 2024-09-14 10:00:00 +0800 CST, Location=Asia/Shanghai
陷阱：Parse 默认 UTC，ParseInLocation 才能指定本地时区
```

**常见坑**：
- `time.Parse` 默认 UTC，处理本地时间要用 `ParseInLocation`
- 夏令时（DST）会导致某些时区的时间不存在或重复
- 避免在数据库中存储带时区的时间字符串，优先用 UTC

---

## 15.5 Timer 与 Ticker

### Timer：一次性延迟

```go
timer := time.NewTimer(100 * time.Millisecond)
<-timer.C // 等待触发

// 或简单用法（不能取消）
<-time.After(100 * time.Millisecond)
```

### Ticker：周期性触发

```go
ticker := time.NewTicker(50 * time.Millisecond)
defer ticker.Stop() // 必须停止，否则会泄漏

for t := range ticker.C {
    fmt.Println(t)
}
```

### AfterFunc：延迟回调

```go
time.AfterFunc(50*time.Millisecond, func() {
    fmt.Println("回调执行")
})
```

### 实测输出

```text
Timer 触发，耗时约 100.2256ms
  Ticker: 13:30:17.719
  Ticker: 13:30:17.769
  Ticker: 13:30:17.819
AfterFunc 回调执行
```

**最佳实践**：
- `time.After` 不能取消，长时间等待会内存泄漏，优先用 `time.NewTimer`
- `Ticker` 必须调用 `Stop()`，否则 goroutine 会泄漏
- `Reset()` 前要先 `Stop()` 并排空通道，否则可能立即触发

---

## 15.6 单调时钟与 Since

`time.Since(start)` 和 `time.Until(deadline)` 使用单调时钟，不受系统时间调整影响。

```go
start := time.Now()
time.Sleep(100 * time.Millisecond)
elapsed := time.Since(start)
```

### 单调时钟的局限

单调时钟信息在序列化后会丢失：

```go
t := time.Now()
t2, _ := time.Parse(time.RFC3339Nano, t.Format(time.RFC3339Nano))
// t2 丢失了单调时钟信息
```

### 实测输出

```text
Sleep 100ms 后，time.Since = 100.7626ms
单调时钟保证了即使系统时间被用户或 NTP 调整，计时仍然准确
序列化前 Wall+Monotonic: 2026-09-14 13:30:17.9716433 +0800 CST m=+0.419493601
序列化后只有 Wall: 2026-09-14 13:30:17.9716433 +0800 CST
通过网络传输或数据库存储后，单调时钟信息会丢失
```

**工程建议**：
- 超时判断、性能计时优先用 `time.Since`（单调时钟）
- 不要比较两个来自不同机器的时间（单调时钟不可比）
- 数据库存储时间用 UTC，应用层再做时区转换

---

## 15.7 math 与 math/rand/v2

### math 包常用函数

```go
math.Pi                    // 3.141593...
math.Sqrt(2)               // 平方根
math.Pow(2, 10)            // 幂运算
math.Max(3, 5)             // 最大值
math.Min(3, 5)             // 最小值
math.Ceil(2.3)             // 向上取整
math.Floor(2.7)            // 向下取整
math.Abs(-5)               // 绝对值
```

### math/rand/v2（Go 1.22+）

```go
import "math/rand/v2"

rand.IntN(100)        // [0, 100) 的随机整数
rand.Float64()        // [0.0, 1.0) 的随机浮点
rand.NormFloat64()    // 标准正态分布

// 可复现的序列
r := rand.New(rand.NewPCG(42, 1))
fmt.Println(r.IntN(100))
```

### 实测输出

```text
math.Pi = 3.141593
math.Sqrt(2) = 1.414214
math.Pow(2, 10) = 1024
math.Max(3, 5) = 5, math.Min(3, 5) = 3
math.Ceil(2.3) = 3, math.Floor(2.7) = 2
math.Abs(-5) = 5
math/rand/v2 随机数:
  IntN(100) = 65
  Float64() = 0.7298
  N(0, 1) = 1.4499（正态分布）
固定种子序列: 18, 57, 57
```

**注意**：`math/rand` 是伪随机数，不适合加密场景。需要密码学安全随机数时，使用 `crypto/rand`。

---

## 15.8 sort、slices.SortFunc 与 cmp

### 经典 sort 包

```go
nums := []int{3, 1, 4, 1, 5, 9, 2, 6}
sort.Ints(nums) // 原地排序

// 二分查找
idx := sort.Search(len(nums), func(i int) bool {
    return nums[i] >= 5
})
```

### slices 包（Go 1.21+）

```go
type Person struct {
    Name string
    Age  int
}

people := []Person{{"Bob", 30}, {"Alice", 25}, {"Charlie", 35}}

// 按 Age 排序
slices.SortFunc(people, func(a, b Person) int {
    return cmp.Compare(a.Age, b.Age)
})

// 二分查找
i, found := slices.BinarySearch(nums2, 5)

// 包含判断
slices.Contains(nums2, 5)
```

### 实测输出

```text
sort.Ints: [1 1 2 3 4 5 6 9]
sort.Search >=5: 索引=5, 值=5
按 Age 排序: [{Alice 25} {Bob 30} {Charlie 35}]
按 Name 排序: [{Alice 25} {Bob 30} {Charlie 35}]
slices.BinarySearch(5): 索引=2, 找到=true
slices.Contains(nums2, 5) = true
```

**性能对比**：
- `slices.Sort`/`slices.SortFunc` 通常比 `sort.Slice` 更快，因为泛型避免了接口转换开销
- `slices.Sort` 要求元素可比较（`constraints.Ordered`）
- `slices.SortFunc` 适用于自定义比较逻辑

---

## 15.9 maps、min / max / clear

### maps 包（Go 1.21+）

```go
m1 := map[string]int{"a": 1, "b": 2}
m2 := map[string]int{"a": 1, "b": 2}

// 比较
maps.Equal(m1, m2) // true

// 复制
maps.Copy(dst, src)

// 克隆
clone := maps.Clone(m1)
```

### min / max（Go 1.21+）

```go
min(3, 7, 2, 9)   // 2
max(3, 7, 2, 9)   // 9
min("banana", "apple", "cherry") // "apple"
```

### clear（Go 1.21+）

```go
m := map[string]int{"a": 1, "b": 2, "c": 3}
clear(m) // m 变为空 map，len = 0

s := []int{1, 2, 3, 4, 5}
clear(s) // s 变为 [0, 0, 0, 0, 0]，len 不变
```

### 实测输出

```text
maps.Equal(m1, m2) = true
maps.Equal(m1, m3) = false
maps.Copy 后 dst = map[a:1 b:2 x:10]
clone = map[a:1 b:2 c:3], m1 = map[a:1 b:2]（互不影响）
min(3, 7, 2, 9) = 2
max(3, 7, 2, 9) = 9
min("banana", "apple", "cherry") = apple
clear 前: map[a:1 b:2 c:3], len=3
clear 后: map[], len=0
clear 前 slice: [1 2 3 4 5], len=5, cap=5
clear 后 slice: [0 0 0 0 0], len=5, cap=5
```

**注意**：
- `clear(map)` 清空所有键值对，但保留 map 本身（可以继续使用）
- `clear(slice)` 将所有元素置为零值，但长度和容量不变
- `maps` 包操作都是浅拷贝

---

## 3 个真实报错怎么读

### 报错 1：Parse 时区错误

```text
parsing time "2024-09-14 10:00:00" as "2006-01-02T15:04:05Z07:00": cannot parse " 10:00:00" as "T"
```

**原因**：layout 与实际字符串格式不匹配，比如用 `time.RFC3339` 解析普通日期时间字符串。

**解法**：确保 layout 与输入字符串格式一致。普通格式用 `"2006-01-02 15:04:05"`。

### 报错 2：slices.SortFunc 类型不匹配

```text
cannot use generic function slices.SortFunc without instantiation
```

**原因**：`slices.SortFunc` 是泛型函数，编译器无法推导类型参数。

**解法**：确保传入的切片类型明确，或显式指定类型参数：

```go
slices.SortFunc(people, func(a, b Person) int { ... })
```

### 报错 3：math/rand/v2 未找到

```text
cannot find package "math/rand/v2"
```

**原因**：Go 版本低于 1.22，`math/rand/v2` 尚未引入。

**解法**：升级到 Go 1.22+，或继续使用 `math/rand`（注意种子初始化方式不同）。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| Parse 后的时间比预期晚/早 8 小时 | `time.Parse` 默认 UTC | 用 `ParseInLocation` 指定时区 |
| 时间格式化输出是 UTC | `time.Now()` 有时区信息，但 `Format` 不改变时区 | 用 `t.In(location)` 转换后再格式化 |
| Timer/Timer 泄漏 | 忘记 `Stop()` 或 `time.After` 不能取消 | 用 `time.NewTimer` + `defer timer.Stop()` |
| `slices.Sort` 编译失败 | 元素类型不可比较 | 用 `slices.SortFunc` + 自定义比较函数 |
| `rand.Intn` 每次运行结果相同 | `math/rand` 默认种子固定 | 调用 `rand.Seed(time.Now().UnixNano())`，或用 `math/rand/v2` |
| `clear(map)` 后 map 为 nil | 误解 clear 行为 | `clear` 清空键值但保留 map，可以继续写入 |
| `time.Since` 在系统时间调整后不准 | 使用了非单调时钟的比较 | `time.Since` 本身使用单调时钟，但要确保序列化后重新解析的时间没有单调时钟 |

---

## 练习

### 第 1 题

写一个函数 `FormatDuration`，接收 `time.Duration`，返回人类可读的字符串，如 `"2小时30分"`、`"1分45秒"`、`"500毫秒"`。小于 1 秒的显示毫秒，大于等于 1 分钟的显示分钟和秒。

::: details 第 1 题参考答案

```go
func FormatDuration(d time.Duration) string {
    if d < time.Second {
        return fmt.Sprintf("%d毫秒", d.Milliseconds())
    }
    if d < time.Minute {
        return fmt.Sprintf("%.0f秒", d.Seconds())
    }
    m := int(d.Minutes())
    s := int(d.Seconds()) % 60
    return fmt.Sprintf("%d分%d秒", m, s)
}
```

**为什么这样写更好**：Duration 的 `Hours()`/`Minutes()`/`Seconds()` 返回浮点数，实际展示时应该转换为整数，避免显示 `"2.5分钟"` 这类不友好的格式。

:::

### 第 2 题

解析字符串 `"2024-12-25 08:00:00"`，假设它是北京时间（Asia/Shanghai），然后转换为 UTC 的 RFC3339 格式输出。

::: details 第 2 题参考答案

```go
sh, _ := time.LoadLocation("Asia/Shanghai")
t, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-12-25 08:00:00", sh)
utc := t.UTC()
fmt.Println(utc.Format(time.RFC3339)) // 2024-12-25T00:00:00Z
```

**关键点**：必须用 `ParseInLocation` 而不是 `Parse`，否则 08:00 会被当作 UTC 时间，转换后结果会错 8 小时。

:::

### 第 3 题

用 `slices.SortFunc` 对一个 `[]string` 按字符串长度排序，长度相同则按字典序排序。

::: details 第 3 题参考答案

```go
words := []string{"Go", "Python", "Java", "C", "Rust", "Ruby"}
slices.SortFunc(words, func(a, b string) int {
    if len(a) != len(b) {
        return cmp.Compare(len(a), len(b))
    }
    return cmp.Compare(a, b)
})
// 结果: [C, Go, Java, Ruby, Rust, Python]
```

**为什么用 `cmp.Compare`**：`cmp.Compare` 是 Go 1.21+ 引入的通用比较函数，避免手动写 `if a < b { return -1 }` 的样板代码，语义更清晰。

:::

### 第 4 题

写一个函数 `RandomIntRange(min, max int)`，返回 `[min, max)` 范围内的随机整数。要求使用 `math/rand/v2`，并确保在 `min >= max` 时 panic。

::: details 第 4 题参考答案

```go
func RandomIntRange(min, max int) int {
    if min >= max {
        panic(fmt.Sprintf("invalid range: min=%d >= max=%d", min, max))
    }
    return min + rand.IntN(max-min)
}
```

**为什么用 `rand.IntN(n)`**：`IntN(n)` 直接返回 `[0, n)` 的随机数，不需要自己取模，避免了取模偏差问题。`math/rand/v2` 的 `IntN` 使用拒绝采样算法，分布更均匀。

:::

### 第 5 题

对比 `time.After(5 * time.Minute)` 和 `time.NewTimer(5 * time.Minute)` 在以下场景中的差异：一个 HTTP 请求处理函数中设置 5 分钟超时，但请求通常在 10 秒内完成。哪种写法更好？为什么？

::: details 第 5 题参考答案

```go
// 不好的写法：内存泄漏
func handleBad(w http.ResponseWriter, r *http.Request) {
    select {
    case <-time.After(5 * time.Minute):
        // 超时处理
    case result := <-doWork():
        // 正常处理
    }
}

// 好的写法：可取消
func handleGood(w http.ResponseWriter, r *http.Request) {
    timer := time.NewTimer(5 * time.Minute)
    defer timer.Stop()
    select {
    case <-timer.C:
        // 超时处理
    case result := <-doWork():
        // 正常处理
    }
}
```

**原因**：`time.After` 创建的 Timer 在超时前不会被 GC 回收。如果请求在 10 秒内完成，但 Timer 还要等 5 分钟才到期，这段时间内 Timer 会一直占用内存。`NewTimer` + `Stop()` 可以立即释放资源。

:::

### 第 6 题

使用 `maps.Clone` 和 `clear` 实现一个函数 `ResetMap(m map[string]int)`，将 map 清空后填入一组默认值 `{"status": 200, "count": 0}`，但要求不影响传入的原始 map（函数内部操作副本）。

::: details 第 6 题参考答案

```go
func ResetMap(m map[string]int) map[string]int {
    // 克隆一份，不影响原始 map
    result := mapsClone(m)
    clear(result)
    result["status"] = 200
    result["count"] = 0
    return result
}
```

**注意**：`maps.Clone` 在 Go 1.21+ 标准库中可用，是浅拷贝。如果值类型是指针或切片，克隆后的 map 和原 map 会共享这些值。本章为了兼容性演示了手写 `mapsClone` 的实现。

:::

---

## 小结

本章覆盖了 Go 标准库中时间、数学与排序的核心能力，关键要点：

- **`time.Time`**：包含墙钟时间和单调时钟，使用参考时间 `2006-01-02 15:04:05` 作为 layout 模板。
- **`Duration`**：纳秒级整数，支持 `Add`/`Sub`/`Since` 等运算，优先用 `time.Since` 做计时。
- **时区陷阱**：`time.Parse` 默认 UTC，处理本地时间用 `ParseInLocation`；数据库存储用 UTC。
- **Timer/Ticker**：`NewTimer` + `Stop()` 可避免泄漏，`time.After` 不能取消，长时间等待会内存泄漏。
- **单调时钟**：`time.Since` 使用单调时钟，不受系统时间调整影响，但序列化后会丢失。
- **`math`/`math/rand/v2`**：标准数学函数和伪随机数生成，`rand/v2` 提供更好的 API 和分布算法。
- **排序**：`slices.Sort`/`slices.SortFunc` 利用泛型避免接口开销，是 Go 1.21+ 的推荐写法。
- **新工具函数**：`maps.Equal`/`Clone`/`Copy`、`min`/`max`、`clear` 大幅简化了集合操作。

掌握这些标准库工具后，你的代码将更加简洁、高效和可靠。下一章将学习标准库精讲（二）：正则、文本与模板，深入掌握字符串处理和文本生成技术。
