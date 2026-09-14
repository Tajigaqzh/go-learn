package go07_slices_maps

import "testing"

// TestArrayValueType 测试数组是值类型
func TestArrayValueType(t *testing.T) {
	arr1 := [3]int{1, 2, 3}
	arr2 := arr1
	arr2[0] = 999

	if arr1[0] != 1 {
		t.Errorf("数组赋值应该是复制，arr1[0] 应该是 1，实际是 %d", arr1[0])
	}
	if arr2[0] != 999 {
		t.Errorf("arr2[0] 应该是 999，实际是 %d", arr2[0])
	}
}

// TestSliceSharing 测试切片共享底层数组
func TestSliceSharing(t *testing.T) {
	original := []int{1, 2, 3, 4, 5}
	sub := original[1:3] // [2, 3]

	sub[0] = 999

	if original[1] != 999 {
		t.Errorf("修改子切片应该影响原切片，original[1] 应该是 999，实际是 %d", original[1])
	}
}

// TestCopy 测试 copy 函数
func TestCopy(t *testing.T) {
	src := []int{1, 2, 3, 4, 5}
	dst := make([]int, 3)

	n := copy(dst, src)

	if n != 3 {
		t.Errorf("copy 应该返回 3，实际返回 %d", n)
	}
	if len(dst) != 3 {
		t.Errorf("dst 长度应该是 3，实际是 %d", len(dst))
	}
	if dst[0] != 1 || dst[1] != 2 || dst[2] != 3 {
		t.Errorf("dst 内容不正确: %v", dst)
	}

	// 修改 dst 不影响 src
	dst[0] = 999
	if src[0] != 1 {
		t.Errorf("修改 dst 不应该影响 src")
	}
}

// TestNilSlice 测试 nil 切片
func TestNilSlice(t *testing.T) {
	var s []int

	if s != nil {
		t.Errorf("nil 切片应该等于 nil")
	}
	if len(s) != 0 {
		t.Errorf("nil 切片长度应该是 0")
	}
	if cap(s) != 0 {
		t.Errorf("nil 切片容量应该是 0")
	}

	// append 到 nil 切片是安全的
	s = append(s, 1)
	if len(s) != 1 || s[0] != 1 {
		t.Errorf("append 到 nil 切片失败")
	}
}

// TestEmptySlice 测试空切片
func TestEmptySlice(t *testing.T) {
	s := []int{}

	if s == nil {
		t.Errorf("空切片不应该等于 nil")
	}
	if len(s) != 0 {
		t.Errorf("空切片长度应该是 0")
	}
}

// TestMapBasics 测试 map 基本操作
func TestMapBasics(t *testing.T) {
	m := make(map[string]int)

	// 增
	m["key1"] = 100
	if m["key1"] != 100 {
		t.Errorf("map 写入失败")
	}

	// 查（带 ok）
	if v, ok := m["key1"]; !ok || v != 100 {
		t.Errorf("map 查询失败")
	}

	// 查不存在的键
	if v, ok := m["nonexist"]; ok || v != 0 {
		t.Errorf("查询不存在的键应该返回零值和 ok=false")
	}

	// 改
	m["key1"] = 200
	if m["key1"] != 200 {
		t.Errorf("map 修改失败")
	}

	// 删
	delete(m, "key1")
	if _, ok := m["key1"]; ok {
		t.Errorf("delete 后不应该再找到键")
	}
}

// TestNilMap 测试 nil map
func TestNilMap(t *testing.T) {
	var m map[string]int

	if m != nil {
		t.Errorf("nil map 应该等于 nil")
	}

	// 读 nil map 返回零值
	v := m["key"]
	if v != 0 {
		t.Errorf("读 nil map 应该返回零值")
	}

	// 查询 nil map
	if _, ok := m["key"]; ok {
		t.Errorf("查询 nil map 应该返回 ok=false")
	}

	// 遍历 nil map 不会 panic
	count := 0
	for range m {
		count++
	}
	if count != 0 {
		t.Errorf("遍历 nil map 应该是 0 次迭代")
	}
}

// TestMapLen 测试 map 长度
func TestMapLen(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	if len(m) != 3 {
		t.Errorf("map 长度应该是 3，实际是 %d", len(m))
	}

	delete(m, "b")
	if len(m) != 2 {
		t.Errorf("删除后 map 长度应该是 2，实际是 %d", len(m))
	}
}

// TestFullSliceExpression 测试三下标切片
func TestFullSliceExpression(t *testing.T) {
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// s[2:5:6] -> len=3, cap=4
	sub := s[2:5:6]

	if len(sub) != 3 {
		t.Errorf("sub 长度应该是 3，实际是 %d", len(sub))
	}
	if cap(sub) != 4 {
		t.Errorf("sub 容量应该是 4，实际是 %d", cap(sub))
	}
	if sub[0] != 2 || sub[1] != 3 || sub[2] != 4 {
		t.Errorf("sub 内容不正确: %v", sub)
	}
}

// TestAppendGrow 测试 append 扩容
func TestAppendGrow(t *testing.T) {
	s := make([]int, 0, 2)

	if cap(s) != 2 {
		t.Errorf("初始容量应该是 2")
	}

	s = append(s, 1, 2)
	if len(s) != 2 || cap(s) != 2 {
		t.Errorf("append 两个元素后 len 应该是 2，cap 应该是 2")
	}

	// 触发扩容
	s = append(s, 3)
	if len(s) != 3 {
		t.Errorf("len 应该是 3")
	}
	if cap(s) <= 2 {
		t.Errorf("扩容后 cap 应该大于 2，实际是 %d", cap(s))
	}
}

// BenchmarkAppendPrealloc 对比预分配的性能
func BenchmarkAppendPrealloc(b *testing.B) {
	b.Run("NoPrealloc", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := []int{}
			for j := 0; j < 1000; j++ {
				s = append(s, j)
			}
		}
	})

	b.Run("WithPrealloc", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := make([]int, 0, 1000)
			for j := 0; j < 1000; j++ {
				s = append(s, j)
			}
		}
	})
}

// BenchmarkMapPrealloc 对比 map 预分配的性能
func BenchmarkMapPrealloc(b *testing.B) {
	b.Run("NoPrealloc", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[int]int)
			for j := 0; j < 1000; j++ {
				m[j] = j
			}
		}
	})

	b.Run("WithPrealloc", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[int]int, 1000)
			for j := 0; j < 1000; j++ {
				m[j] = j
			}
		}
	})
}
