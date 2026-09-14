package go18_serde_config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// TestUserJSONTags 验证标签规则：omitempty、-、,string、未导出字段。
func TestUserJSONTags(t *testing.T) {
	u := User{ID: 1, Name: "Ada", Age: 36, Password: "s3cr3t", internal: "hidden"}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal 失败：%v", err)
	}
	if want := `{"id":1,"name":"Ada","age":"36"}`; string(data) != want {
		t.Errorf("Marshal = %s, want %s", data, want)
	}
	if strings.Contains(string(data), "s3cr3t") || strings.Contains(string(data), "hidden") {
		t.Errorf("敏感字段或未导出字段被序列化了：%s", data)
	}

	var back User
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal 失败：%v", err)
	}
	if back.ID != 1 || back.Name != "Ada" || back.Age != 36 {
		t.Errorf("Unmarshal 结果 = %+v", back)
	}
}

// TestUserJSONWrongType 类型不匹配时应该返回错误，而不是 panic。
func TestUserJSONWrongType(t *testing.T) {
	var u User
	err := json.Unmarshal([]byte(`{"age":"abc"}`), &u)
	if err == nil {
		t.Fatal("age 收到非数字字符串时应该报错")
	}
	if !strings.Contains(err.Error(), "User.age") {
		t.Errorf("错误信息应该指出出错字段，得到 %v", err)
	}
}

// TestAccountOmitEmpty 固化 omitempty 与 nil / 空切片的行为。
func TestAccountOmitEmpty(t *testing.T) {
	zero := 0
	cases := []struct {
		name string
		in   Account
		want string
	}{
		{"全零值", Account{Name: "Ada"}, `{"name":"Ada","empty":null}`},
		{
			"带地址与空切片",
			Account{Name: "Ada", Address: &Address{City: "上海"}, Tags: []string{}, Empty: []string{}},
			`{"name":"Ada","address":{"city":"上海"},"empty":[]}`,
		},
		{"指针指向零值不算空", Account{Name: "Ada", Score: &zero}, `{"name":"Ada","empty":null,"score":0}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := json.Marshal(c.in)
			if err != nil {
				t.Fatalf("Marshal 失败：%v", err)
			}
			if string(data) != c.want {
				t.Errorf("Marshal = %s, want %s", data, c.want)
			}
		})
	}
}

// TestNumberPrecision 验证 float64 会丢精度、UseNumber 不会。
func TestNumberPrecision(t *testing.T) {
	const bigID int64 = 9007199254740993
	const payload = `{"id": 9007199254740993}`

	var asAny map[string]any
	if err := json.Unmarshal([]byte(payload), &asAny); err != nil {
		t.Fatalf("解析到 any 失败：%v", err)
	}
	if got := int64(asAny["id"].(float64)); got == bigID {
		t.Errorf("float64 路径居然没丢精度：%d", got)
	}

	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber()
	var asNumber map[string]any
	if err := decoder.Decode(&asNumber); err != nil {
		t.Fatalf("UseNumber 解析失败：%v", err)
	}
	value, err := asNumber["id"].(json.Number).Int64()
	if err != nil {
		t.Fatalf("Int64() 失败：%v", err)
	}
	if value != bigID {
		t.Errorf("UseNumber 后 = %d, want %d", value, bigID)
	}

	var typed struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(payload), &typed); err != nil {
		t.Fatalf("解析到 int64 字段失败：%v", err)
	}
	if typed.ID != bigID {
		t.Errorf("int64 字段 = %d, want %d", typed.ID, bigID)
	}
}

// TestDecoderStream 验证一行一个 JSON 的流式解析。
func TestDecoderStream(t *testing.T) {
	const ndjson = "{\"id\":1,\"name\":\"a\"}\n{\"id\":2,\"name\":\"b\"}\n"
	decoder := json.NewDecoder(strings.NewReader(ndjson))

	var got []Event
	for {
		var event Event
		err := decoder.Decode(&event)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Decode 失败：%v", err)
		}
		got = append(got, event)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].ID != 2 {
		t.Errorf("解析结果 = %+v", got)
	}
}

// TestDecoderDisallowUnknownFields 验证严格模式会拒绝多余字段。
func TestDecoderDisallowUnknownFields(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"id":1,"unknown":true}`))
	decoder.DisallowUnknownFields()

	var event Event
	err := decoder.Decode(&event)
	if err == nil {
		t.Fatal("严格模式下未知字段应该报错")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("错误信息 = %v", err)
	}
}

