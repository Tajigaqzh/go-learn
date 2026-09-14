// Package go15_stdlib_time_sort 演示 Go 标准库中的时间、数学与排序。
//
// 涵盖主题：
//   - time.Time / Duration / Location
//   - Layout 格式化与解析
//   - 时区与 UTC 陷阱
//   - Timer / Ticker
//   - 单调时钟与 time.Since
//   - math 与 math/rand/v2
//   - sort、slices.SortFunc / cmp
//   - maps、min / max / clear
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go15_stdlib_time_sort/
package go15_stdlib_time_sort

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sort"
	"time"
)

// Demo 是第 15 章的入口函数，按小节顺序演示标准库的核心能力。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s ==========\n", Chapter, "stdlib_time_sort")
	fmt.Println("========================================")

	section1_TimeBasics()
	section2_Duration()
	section3_LayoutFormat()
	section4_TimezoneTrap()
	section5_TimerTicker()
	section6_MonotonicClock()
	section7_MathAndRand()
	section8_SortAndCmp()
	section9_MapsMinMaxClear()

	fmt.Println("========================================")
	fmt.Println("========== 标准库精讲（一）演示结束 ==========")
	fmt.Println()
}

// --- 15.1 time.Time 基础与创建 ---
func section1_TimeBasics() {
	fmt.Println("\n--- 15.1 time.Time 基础与创建 ---")

	// 获取当前时间
	now := time.Now()
	fmt.Printf("time.Now() = %v\n", now)
	fmt.Printf("  年=%d, 月=%d, 日=%d\n", now.Year(), now.Month(), now.Day())
	fmt.Printf("  时=%d, 分=%d, 秒=%d\n", now.Hour(), now.Minute(), now.Second())
	fmt.Printf("  星期=%s, 年中第%d天\n", now.Weekday(), now.YearDay())

	// 创建指定时间
	t := time.Date(2024, time.September, 14, 10, 30, 0, 0, time.UTC)
	fmt.Printf("time.Date(...) = %v\n", t)

	// Unix 时间戳
	fmt.Printf("now.Unix() = %d（秒）\n", now.Unix())
	fmt.Printf("now.UnixMilli() = %d（毫秒）\n", now.UnixMilli())
	fmt.Printf("now.UnixNano() = %d（纳秒）\n", now.UnixNano())
}

// --- 15.2 Duration 与时间点运算 ---
func section2_Duration() {
	fmt.Println("\n--- 15.2 Duration 与时间点运算 ---")

	// Duration 是纳秒级的有符号整数
	d := 2*time.Hour + 30*time.Minute + 15*time.Second
	fmt.Printf("Duration = %v\n", d)
	fmt.Printf("  小时=%.2f, 分钟=%.0f, 秒=%.0f\n", d.Hours(), d.Minutes(), d.Seconds())
	fmt.Printf("  毫秒=%d, 微秒=%d, 纳秒=%d\n", d.Milliseconds(), d.Microseconds(), d.Nanoseconds())

	// 时间点加减
	now := time.Now()
	later := now.Add(2 * time.Hour)
	fmt.Printf("now + 2h = %v\n", later)

	// 两个时间点的差值
	diff := later.Sub(now)
	fmt.Printf("later - now = %v\n", diff)

	// 比较时间点
	fmt.Printf("now.Before(later) = %t\n", now.Before(later))
	fmt.Printf("now.Equal(later) = %t\n", now.Equal(later))
}

// --- 15.3 Layout 格式化与解析 ---
func section3_LayoutFormat() {
	fmt.Println("\n--- 15.3 Layout 格式化与解析 ---")

	now := time.Date(2024, time.September, 14, 10, 30, 0, 0, time.UTC)

	// 使用参考时间 Mon Jan 2 15:04:05 MST 2006 作为 layout
	fmt.Printf("RFC3339: %s\n", now.Format(time.RFC3339))
	fmt.Printf("自定义: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("中文: %s\n", now.Format("2006年01月02日 15时04分"))

	// 解析时间字符串
	s := "2024-09-14 10:30:00"
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		fmt.Printf("解析失败: %v\n", err)
	} else {
		fmt.Printf("解析结果: %v\n", t)
	}

	// 带时区的解析
	s2 := "2024-09-14T10:30:00+08:00"
	t2, err := time.Parse(time.RFC3339, s2)
	if err != nil {
		fmt.Printf("解析失败: %v\n", err)
	} else {
		fmt.Printf("带时区解析: %v\n", t2)
	}
}

