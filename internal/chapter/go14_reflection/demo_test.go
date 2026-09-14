package go14_reflection

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestDescribe 验证「类型/种类」的描述格式。
func TestDescribe(t *testing.T) {
	type Celsius float64
	cases := []struct {
		in   any
		want string
	}{
		{42, "int/int"},
		{"text", "string/string"},
		{Celsius(1), "go14_reflection.Celsius/float64"},
		{[]int{1}, "[]int/slice"},
		{nil, "<nil>/invalid"},
	}
	for _, c := range cases {
		if got := Describe(c.in); got != c.want {
			t.Errorf("Describe(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestParseTag 验证结构体标签解析。
func TestParseTag(t *testing.T) {
	cases := []struct {
		tag         string
		wantName    string
		wantOptions []string
	}{
		{"", "", nil},
		{"id", "id", nil},
		{"nickname,omitempty", "nickname", []string{"omitempty"}},
		{"balance,omitempty,string", "balance", []string{"omitempty", "string"}},
		{"-,", "-", nil},
	}
	for _, c := range cases {
		name, options := ParseTag(c.tag)
		if name != c.wantName {
			t.Errorf("ParseTag(%q) name = %q, want %q", c.tag, name, c.wantName)
		}
		if strings.Join(options, ",") != strings.Join(c.wantOptions, ",") {
			t.Errorf("ParseTag(%q) options = %v, want %v", c.tag, options, c.wantOptions)
		}
	}
}

// TestJSONFieldNames 验证按标签推导输出字段名，并跳过未导出字段。
func TestJSONFieldNames(t *testing.T) {
	type record struct {
		ID      int `json:"id"`
		Name    string
		Skipped string `json:"-"`
		hidden  string
	}

	got, err := JSONFieldNames(record{})
	if err != nil {
		t.Fatalf("JSONFieldNames(record{}) 返回错误：%v", err)
	}
	if want := []string{"id", "Name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("JSONFieldNames = %v, want %v", got, want)
	}

	// 指针也要能处理。
	if _, err := JSONFieldNames(&record{}); err != nil {
		t.Errorf("JSONFieldNames(*record) 返回错误：%v", err)
	}
	if _, err := JSONFieldNames(42); err == nil {
		t.Error("JSONFieldNames(42) 应该返回错误")
	}
	var nilRecord *record
	if _, err := JSONFieldNames(nilRecord); err != nil {
		t.Errorf("JSONFieldNames(nil 指针) 返回错误：%v", err)
	}
}

// TestSetInt64Field 验证反射改值的各种边界。
func TestSetInt64Field(t *testing.T) {
	t.Run("成功设置", func(t *testing.T) {
		p := &Person{Name: "Alice", Age: 1}
		if err := SetInt64Field(p, "Age", 30); err != nil {
			t.Fatalf("SetInt64Field 返回错误：%v", err)
		}
		if p.Age != 30 {
			t.Errorf("Age = %d, want 30", p.Age)
		}
	})

	t.Run("传值必须报错", func(t *testing.T) {
		if err := SetInt64Field(Person{}, "Age", 30); err == nil {
			t.Error("传结构体值应该报「需要指针」")
		}
	})

	t.Run("nil 指针必须报错", func(t *testing.T) {
		var p *Person
		if err := SetInt64Field(p, "Age", 30); err == nil {
			t.Error("nil 指针应该报错，而不是 panic")
		}
	})

	t.Run("未导出字段必须报错", func(t *testing.T) {
		if err := SetInt64Field(&Person{note: "x"}, "note", 30); err == nil {
			t.Error("未导出字段应该报错")
		}
	})

	t.Run("不存在的字段必须报错", func(t *testing.T) {
		if err := SetInt64Field(&Person{}, "Email", 30); err == nil {
			t.Error("不存在的字段应该报错")
		}
	})

	t.Run("类型不匹配必须报错", func(t *testing.T) {
		if err := SetInt64Field(&Person{}, "Name", 30); err == nil {
			t.Error("给字符串字段设置整数应该报错")
		}
	})
}

// TestCallByName 验证动态调用的正常路径与三种错误。
func TestCallByName(t *testing.T) {
	calc := &Calculator{}

	got, err := CallByName(calc, "Add", 2, 3)
	if err != nil || len(got) != 1 || got[0] != 5 {
		t.Fatalf("CallByName(Add, 2, 3) = %v, %v，want [5], nil", got, err)
	}

	got, err = CallByName(calc, "Divide", 10.0, 4.0)
	if err != nil || len(got) != 2 || got[0] != 2.5 || got[1] != nil {
		t.Fatalf("CallByName(Divide, 10, 4) = %v, %v，want [2.5 <nil>], nil", got, err)
	}

	got, err = CallByName(calc, "Divide", 10.0, 0.0)
	if err != nil || len(got) != 2 || got[1] == nil {
		t.Fatalf("CallByName(Divide, 10, 0) 应该把方法返回的错误原样带回来，得到 %v, %v", got, err)
	}

	if _, err := CallByName(calc, "Missing"); err == nil {
		t.Error("调用不存在的方法应该报错")
	}
	if _, err := CallByName(calc, "Add", 1); err == nil {
		t.Error("参数个数不匹配应该报错")
	}
	if _, err := CallByName(calc, "Add", 1, "x"); err == nil {
		t.Error("参数类型不匹配应该报错")
	}
	if _, err := CallByName(nil, "Add", 1, 2); err == nil {
		t.Error("接收者为 nil 应该报错")
	}
}

// TestUnmarshalInto 验证手写的反射反序列化同样要求指针。
func TestUnmarshalInto(t *testing.T) {
	p := Person{}
	err := UnmarshalInto(&p, map[string]any{"Name": "Eve", "Age": 25, "note": "被跳过"})
	if err != nil {
		t.Fatalf("UnmarshalInto 返回错误：%v", err)
	}
	if p.Name != "Eve" || p.Age != 25 {
		t.Errorf("UnmarshalInto 结果 = %+v, want Name=Eve Age=25", p)
	}
	if p.note != "" {
		t.Errorf("未导出字段不应该被写入，得到 %q", p.note)
	}

	if err := UnmarshalInto(p, map[string]any{"Name": "Eve"}); err == nil {
		t.Error("传值调用应该报错")
	}
}

// TestSimpleJSONMatchesEncodingJSON 手写实现要和标准库输出一致。
func TestSimpleJSONMatchesEncodingJSON(t *testing.T) {
	type inner struct {
		Flag bool `json:"flag"`
	}
	type account struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Balance  int    `json:"balance,omitempty"`
		Nick     string `json:"nickname,omitempty"`
		Skipped  string `json:"-"`
		hidden   string
		Tags     []string          `json:"tags"`
		Scores   map[string]int    `json:"scores"`
		Detail   inner             `json:"detail"`
		Extra    *string           `json:"extra"`
		NilSlice []int             `json:"nil_slice"`
		EmptyMap map[string]string `json:"empty_map"`
	}

	cases := map[string]any{
		"带零值和未导出字段":  account{ID: 1, Name: "Ada", hidden: "secret"},
		"含切片和 map":   account{ID: 2, Tags: []string{"a", "b"}, Scores: map[string]int{"math": 90, "art": 85}},
		"嵌套与自己赋值的指针": account{ID: 3, Detail: inner{Flag: true}, Extra: ptr("x")},
		"空切片与空 map":  account{ID: 4, Tags: []string{}, Scores: map[string]int{}, EmptyMap: map[string]string{}},
		"基础类型":       []any{42, "text", true, 3.5, nil},
	}

	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			std, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("json.Marshal 失败：%v", err)
			}
			mine, err := SimpleJSON(v)
			if err != nil {
				t.Fatalf("SimpleJSON 失败：%v", err)
			}
			if mine != string(std) {
				t.Errorf("SimpleJSON = %s, json.Marshal = %s", mine, std)
			}
		})
	}
}

