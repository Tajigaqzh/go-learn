package go18_serde_config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config 是应用配置，字段标签与配置文件里的 key 一一对应。
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LogLevel string `json:"log_level"`
	Password string `json:"password"`
}

// DefaultConfig 返回内置默认值，作为配置分层的最底层。
func DefaultConfig() Config {
	return Config{Host: "localhost", Port: 8080, LogLevel: "info"}
}

// EnvLookup 抽象环境变量查询：生产传 os.LookupEnv，测试传假实现。
type EnvLookup func(key string) (string, bool)

// EnvPrefix 是所有环境变量的前缀。
const EnvPrefix = "APP_"

// LoadConfig 按「默认值 → 配置文件 → 环境变量」的顺序合并配置。
//
// 每一层只覆盖自己提供过的字段，所以文件里漏写的项会自动保留默认值；
// 环境变量优先级最高，适合覆盖部署相关的值。
func LoadConfig(defaults Config, fileJSON string, lookup EnvLookup) (Config, error) {
	cfg := defaults

	if strings.TrimSpace(fileJSON) != "" {
		if err := json.Unmarshal([]byte(fileJSON), &cfg); err != nil {
			return Config{}, fmt.Errorf("解析配置文件: %w", err)
		}
	}

	if lookup == nil {
		return cfg, nil
	}
	if value, ok := lookup(EnvPrefix + "HOST"); ok {
		cfg.Host = value
	}
	if value, ok := lookup(EnvPrefix + "PORT"); ok {
		port, err := ParseInt64("APP_PORT", value)
		if err != nil {
			return Config{}, err
		}
		cfg.Port = int(port)
	}
	if value, ok := lookup(EnvPrefix + "LOG_LEVEL"); ok {
		cfg.LogLevel = value
	}
	if value, ok := lookup(EnvPrefix + "PASSWORD"); ok {
		cfg.Password = value
	}
	return cfg, nil
}

// String 让 Config 在打印时自动脱敏，避免密码进日志。
func (c Config) String() string {
	return fmt.Sprintf("Config{Host:%q Port:%d LogLevel:%q Password:%s}",
		c.Host, c.Port, c.LogLevel, MaskSecret(c.Password))
}

// MaskSecret 保留首字符，其余用 * 代替；空串返回「(未设置)」。
func MaskSecret(secret string) string {
	if secret == "" {
		return "(未设置)"
	}
	runes := []rune(secret)
	if len(runes) == 1 {
		return "*"
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

// LogView 是专供日志与审计输出的脱敏结构体。
type LogView struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LogLevel string `json:"log_level"`
	Password string `json:"password"`
}

// Redacted 返回可以安全写进日志的副本。
func (c Config) Redacted() LogView {
	return LogView{
		Host:     c.Host,
		Port:     c.Port,
		LogLevel: c.LogLevel,
		Password: MaskSecret(c.Password),
	}
}

// --- 18.8 环境变量与默认值 ---
func demoEnvironment() {
	// 演示里自己设置，保证每次运行的输出一致；生产代码不会这么写。
	_ = os.Setenv(EnvPrefix+"PORT", "8080")

	port, ok := os.LookupEnv(EnvPrefix + "PORT")
	fmt.Printf("LookupEnv(APP_PORT) = %q，ok = %t\n", port, ok)

	_, ok = os.LookupEnv(EnvPrefix + "MISSING")
	fmt.Printf("LookupEnv(APP_MISSING) → ok = %t\n", ok)
	fmt.Printf("Getenv(APP_MISSING) = %q（只调 Getenv 分不清「空值」和「没设置」）\n",
		os.Getenv(EnvPrefix+"MISSING"))

	parsed, err := ParseInt64("APP_PORT", port)
	fmt.Printf("解析成整数：%d，err = %v\n", parsed, err)

	_, err = ParseInt64("APP_PORT", "abc")
	fmt.Printf("解析非法值 → err = %v\n", err)

	fmt.Println("惯例：用 LookupEnv 判断「有没有设置」，解析失败直接报错，别用零值当默认值糊过去。")
}

// --- 18.9 配置分层与优先级 ---
func demoConfigLayers() {
	defaults := DefaultConfig()
	fmt.Printf("① 默认值        %v\n", defaults)

	const fileJSON = `{"host":"0.0.0.0","port":9000,"log_level":"debug","password":"file-secret"}`
	fromFile, err := LoadConfig(defaults, fileJSON, nil)
	fmt.Printf("② 叠加配置文件  %v，err = %v\n", fromFile, err)

	env := func(key string) (string, bool) {
		values := map[string]string{"APP_PORT": "7000", "APP_PASSWORD": "env-secret"}
		value, ok := values[key]
		return value, ok
	}
	merged, err := LoadConfig(defaults, fileJSON, env)
	fmt.Printf("③ 叠加环境变量  %v，err = %v\n", merged, err)

	partial, err := LoadConfig(defaults, `{"port":9000}`, nil)
	fmt.Printf("文件只写一个字段，其余保留默认值：%v，err = %v\n", partial, err)

	_, err = LoadConfig(defaults, `{"port":"不是数字"}`, nil)
	fmt.Printf("配置文件类型写错 → err = %v\n", err)

	_, err = LoadConfig(defaults, `{"port": 1}`, func(key string) (string, bool) {
		if key == EnvPrefix+"PORT" {
			return "不是数字", true
		}
		return "", false
	})
	fmt.Printf("环境变量类型写错 → err = %v\n", err)
}

// --- 18.10 敏感信息脱敏与格式选型 ---
func demoRedaction() {
	cfg := Config{Host: "0.0.0.0", Port: 8080, LogLevel: "info", Password: "s3cr3t-value"}

	fmt.Printf("打印配置（自动走 String()）：%v\n", cfg)
	fmt.Printf("字符串长度：Password 原值 %d 个字符，脱敏后 %q\n",
		len(cfg.Password), MaskSecret(cfg.Password))

	view, err := json.Marshal(cfg.Redacted())
	fmt.Printf("Redacted() 之后序列化：%s，err = %v\n", view, err)

	raw, err := json.Marshal(cfg)
	fmt.Printf("直接序列化 Config：%s，err = %v\n", raw, err)
	fmt.Println("Config 没有实现 MarshalJSON，所以直接序列化会把密码原样写出去——")
	fmt.Println("日志、审计、暴露给前端的结构体，一律用 Redacted() 的结果。")

	fmt.Println("\n格式选型：")
	fmt.Println("  对外 HTTP 接口     encoding/json（标准、可读、跨语言）")
	fmt.Println("  本地配置文件       YAML / TOML（人类友好，用第三方库）+ 环境变量覆盖")
	fmt.Println("  Go 进程之间        gob（带类型信息，启动有注册开销，不能跨语言）")
	fmt.Println("  跨语言高性能       protobuf（本章范围外）")
}
