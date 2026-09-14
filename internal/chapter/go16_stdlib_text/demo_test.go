package go16_stdlib_text

import (
	"bytes"
	"encoding/csv"
	"html/template"
	"regexp"
	"strings"
	"testing"
	texttemplate "text/template"
	"unicode"
	"unicode/utf8"
)

// TestRegexpCompile 测试正则表达式编译。
func TestRegexpCompile(t *testing.T) {
	tests := []struct {
		pattern string
		valid   bool
	}{
		{`\d+`, true},
		{`[a-z]+`, true},
		{`[`, false}, // 未闭合的字符类
		{`(?P<name>\w+)`, true},
	}

	for _, tt := range tests {
		_, err := regexp.Compile(tt.pattern)
		if tt.valid && err != nil {
			t.Errorf("期望 %q 编译成功，实际失败：%v", tt.pattern, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("期望 %q 编译失败，实际成功", tt.pattern)
		}
	}
}

// TestRegexpMatch 测试正则匹配。
func TestRegexpMatch(t *testing.T) {
	re := regexp.MustCompile(`\d{3}-\d{4}`)

	tests := []struct {
		input string
		match bool
	}{
		{"Call 123-4567", true},
		{"No phone", false},
		{"999-0000", true},
		{"123-456", false}, // 只有 3 位
	}

	for _, tt := range tests {
		got := re.MatchString(tt.input)
		if got != tt.match {
			t.Errorf("MatchString(%q) = %v, 期望 %v", tt.input, got, tt.match)
		}
	}
}

// TestRegexpFindAll 测试查找所有匹配。
func TestRegexpFindAll(t *testing.T) {
	re := regexp.MustCompile(`\d+`)
	text := "Order 123 costs $456 in 2024"

	matches := re.FindAllString(text, -1)
	expected := []string{"123", "456", "2024"}

	if len(matches) != len(expected) {
		t.Fatalf("期望 %d 个匹配，实际 %d 个", len(expected), len(matches))
	}

	for i := range matches {
		if matches[i] != expected[i] {
			t.Errorf("匹配[%d] = %q, 期望 %q", i, matches[i], expected[i])
		}
	}
}

// TestRegexpNamedCapture 测试命名捕获组。
func TestRegexpNamedCapture(t *testing.T) {
	re := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
	text := "2024-09-14"

	match := re.FindStringSubmatch(text)
	if match == nil {
		t.Fatal("未匹配")
	}

	names := re.SubexpNames()
	result := make(map[string]string)
	for i, name := range names {
		if i > 0 && i < len(match) {
			result[name] = match[i]
		}
	}

	expected := map[string]string{
		"year":  "2024",
		"month": "09",
		"day":   "14",
	}

	for k, v := range expected {
		if result[k] != v {
			t.Errorf("捕获组 %s = %q, 期望 %q", k, result[k], v)
		}
	}
}

// TestRegexpGreedy 测试贪婪与非贪婪。
func TestRegexpGreedy(t *testing.T) {
	text := `<div>Hello</div><div>World</div>`

	// 贪婪
	greedy := regexp.MustCompile(`<div>.*</div>`)
	greedyMatch := greedy.FindString(text)
	if greedyMatch != text {
		t.Errorf("贪婪匹配 = %q, 期望 %q", greedyMatch, text)
	}

	// 非贪婪
	nonGreedy := regexp.MustCompile(`<div>.*?</div>`)
	nonGreedyMatches := nonGreedy.FindAllString(text, -1)
	expected := []string{"<div>Hello</div>", "<div>World</div>"}

	if len(nonGreedyMatches) != len(expected) {
		t.Fatalf("非贪婪匹配数量 = %d, 期望 %d", len(nonGreedyMatches), len(expected))
	}

	for i := range nonGreedyMatches {
		if nonGreedyMatches[i] != expected[i] {
			t.Errorf("非贪婪匹配[%d] = %q, 期望 %q", i, nonGreedyMatches[i], expected[i])
		}
	}
}

// TestTextTemplate 测试 text/template。
func TestTextTemplate(t *testing.T) {
	tmplStr := `Hello, {{.Name}}! Age: {{.Age}}`
	tmpl, err := texttemplate.New("test").Parse(tmplStr)
	if err != nil {
		t.Fatalf("解析模板失败：%v", err)
	}

	data := struct {
		Name string
		Age  int
	}{Name: "Alice", Age: 30}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("执行模板失败：%v", err)
	}

	expected := "Hello, Alice! Age: 30"
	if buf.String() != expected {
		t.Errorf("模板输出 = %q, 期望 %q", buf.String(), expected)
	}
}

