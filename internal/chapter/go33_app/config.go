package go33_app

import "os"

// Config 是运行配置：给 main 提供监听地址、数据库连接和日志级别。
type Config struct {
	Addr     string // HTTP 监听地址
	DSN      string // 数据库连接串（演示默认内存库）
	LogLevel string // debug / info / warn / error
}

// LoadConfig 用 os.LookupEnv 加载配置，等价于 LoadConfigFrom(os.LookupEnv)。
func LoadConfig() (*Config, error) {
	return LoadConfigFrom(os.LookupEnv)
}

// LoadConfigFrom 按「默认值 < 环境变量」优先级构造配置；lookup 注入是为了可测试。
func LoadConfigFrom(lookup func(string) (string, bool)) (*Config, error) {
	cfg := &Config{Addr: ":8080", DSN: ":memory:", LogLevel: "info"}
	if v, ok := lookup("APP_ADDR"); ok && v != "" {
		cfg.Addr = v
	}
	if v, ok := lookup("APP_DSN"); ok && v != "" {
		cfg.DSN = v
	}
	if v, ok := lookup("APP_LOG_LEVEL"); ok && v != "" {
		if !validLevel(v) {
			return nil, badRequest("APP_LOG_LEVEL 非法: %q", v)
		}
		cfg.LogLevel = v
	}
	return cfg, nil
}

// validLevel 只接受四种日志级别，越早拒绝非法配置越省事。
func validLevel(s string) bool {
	switch s {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
