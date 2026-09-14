// Package go18_serde_config 演示 Go 的序列化与配置：encoding/json 的标签与嵌套规则、
// 数字精度、Decoder 流式解析、自定义 MarshalJSON，以及 xml / csv / gob、
// 简单的 YAML 子集解析、环境变量、配置分层与敏感信息脱敏。
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go18_serde_config/
package go18_serde_config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// User 演示 json 标签的常见写法。
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"` // 零值时整个字段不输出
	Password string `json:"-"`               // 永远不参与序列化
	Age      int    `json:"age,string"`      // 以 JSON 字符串形式编码数字
	internal string // 未导出字段：不参与序列化
}

// Address 演示嵌套结构。
type Address struct {
	City string `json:"city"`
}

// Account 演示指针、切片与 omitempty 的组合行为。
type Account struct {
	Name    string   `json:"name"`
	Address *Address `json:"address,omitempty"` // nil 指针会被省略
	Tags    []string `json:"tags,omitempty"`    // 空切片同样会被省略
	Empty   []string `json:"empty"`             // 没有 omitempty：nil 切片编码成 null
	Score   *int     `json:"score,omitempty"`   // 指针指向 0 时不算空
}

// Event 是流式解析示例里的一行 NDJSON。
type Event struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Status 是对外表现为字符串的枚举类型。
type Status int

// 枚举取值。
const (
	StatusActive Status = iota
	StatusPaused
	StatusClosed
)

// statusNames 把枚举值映射成对外文案。
var statusNames = map[Status]string{
	StatusActive: "active",
	StatusPaused: "paused",
	StatusClosed: "closed",
}

// MarshalJSON 用值接收者实现，保证值和指针都能正确序列化。
func (s Status) MarshalJSON() ([]byte, error) {
	name, ok := statusNames[s]
	if !ok {
		return nil, fmt.Errorf("unknown status %d", int(s))
	}
	return json.Marshal(name)
}

// UnmarshalJSON 用指针接收者实现，因为要写回调用方的值。
func (s *Status) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	for value, text := range statusNames {
		if text == name {
			*s = value
			return nil
		}
	}
	return fmt.Errorf("unknown status %q", name)
}

// Demo 运行第 18 章的所有示例。
func Demo() {
	fmt.Println("========== go18_serde_config: 序列化与配置 ==========")

	fmt.Println("\n--- 1. json.Marshal / Unmarshal 与标签规则 ---")
	demoJSONBasics()

	fmt.Println("\n--- 2. 嵌套、指针与 omitempty 的真实语义 ---")
	demoNestedAndPointers()

	fmt.Println("\n--- 3. 数字精度：json.Number 与 UseNumber ---")
	demoNumberPrecision()

	fmt.Println("\n--- 4. Decoder：流式解析与严格模式 ---")
	demoDecoder()

	fmt.Println("\n--- 5. 自定义 MarshalJSON / UnmarshalJSON ---")
	demoCustomMarshaler()

	fmt.Println("\n--- 6. encoding/xml、encoding/csv、encoding/gob ---")
	demoOtherFormats()

	fmt.Println("\n--- 7. YAML / TOML：标准库没有，手写一个简单解析器 ---")
	demoSimpleYAML()

	fmt.Println("\n--- 8. 环境变量与默认值 ---")
	demoEnvironment()

	fmt.Println("\n--- 9. 配置分层与优先级 ---")
	demoConfigLayers()

	fmt.Println("\n--- 10. 敏感信息脱敏与选型建议 ---")
	demoRedaction()

	fmt.Println("\n========== 序列化与配置演示结束 ==========")
}

// --- 18.1 json.Marshal / Unmarshal 与标签规则 ---
func demoJSONBasics() {
	u := User{
		ID:       1,
		Name:     "Ada",
		Password: "s3cr3t",
		Age:      36,
		internal: "只在本包可见",
	}

	data, err := json.Marshal(u)
	fmt.Printf("Marshal   = %s，err = %v\n", data, err)

	var back User
	if err := json.Unmarshal(data, &back); err != nil {
		fmt.Println("Unmarshal 失败：", err)
	}
	fmt.Printf("Unmarshal = %+v\n", back)
	fmt.Printf("Password 没有被序列化：%t\n", back.Password == "")

	fmt.Println("标签规则：omitempty 省略零值，- 永远跳过，,string 把数字写成字符串，未导出字段一律跳过。")

	// 类型不匹配时 Unmarshal 会返回错误，而不是 panic。
	var broken User
	err = json.Unmarshal([]byte(`{"age":"abc"}`), &broken)
	fmt.Printf("age 收到非数字字符串：err = %v\n", err)
}

// --- 18.2 嵌套、指针与 omitempty 的真实语义 ---
func demoNestedAndPointers() {
	zero := 0
	cases := []struct {
		name string
		in   Account
	}{
		{"全零值", Account{Name: "Ada"}},
		{"带地址和空切片", Account{Name: "Ada", Address: &Address{City: "上海"}, Tags: []string{}, Empty: []string{}}},
		{"指针指向零值", Account{Name: "Ada", Score: &zero}},
	}

	for _, c := range cases {
		data, err := json.Marshal(c.in)
		fmt.Printf("%-12s → %s（err = %v）\n", c.name, data, err)
	}

	fmt.Println("要点：omitempty 把「nil 指针、空切片、空串、0、false」都当作空；")
	fmt.Println("      没有 omitempty 时，nil 指针和 nil 切片都会输出 null，而不是被省略。")

	// 嵌套结构体里的指针为 nil 时，逐层输出 null。
	type Response struct {
		Code int      `json:"code"`
		Data *Account `json:"data"`
	}
	data, _ := json.Marshal(Response{Code: 200})
	fmt.Printf("嵌套 nil 指针 → %s\n", data)
}