// TestHTMLTemplateEscape 测试 html/template 自动转义。
func TestHTMLTemplateEscape(t *testing.T) {
	tmplStr := `<div>{{.Content}}</div>`
	tmpl, _ := template.New("html").Parse(tmplStr)

	data := struct{ Content string }{Content: "<script>alert('XSS')</script>"}

	var buf bytes.Buffer
	tmpl.Execute(&buf, data)

	// html/template 会转义 < > 等字符
	if strings.Contains(buf.String(), "<script>") {
		t.Errorf("html/template 未转义：%s", buf.String())
	}

	expected := "<div>&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;</div>"
	if buf.String() != expected {
		t.Errorf("转义结果 = %q, 期望 %q", buf.String(), expected)
	}
}

// TestUTF8RuneCount 测试 UTF-8 字符计数。
func TestUTF8RuneCount(t *testing.T) {
	tests := []struct {
		s         string
		byteCount int
		runeCount int
	}{
		{"Go", 2, 2},
		{"Go语言", 8, 4},     // 每个汉字 3 字节
		{"😀", 4, 1},        // emoji 4 字节
		{"Hello世界", 11, 7}, // 5 + 6
	}

	for _, tt := range tests {
		if len(tt.s) != tt.byteCount {
			t.Errorf("len(%q) = %d, 期望 %d", tt.s, len(tt.s), tt.byteCount)
		}
		if utf8.RuneCountInString(tt.s) != tt.runeCount {
			t.Errorf("RuneCountInString(%q) = %d, 期望 %d",
				tt.s, utf8.RuneCountInString(tt.s), tt.runeCount)
		}
	}
}

// TestUnicodeCategory 测试 unicode 字符分类。
func TestUnicodeCategory(t *testing.T) {
	tests := []struct {
		r      rune
		letter bool
		digit  bool
		space  bool
	}{
		{'A', true, false, false},
		{'5', false, true, false},
		{' ', false, false, true},
		{'中', true, false, false},
		{'!', false, false, false},
	}

	for _, tt := range tests {
		if unicode.IsLetter(tt.r) != tt.letter {
			t.Errorf("IsLetter(%c) = %v, 期望 %v", tt.r, unicode.IsLetter(tt.r), tt.letter)
		}
		if unicode.IsDigit(tt.r) != tt.digit {
			t.Errorf("IsDigit(%c) = %v, 期望 %v", tt.r, unicode.IsDigit(tt.r), tt.digit)
		}
		if unicode.IsSpace(tt.r) != tt.space {
			t.Errorf("IsSpace(%c) = %v, 期望 %v", tt.r, unicode.IsSpace(tt.r), tt.space)
		}
	}
}

// TestCSVReadWrite 测试 CSV 读写。
func TestCSVReadWrite(t *testing.T) {
	// 写入
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	records := [][]string{
		{"Name", "Age"},
		{"Alice", "30"},
		{"Bob", "25"},
	}
	for _, rec := range records {
		if err := w.Write(rec); err != nil {
			t.Fatalf("写入 CSV 失败：%v", err)
		}
	}
	w.Flush()

	// 读取
	r := csv.NewReader(&buf)
	readRecords, err := r.ReadAll()
	if err != nil {
		t.Fatalf("读取 CSV 失败：%v", err)
	}

	if len(readRecords) != len(records) {
		t.Fatalf("记录数量 = %d, 期望 %d", len(readRecords), len(records))
	}

	for i := range records {
		for j := range records[i] {
			if readRecords[i][j] != records[i][j] {
				t.Errorf("记录[%d][%d] = %q, 期望 %q",
					i, j, readRecords[i][j], records[i][j])
			}
		}
	}
}

// TestReplacer 测试 strings.Replacer。
func TestReplacer(t *testing.T) {
	replacer := strings.NewReplacer(
		"apple", "🍎",
		"banana", "🍌",
	)

	input := "I like apple and banana"
	output := replacer.Replace(input)
	expected := "I like 🍎 and 🍌"

	if output != expected {
		t.Errorf("Replacer.Replace(%q) = %q, 期望 %q", input, output, expected)
	}
}

// TestRegexpSplit 测试正则分割。
func TestRegexpSplit(t *testing.T) {
	re := regexp.MustCompile(`\s+`)
	text := "Go   is    awesome"

	parts := re.Split(text, -1)
	expected := []string{"Go", "is", "awesome"}

	if len(parts) != len(expected) {
		t.Fatalf("切分数量 = %d, 期望 %d", len(parts), len(expected))
	}

	for i := range parts {
		if parts[i] != expected[i] {
			t.Errorf("parts[%d] = %q, 期望 %q", i, parts[i], expected[i])
		}
	}
}

// TestTemplateRange 测试模板 range 遍历。
func TestTemplateRange(t *testing.T) {
	tmplStr := `{{range .Items}}- {{.}}
{{end}}`
	tmpl, _ := texttemplate.New("range").Parse(tmplStr)

	data := struct{ Items []string }{Items: []string{"A", "B", "C"}}

	var buf bytes.Buffer
	tmpl.Execute(&buf, data)

	expected := "- A\n- B\n- C\n"
	if buf.String() != expected {
		t.Errorf("模板输出 = %q, 期望 %q", buf.String(), expected)
	}
}
