package go31_unsafe_cgo

import (
	"testing"
	"unsafe"
)

// TestUnsafeOffsetAccess 验证 31.2 的结论：用 uintptr(unsafe.Pointer(&p)) + Offsetof
// 能精确定位并读写结构体的某个字段，且不碰相邻字段。
// 注意：uintptr -> unsafe.Pointer 必须在一个表达式里完成，拆成两步、把 uintptr
// 存到本地变量再转回指针是不安全的（GC 可能移动对象），这正是正文强调的坑。
func TestUnsafeOffsetAccess(t *testing.T) {
	type Point struct{ x, y int32 }
	p := Point{x: 10, y: 20}

	py := (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&p)) + unsafe.Offsetof(p.y)))
	if *py != 20 {
		t.Fatalf("通过偏移读 y 应为 20，得到 %d", *py)
	}
	*py = 30
	if p.y != 30 || p.x != 10 {
		t.Fatalf("写 y 后应 y=30 且 x 不变，得到 x=%d y=%d", p.x, p.y)
	}
}

// TestUnsafeSizeofInvariants 验证 31.1 讲的内存对齐规律：结构体大小不小于各字段
// 大小之和，且每个字段的偏移量都对齐到它的对齐要求。
// 具体字节数随平台/指针宽度变化，所以这里只断言「不变量」而非硬编码数字。
func TestUnsafeSizeofInvariants(t *testing.T) {
	type Example struct {
		a bool
		b int32
		c int64
		d string
	}
	var e Example

	sum := unsafe.Sizeof(e.a) + unsafe.Sizeof(e.b) + unsafe.Sizeof(e.c) + unsafe.Sizeof(e.d)
	if unsafe.Sizeof(e) < sum {
		t.Fatalf("Sizeof(%d) 不应小于字段之和(%d)", unsafe.Sizeof(e), sum)
	}
	if unsafe.Offsetof(e.b)%unsafe.Alignof(e.b) != 0 {
		t.Fatalf("字段 b 偏移 %d 未对齐到 %d", unsafe.Offsetof(e.b), unsafe.Alignof(e.b))
	}
	if unsafe.Offsetof(e.c)%unsafe.Alignof(e.c) != 0 {
		t.Fatalf("字段 c 偏移 %d 未对齐到 %d", unsafe.Offsetof(e.c), unsafe.Alignof(e.c))
	}
}
