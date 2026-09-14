package go15_stdlib_time_sort

import (
	"slices"
	"testing"
	"time"
)

// TestTimeCreation 验证时间创建。
func TestTimeCreation(t *testing.T) {
	tm := time.Date(2024, time.September, 14, 10, 30, 0, 0, time.UTC)
	if tm.Year() != 2024 || tm.Month() != time.September || tm.Day() != 14 {
		t.Errorf("unexpected date: %v", tm)
	}
}

// TestDurationArithmetic 验证 Duration 运算。
func TestDurationArithmetic(t *testing.T) {
	d := 2*time.Hour + 30*time.Minute
	if d.Hours() != 2.5 {
		t.Errorf("duration = %f hours, want 2.5", d.Hours())
	}
}

// TestTimeParseFormat 验证时间解析与格式化。
func TestTimeParseFormat(t *testing.T) {
	s := "2024-09-14 10:30:00"
	tm, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	formatted := tm.Format("2006-01-02 15:04:05")
	if formatted != s {
		t.Errorf("formatted = %q, want %q", formatted, s)
	}
}

// TestSortInts 验证 sort.Ints。
func TestSortInts(t *testing.T) {
	nums := []int{3, 1, 4, 1, 5}
	slices.Sort(nums)
	if !slices.IsSorted(nums) {
		t.Errorf("not sorted: %v", nums)
	}
}

// TestSlicesSortFunc 验证 slices.SortFunc。
func TestSlicesSortFunc(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{{"Bob", 30}, {"Alice", 25}}
	slices.SortFunc(people, func(a, b Person) int {
		return a.Age - b.Age
	})
	if people[0].Name != "Alice" {
		t.Errorf("first = %v, want Alice", people[0])
	}
}

// TestMapsEqual 验证 mapsEqual。
func TestMapsEqual(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"a": 1, "b": 2}
	m3 := map[string]int{"a": 1, "b": 3}
	if !mapsEqual(m1, m2) {
		t.Error("mapsEqual(m1, m2) should be true")
	}
	if mapsEqual(m1, m3) {
		t.Error("mapsEqual(m1, m3) should be false")
	}
}

// TestMapsClone 验证 mapsClone。
func TestMapsClone(t *testing.T) {
	m1 := map[string]int{"a": 1}
	m2 := mapsClone(m1)
	m2["b"] = 2
	if _, ok := m1["b"]; ok {
		t.Error("clone should not affect original")
	}
}

// TestMinMax 验证 min/max。
func TestMinMax(t *testing.T) {
	if min(3, 7, 2) != 2 {
		t.Errorf("min(3,7,2) = %d, want 2", min(3, 7, 2))
	}
	if max(3, 7, 2) != 7 {
		t.Errorf("max(3,7,2) = %d, want 7", max(3, 7, 2))
	}
}

// TestChapterMetadata 保证章节编号和标题跟文档里的章节名一致。
func TestChapterMetadata(t *testing.T) {
	if Chapter != 15 {
		t.Errorf("Chapter = %d, want 15", Chapter)
	}
	if ChapterTitle != "标准库精讲（一）：时间、数学与排序" {
		t.Errorf("ChapterTitle = %q, want 标准库精讲（一）：时间、数学与排序", ChapterTitle)
	}
}