// --- 15.4 时区与 Location 陷阱 ---
func section4_TimezoneTrap() {
	fmt.Println("\n--- 15.4 时区与 Location 陷阱 ---")

	// 加载时区（可能失败，因为需要时区数据库）
	sh, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Printf("加载时区失败: %v\n", err)
		sh = time.FixedZone("CST", 8*60*60) // 备用方案
	}

	// 同一个时刻，不同时区表示
	utc := time.Date(2024, time.September, 14, 10, 0, 0, 0, time.UTC)
	local := utc.In(sh)
	fmt.Printf("UTC:  %v\n", utc)
	fmt.Printf("上海: %v\n", local)
	fmt.Printf("两者是同一时刻: %t\n", utc.Equal(local))

	// 陷阱：time.Parse 不带时区信息时，使用 UTC
	t, _ := time.Parse("2006-01-02 15:04:05", "2024-09-14 10:00:00")
	fmt.Printf("Parse 结果（无时区用 UTC）: %v, Location=%v\n", t, t.Location())

	// time.ParseInLocation 可以指定时区
	t2, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-09-14 10:00:00", sh)
	fmt.Printf("ParseInLocation 结果: %v, Location=%v\n", t2, t2.Location())

	fmt.Println("陷阱：Parse 默认 UTC，ParseInLocation 才能指定本地时区")
}

// --- 15.5 Timer 与 Ticker ---
func section5_TimerTicker() {
	fmt.Println("\n--- 15.5 Timer 与 Ticker ---")

	// Timer：一次性延迟执行
	timer := time.NewTimer(100 * time.Millisecond)
	start := time.Now()
	<-timer.C
	fmt.Printf("Timer 触发，耗时约 %v\n", time.Since(start))

	// 也可以用 time.After（不能手动停止，会内存泄漏）
	// <-time.After(100 * time.Millisecond)

	// Ticker：周期性触发
	ticker := time.NewTicker(50 * time.Millisecond)
	count := 0
	for t := range ticker.C {
		fmt.Printf("  Ticker: %v\n", t.Format("15:04:05.000"))
		count++
		if count >= 3 {
			ticker.Stop()
			break
		}
	}

	// time.AfterFunc：延迟执行回调
	done := make(chan bool)
	time.AfterFunc(50*time.Millisecond, func() {
		fmt.Println("AfterFunc 回调执行")
		done <- true
	})
	<-done
}

// --- 15.6 单调时钟与 Since ---
func section6_MonotonicClock() {
	fmt.Println("\n--- 15.6 单调时钟与 Since ---")

	// time.Since 使用单调时钟，不受系统时间调整影响
	start := time.Now()
	time.Sleep(100 * time.Millisecond)
	elapsed := time.Since(start)
	fmt.Printf("Sleep 100ms 后，time.Since = %v\n", elapsed)

	// 单调时钟存在于 time.Time 内部，只在本地运算时有效
	fmt.Println("单调时钟保证了即使系统时间被用户或 NTP 调整，计时仍然准确")

	// 序列化后单调时钟会丢失
	t := time.Now()
	t2, _ := time.Parse(time.RFC3339Nano, t.Format(time.RFC3339Nano))
	fmt.Printf("序列化前 Wall+Monotonic: %v\n", t)
	fmt.Printf("序列化后只有 Wall: %v\n", t2)
	fmt.Println("通过网络传输或数据库存储后，单调时钟信息会丢失")
}

// --- 15.7 math 与 math/rand/v2 ---
func section7_MathAndRand() {
	fmt.Println("\n--- 15.7 math 与 math/rand/v2 ---")

	// math 包常用函数
	fmt.Printf("math.Pi = %.6f\n", math.Pi)
	fmt.Printf("math.Sqrt(2) = %.6f\n", math.Sqrt(2))
	fmt.Printf("math.Pow(2, 10) = %.0f\n", math.Pow(2, 10))
	fmt.Printf("math.Max(3, 5) = %.0f, math.Min(3, 5) = %.0f\n", math.Max(3, 5), math.Min(3, 5))
	fmt.Printf("math.Ceil(2.3) = %.0f, math.Floor(2.7) = %.0f\n", math.Ceil(2.3), math.Floor(2.7))
	fmt.Printf("math.Abs(-5) = %.0f\n", math.Abs(-5))

	// math/rand/v2（Go 1.22+）
	fmt.Println("math/rand/v2 随机数:")
	fmt.Printf("  IntN(100) = %d\n", rand.IntN(100))
	fmt.Printf("  Float64() = %.4f\n", rand.Float64())
	fmt.Printf("  N(0, 1) = %.4f（正态分布）\n", rand.NormFloat64())

	// 可复现的随机序列：使用固定种子
	r := rand.New(rand.NewPCG(42, 1))
	fmt.Printf("固定种子序列: %d, %d, %d\n", r.IntN(100), r.IntN(100), r.IntN(100))
}

