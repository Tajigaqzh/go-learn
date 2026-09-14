package go08_strings

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncateBytes 验证截断点会回退到字符边界。
func TestTruncateBytes(t *testing.T) {
	s := "Go 学习笔记" // G o 空格 + 学 习 笔 记 = 3 + 4*3 = 15 字节
	cases := []struct {
		name     string
		maxBytes int
		want     string
	}{
		{"负数返回空串", -1, ""},
		{"零字节返回空串", 0, ""},
		{"只够 ASCII", 2, "Go"},
		{"切点落在多字节字符中间要回退", 4, "Go "},
		{"切点正好在字符边界", 6, "Go 学"},
		{"超出长度返回原串", 100, s},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TruncateBytes(s, c.maxBytes); got != c.want {
				t.Errorf("TruncateBytes(%q, %d) = %q, want %q", s, c.maxBytes, got, c.want)
			}
		})
	}
}

// TestTruncateBytesAlwaysValidUTF8 保证任意长度截断都不会产出非法 UTF-8。
func TestTruncateBytesAlwaysValidUTF8(t *testing.T) {
	s := "Go 学习 🚀!"
	for n := 0; n <= len(s)+1; n++ {
		got := TruncateBytes(s, n)
		if !utf8.ValidString(got) {
			t.Fatalf("TruncateBytes(%q, %d) = %q，不是合法 UTF-8", s, n, got)
		}
		if len(got) > max(n, 0) {
			t.Fatalf("TruncateBytes(%q, %d) 的结果长度 %d 超过上限", s, n, len(got))
		}
		if !strings.HasPrefix(s, got) {
			t.Fatalf("TruncateBytes(%q, %d) = %q，不是原串的前缀", s, n, got)
		}
	}
}