// TestStatusCustomMarshaler 验证枚举的自定义序列化与错误处理。
func TestStatusCustomMarshaler(t *testing.T) {
	type job struct {
		Status Status `json:"status"`
	}

	data, err := json.Marshal(job{Status: StatusClosed})
	if err != nil {
		t.Fatalf("Marshal 失败：%v", err)
	}
	if string(data) != `{"status":"closed"}` {
		t.Errorf("Marshal = %s", data)
	}

	var back job
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal 失败：%v", err)
	}
	if back.Status != StatusClosed {
		t.Errorf("Status = %d, want %d", back.Status, StatusClosed)
	}

	if err := json.Unmarshal([]byte(`{"status":"unknown"}`), &back); err == nil {
		t.Error("未知枚举值应该报错")
	}
	if _, err := json.Marshal(Status(99)); err == nil {
		t.Error("未知枚举值序列化应该报错")
	}
}

// TestBadStatusPointerReceiver 固化「指针接收者的 MarshalJSON 不生效」这个坑。
func TestBadStatusPointerReceiver(t *testing.T) {
	valueData, err := json.Marshal(withBadStatus{Status: BadStatus(1)})
	if err != nil {
		t.Fatalf("Marshal 失败：%v", err)
	}
	if string(valueData) != `{"status":1}` {
		t.Errorf("值字段 = %s, want {\"status\":1}", valueData)
	}

	pointerData, err := json.Marshal(&withBadStatus{Status: BadStatus(1)})
	if err != nil {
		t.Fatalf("Marshal 失败：%v", err)
	}
	if string(pointerData) != `{"status":"bad"}` {
		t.Errorf("取地址后 = %s, want {\"status\":\"bad\"}", pointerData)
	}
}

// TestParseSimpleYAML 验证手写 YAML 子集解析器。
func TestParseSimpleYAML(t *testing.T) {
	const src = "# 注释\napp:\n  name: \"go-learn\"  # 行尾注释\n  debug: true\n  port: 8080\nrate: 0.5\nempty: null\nlist: [a, b]\n"
	parsed, err := ParseSimpleYAML(src)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}

	app, ok := parsed["app"].(map[string]any)
	if !ok {
		t.Fatalf("app 应该是嵌套 map，得到 %T", parsed["app"])
	}
	if app["name"] != "go-learn" {
		t.Errorf("app.name = %v", app["name"])
	}
	if app["debug"] != true {
		t.Errorf("app.debug = %v", app["debug"])
	}
	if app["port"] != int64(8080) {
		t.Errorf("app.port = %v（%T）", app["port"], app["port"])
	}
	if parsed["rate"] != 0.5 {
		t.Errorf("rate = %v", parsed["rate"])
	}
	if parsed["empty"] != nil {
		t.Errorf("empty = %v, want nil", parsed["empty"])
	}
	if parsed["list"] != "[a, b]" {
		t.Errorf("暂不支持的列表应该原样返回字符串，得到 %v", parsed["list"])
	}
}

// TestParseSimpleYAMLErrors 验证解析错误带行号。
func TestParseSimpleYAMLErrors(t *testing.T) {
	if _, err := ParseSimpleYAML("app:\n  name go-learn\n"); err == nil {
		t.Error("缺少冒号应该报错")
	} else if !strings.Contains(err.Error(), "第 2 行") {
		t.Errorf("错误信息应该带行号，得到 %v", err)
	}

	if _, err := ParseSimpleYAML(": 1\n"); err == nil {
		t.Error("空 key 应该报错")
	}
}