// TestSimpleJSONUnsupported 不支持的类型要返回错误而不是 panic。
func TestSimpleJSONUnsupported(t *testing.T) {
	_, err := SimpleJSON(make(chan int))
	if err == nil {
		t.Fatal("通道类型应该返回错误")
	}
	if !strings.Contains(err.Error(), "不支持") {
		t.Errorf("错误信息应该说明原因，得到 %v", err)
	}
}

// TestDeepEqualBoundaries 固化 14.8 里那些反直觉的结论。
func TestDeepEqualBoundaries(t *testing.T) {
	var nilSlice []int
	if reflect.DeepEqual(nilSlice, []int{}) {
		t.Error("nil 切片与空切片不应该 DeepEqual")
	}
	if !reflect.DeepEqual([]int{1}, []int{1}) {
		t.Error("同内容切片应该 DeepEqual")
	}
	if reflect.DeepEqual(1.0, 1) {
		t.Error("不同类型不应该 DeepEqual")
	}
}

// ptr 是测试里构造指针的小工具。
func ptr[T any](v T) *T {
	return &v
}

// ExampleJSONFieldNames 演示字段名解析的结果。
func ExampleJSONFieldNames() {
	type account struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Password string `json:"-"`
		note     string
	}
	names, err := JSONFieldNames(account{})
	fmt.Println(names, err)
	// Output: [id name] <nil>
}

// ExampleParseTag 演示标签解析。
func ExampleParseTag() {
	name, options := ParseTag("balance,omitempty,string")
	fmt.Printf("%s %v\n", name, options)
	// Output: balance [omitempty string]
}