// --- 15.8 sort、slices.SortFunc 与 cmp ---
func section8_SortAndCmp() {
	fmt.Println("\n--- 15.8 sort、slices.SortFunc 与 cmp ---")

	// 经典 sort.Ints
	nums := []int{3, 1, 4, 1, 5, 9, 2, 6}
	sort.Ints(nums)
	fmt.Printf("sort.Ints: %v\n", nums)

	// sort.Search 二分查找
	idx := sort.Search(len(nums), func(i int) bool {
		return nums[i] >= 5
	})
	fmt.Printf("sort.Search >=5: 索引=%d, 值=%d\n", idx, nums[idx])

	// slices.SortFunc（Go 1.21+）
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Bob", 30},
		{"Alice", 25},
		{"Charlie", 35},
	}

	// 按 Age 排序
	slices.SortFunc(people, func(a, b Person) int {
		return cmp.Compare(a.Age, b.Age)
	})
	fmt.Printf("按 Age 排序: %v\n", people)

	// 按 Name 排序
	slices.SortFunc(people, func(a, b Person) int {
		return cmp.Compare(a.Name, b.Name)
	})
	fmt.Printf("按 Name 排序: %v\n", people)

	// slices.BinarySearch
	nums2 := []int{1, 3, 5, 7, 9}
	i, found := slices.BinarySearch(nums2, 5)
	fmt.Printf("slices.BinarySearch(5): 索引=%d, 找到=%t\n", i, found)

	// slices.Contains
	fmt.Printf("slices.Contains(nums2, 5) = %t\n", slices.Contains(nums2, 5))
}

// --- 15.9 maps、min / max / clear ---
func section9_MapsMinMaxClear() {
	fmt.Println("\n--- 15.9 maps、min / max / clear ---")

	// maps.Equal 比较两个 map
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"a": 1, "b": 2}
	m3 := map[string]int{"a": 1, "b": 3}
	fmt.Printf("maps.Equal(m1, m2) = %t\n", mapsEqual(m1, m2))
	fmt.Printf("maps.Equal(m1, m3) = %t\n", mapsEqual(m1, m3))

	// maps.Copy
	dst := map[string]int{"x": 10}
	mapsCopy(dst, m1)
	fmt.Printf("maps.Copy 后 dst = %v\n", dst)

	// maps.Clone
	clone := mapsClone(m1)
	clone["c"] = 3
	fmt.Printf("clone = %v, m1 = %v（互不影响）\n", clone, m1)

	// min / max（Go 1.21+）
	fmt.Printf("min(3, 7, 2, 9) = %d\n", min(3, 7, 2, 9))
	fmt.Printf("max(3, 7, 2, 9) = %d\n", max(3, 7, 2, 9))
	fmt.Printf("min(\"banana\", \"apple\", \"cherry\") = %s\n", min("banana", "apple", "cherry"))

	// clear（Go 1.21+）
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	fmt.Printf("clear 前: %v, len=%d\n", m, len(m))
	clear(m)
	fmt.Printf("clear 后: %v, len=%d\n", m, len(m))

	s := []int{1, 2, 3, 4, 5}
	fmt.Printf("clear 前 slice: %v, len=%d, cap=%d\n", s, len(s), cap(s))
	clear(s)
	fmt.Printf("clear 后 slice: %v, len=%d, cap=%d\n", s, len(s), cap(s))
}

// mapsEqual 比较两个 map 是否相等（Go 1.21+ 有 maps.Equal）。
func mapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// mapsCopy 将 src 的所有键值对复制到 dst。
func mapsCopy(dst, src map[string]int) {
	for k, v := range src {
		dst[k] = v
	}
}

// mapsClone 返回 map 的浅拷贝。
func mapsClone(m map[string]int) map[string]int {
	c := make(map[string]int, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}