// TestLoadConfigLayers 验证「默认值 → 文件 → 环境变量」的优先级。
func TestLoadConfigLayers(t *testing.T) {
	defaults := DefaultConfig()

	fromDefaults, err := LoadConfig(defaults, "", nil)
	if err != nil {
		t.Fatalf("LoadConfig 失败：%v", err)
	}
	if fromDefaults != defaults {
		t.Errorf("无文件无环境变量时 = %+v, want %+v", fromDefaults, defaults)
	}

	fromFile, err := LoadConfig(defaults, `{"port":9000,"log_level":"debug"}`, nil)
	if err != nil {
		t.Fatalf("LoadConfig 失败：%v", err)
	}
	if fromFile.Port != 9000 || fromFile.LogLevel != "debug" {
		t.Errorf("文件层没生效：%+v", fromFile)
	}
	if fromFile.Host != defaults.Host {
		t.Errorf("文件里没写的字段应该保留默认值，得到 %q", fromFile.Host)
	}

	env := func(key string) (string, bool) {
		values := map[string]string{"APP_HOST": "0.0.0.0", "APP_PORT": "7000"}
		value, ok := values[key]
		return value, ok
	}
	merged, err := LoadConfig(defaults, `{"port":9000}`, env)
	if err != nil {
		t.Fatalf("LoadConfig 失败：%v", err)
	}
	if merged.Port != 7000 {
		t.Errorf("环境变量应该覆盖文件：Port = %d, want 7000", merged.Port)
	}
	if merged.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", merged.Host)
	}
	if merged.LogLevel != defaults.LogLevel {
		t.Errorf("没被覆盖的字段应该保留默认值：LogLevel = %q", merged.LogLevel)
	}
}

// TestLoadConfigErrors 验证两层配置出错时的错误信息。
func TestLoadConfigErrors(t *testing.T) {
	defaults := DefaultConfig()

	if _, err := LoadConfig(defaults, `{"port":"不是数字"}`, nil); err == nil {
		t.Error("配置文件类型错误应该报错")
	} else if !strings.Contains(err.Error(), "解析配置文件") {
		t.Errorf("错误信息应该说明来源，得到 %v", err)
	}

	if _, err := LoadConfig(defaults, `{`, nil); err == nil {
		t.Error("非法 JSON 应该报错")
	}

	if _, err := LoadConfig(defaults, "", func(string) (string, bool) {
		return "abc", true
	}); err == nil {
		t.Error("环境变量类型错误应该报错")
	}
}

// TestMaskSecret 验证脱敏规则。
func TestMaskSecret(t *testing.T) {
	cases := map[string]string{
		"":             "(未设置)",
		"a":            "*",
		"ab":           "a*",
		"s3cr3t-value": "s***********",
		"中文密码":         "中***",
	}
	for in, want := range cases {
		if got := MaskSecret(in); got != want {
			t.Errorf("MaskSecret(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestConfigStringRedacts 保证 String() 不会泄露密码。
func TestConfigStringRedacts(t *testing.T) {
	cfg := Config{Host: "0.0.0.0", Port: 8080, LogLevel: "info", Password: "s3cr3t-value"}
	text := cfg.String()
	if strings.Contains(text, "s3cr3t-value") {
		t.Errorf("String() 泄露了密码：%s", text)
	}
	if !strings.Contains(text, "s***********") {
		t.Errorf("String() 应该包含脱敏后的密码：%s", text)
	}

	data, err := json.Marshal(cfg.Redacted())
	if err != nil {
		t.Fatalf("Marshal 失败：%v", err)
	}
	if strings.Contains(string(data), "s3cr3t-value") {
		t.Errorf("Redacted() 之后仍然泄露密码：%s", data)
	}
}

// ExampleParseSimpleYAML 演示解析结果的结构。
func ExampleParseSimpleYAML() {
	parsed, _ := ParseSimpleYAML("app:\n  port: 8080\n")
	app := parsed["app"].(map[string]any)
	fmt.Println(app["port"])
	// Output: 8080
}

// ExampleMaskSecret 演示脱敏输出。
func ExampleMaskSecret() {
	fmt.Println(MaskSecret("s3cr3t-value"))
	// Output: s***********
}