// TestReverseRunes 验证按字符反转时多字节字符不会被拆坏。
func TestReverseRunes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"abc", "cba"},
		{"学习", "习学"},
		{"Go 学", "学 oG"},
		{"", ""},
		{"🚀中a", "a中🚀"},
	}
	for _, c := range cases {
		if got := ReverseRunes(c.in); got != c.want {
			t.Errorf("ReverseRunes(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestReverseRunesTwice 验证反转两次能还原，且中间结果始终是合法 UTF-8。
func TestReverseRunesTwice(t *testing.T) {
	s := "Go 学习 🚀"
	once := ReverseRunes(s)
	if !utf8.ValidString(once) {
		t.Fatalf("ReverseRunes(%q) = %q，不是合法 UTF-8", s, once)
	}
	if got := ReverseRunes(once); got != s {
		t.Errorf("反转两次 = %q, want %q", got, s)
	}
}

// TestTruncateRunes 验证按字符数截断，且结果是合法 UTF-8。
func TestTruncateRunes(t *testing.T) {
	s := "Go 学习笔记"
	cases := []struct {
		name     string
		maxRunes int
		want     string
	}{
		{"负数返回空串", -1, ""},
		{"零个字符返回空串", 0, ""},
		{"取前 2 个字符", 2, "Go"},
		{"取前 4 个字符", 4, "Go 学"},
		{"超过总字符数返回原串", 99, s},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := TruncateRunes(s, c.maxRunes)
			if got != c.want {
				t.Errorf("TruncateRunes(%q, %d) = %q, want %q", s, c.maxRunes, got, c.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("TruncateRunes(%q, %d) = %q，不是合法 UTF-8", s, c.maxRunes, got)
			}
		})
	}
}

// TestTruncateRunesMatchesRuneSlice 与「先转 []rune 再切」的结果保持一致。
func TestTruncateRunesMatchesRuneSlice(t *testing.T) {
	s := "Go 学习 🚀!"
	runes := []rune(s)
	for n := 0; n <= len(runes)+1; n++ {
		want := string(runes[:min(n, len(runes))])
		if got := TruncateRunes(s, n); got != want {
			t.Errorf("TruncateRunes(%q, %d) = %q, want %q", s, n, got, want)
		}
	}
}

// TestConcatStrategiesAgree 四种拼接写法必须产出同一个字符串。
func TestConcatStrategiesAgree(t *testing.T) {
	parts := []string{"Go", " ", "学习", " ", "🚀"}
	want := "Go 学习 🚀"
	got := map[string]string{
		"+":               concatWithPlus(parts),
		"strings.Join":    strings.Join(parts, ""),
		"strings.Builder": concatWithBuilder(parts),
		"bytes.Buffer":    concatWithBuffer(parts),
	}
	for name, v := range got {
		if v != want {
			t.Errorf("%s 拼接结果 = %q, want %q", name, v, want)
		}
	}
}

// TestConcatStrategiesEmpty 空输入不应产生意外结果。
func TestConcatStrategiesEmpty(t *testing.T) {
	var parts []string
	if got := concatWithBuilder(parts); got != "" {
		t.Errorf("concatWithBuilder(nil) = %q, want 空串", got)
	}
	if got := concatWithPlus(parts); got != "" {
		t.Errorf("concatWithPlus(nil) = %q, want 空串", got)
	}
}

// TestStringSlicingKeepsValidUTF8 对照按字节切分与安全截断的差别。
func TestStringSlicingKeepsValidUTF8(t *testing.T) {
	s := "Go 学习"
	broken := s[4:6]
	if utf8.ValidString(broken) {
		t.Fatalf("s[4:6] 的字节 % X 预期是非法 UTF-8", []byte(broken))
	}
	got := TruncateBytes(s, 6)
	if got != "Go 学" || !utf8.ValidString(got) {
		t.Errorf("TruncateBytes(%q, 6) = %q, want %q 且合法", s, got, "Go 学")
	}
}

// TestTruncateOnChineseBoundary 覆盖文档里「切点落在第二个汉字中间」的用例。
func TestTruncateOnChineseBoundary(t *testing.T) {
	s := "学习" // 6 字节、2 个字符
	if got := TruncateBytes(s, 4); got != "学" {
		t.Errorf("TruncateBytes(%q, 4) = %q, want %q", s, got, "学")
	}
	if got := TruncateRunes(s, 7); got != s {
		t.Errorf("TruncateRunes(%q, 7) = %q, want %q（字符数不够时原样返回）", s, got, s)
	}
}

// TestRuneStringEquivalents 验证同一码点的四种写法得到同一个字符串。
func TestRuneStringEquivalents(t *testing.T) {
	cases := map[string]string{
		"字面量":      "中",
		"\\u 转义":   "\u4e2d",
		"UTF-8 字节": string([]byte{0xE4, 0xB8, 0xAD}),
		"rune 转换":  string(rune(20013)),
	}
	for name, got := range cases {
		if got != "中" {
			t.Errorf("%s 写法 = %q, want 中", name, got)
		}
		if len(got) != 3 {
			t.Errorf("%s 写法的字节数 = %d, want 3", name, len(got))
		}
	}
}

// TestRuneCountDiffersFromByteLen 用中文强调 len 数的是字节。
func TestRuneCountDiffersFromByteLen(t *testing.T) {
	s := "中文"
	if len(s) != 6 {
		t.Errorf("len(%q) = %d, want 6", s, len(s))
	}
	if n := utf8.RuneCountInString(s); n != 2 {
		t.Errorf("RuneCountInString(%q) = %d, want 2", s, n)
	}
}

// TestChapterMetadata 保证章节编号和标题与文档里的章节名一致。
func TestChapterMetadata(t *testing.T) {
	if Chapter != 8 {
		t.Errorf("Chapter = %d, want 8", Chapter)
	}
	if ChapterTitle != "字符串、字节与 Unicode" {
		t.Errorf("ChapterTitle = %q, want 字符串、字节与 Unicode", ChapterTitle)
	}
}

// ExampleTruncateBytes 演示安全截断的输出。
func ExampleTruncateBytes() {
	fmt.Printf("%q\n", TruncateBytes("Go 学习笔记", 6))
	// Output: "Go 学"
}

// ExampleReverseRunes 演示按字符反转的输出。
func ExampleReverseRunes() {
	fmt.Println(ReverseRunes("学习 Go"))
	// Output: oG 习学
}