// --- 18.3 数字精度：json.Number 与 UseNumber ---
func demoNumberPrecision() {
	const bigID int64 = 9007199254740993 // 2^53 + 1，超出 float64 能精确表示的整数范围
	payload := fmt.Sprintf(`{"id": %d}`, bigID)
	fmt.Printf("原始 JSON：%s\n", payload)

	var asAny map[string]any
	_ = json.Unmarshal([]byte(payload), &asAny)
	asFloat, _ := asAny["id"].(float64)
	fmt.Printf("解到 any：%v（float64），转回 int64 = %d（精度已经丢了）\n",
		asAny["id"], int64(asFloat))

	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber() // 数字先当成字符串保存，避免中间转成 float64
	var asNumber map[string]any
	_ = decoder.Decode(&asNumber)
	number, _ := asNumber["id"].(json.Number)
	parsed, err := number.Int64()
	fmt.Printf("UseNumber：%v（json.Number），Int64() = %d，err = %v\n", number, parsed, err)

	// 结构化字段不受影响：类型是 int64 时，encoding/json 直接按整数解析。
	var typed struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal([]byte(payload), &typed)
	fmt.Printf("直接解到 int64 字段：%d（精确）\n", typed.ID)
	fmt.Println("结论：金额、雪花 ID、纳秒时间戳这类大整数，要么用 int64 字段，要么开 UseNumber。")
}

// --- 18.4 Decoder：流式解析与严格模式 ---
func demoDecoder() {
	const ndjson = `{"id":1,"name":"first"}
{"id":2,"name":"second"}
{"id":3,"name":"third"}
`
	decoder := json.NewDecoder(strings.NewReader(ndjson))
	for {
		var event Event
		err := decoder.Decode(&event)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Println("解析失败：", err)
			break
		}
		fmt.Printf("读到事件 %d：%s\n", event.ID, event.Name)
	}

	// 严格模式：多余字段直接报错，适合配置文件。
	strict := json.NewDecoder(strings.NewReader(`{"id":1,"name":"a","unknown":true}`))
	strict.DisallowUnknownFields()
	var event Event
	err := strict.Decode(&event)
	fmt.Printf("DisallowUnknownFields → err = %v\n", err)

	fmt.Println("Decoder 面向 io.Reader，适合大文件、HTTP 请求体和一行一个 JSON 的日志。")
}

// --- 18.5 自定义 MarshalJSON / UnmarshalJSON ---
func demoCustomMarshaler() {
	type Job struct {
		Name   string `json:"name"`
		Status Status `json:"status"`
	}

	data, err := json.Marshal(Job{Name: "backup", Status: StatusPaused})
	fmt.Printf("枚举序列化 → %s，err = %v\n", data, err)

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		fmt.Println("反序列化失败：", err)
	}
	fmt.Printf("枚举反序列化 → Status=%v（%d）\n", statusNames[job.Status], job.Status)

	err = json.Unmarshal([]byte(`{"name":"backup","status":"unknown"}`), &job)
	fmt.Printf("未知枚举值 → err = %v\n", err)

	// 值接收者 vs 指针接收者：MarshalJSON 定义在指针上时，值字段不会被特殊处理。
	fmt.Printf("指针接收者的 BadStatus（值字段）→ %s\n", mustJSON(withBadStatus{Status: BadStatus(1)}))
	fmt.Printf("指针接收者的 BadStatus（取地址后）→ %s\n", mustJSON(&withBadStatus{Status: BadStatus(1)}))
	fmt.Println("结论：MarshalJSON 用值接收者，UnmarshalJSON 用指针接收者，这样值和指针都正确。")
}

// BadStatus 故意把 MarshalJSON 定义在指针接收者上，用来演示常见的坑。
type BadStatus int

// MarshalJSON 定义在指针接收者上。
func (s *BadStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal("bad")
}

// withBadStatus 把 BadStatus 作为值字段嵌入。
type withBadStatus struct {
	Status BadStatus `json:"status"`
}

// mustJSON 只用于演示，序列化失败时返回错误文本。
func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "序列化失败: " + err.Error()
	}
	return string(data)
}

// FormatStatus 返回枚举的对外文案，未知值返回 unknown。
func FormatStatus(s Status) string {
	if name, ok := statusNames[s]; ok {
		return name
	}
	return "unknown"
}

// ParseStatus 把对外文案解析回枚举值。
func ParseStatus(text string) (Status, error) {
	for value, name := range statusNames {
		if name == text {
			return value, nil
		}
	}
	return 0, fmt.Errorf("unknown status %q", text)
}

// ParseInt64 解析十进制整数字符串，失败时返回带上下文的错误。
func ParseInt64(field, text string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析 %s=%q: %w", field, text, err)
	}
	return value, nil
}
