package go29_cli_logging

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// Config 保存本章示例程序的运行配置。
type Config struct {
	Host     string
	Port     int
	LogLevel slog.Level
	APIKey   string
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{Host: "127.0.0.1", Port: 8080, LogLevel: slog.LevelInfo}
}

// LoadConfig 按“默认值 < 配置文件 < 环境变量 < 命令行”的顺序合并配置。
func LoadConfig(fileValues, envValues, flagValues map[string]string) (Config, error) {
	cfg := DefaultConfig()
	for _, source := range []map[string]string{fileValues, envValues, flagValues} {
		if err := applyConfig(&cfg, source); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

func applyConfig(cfg *Config, values map[string]string) error {
	for key, value := range values {
		switch strings.ToLower(key) {
		case "host":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("host 不能为空")
			}
			cfg.Host = value
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("port %q 必须是 1..65535 的整数", value)
			}
			cfg.Port = port
		case "log_level":
			var level slog.Level
			if err := level.UnmarshalText([]byte(value)); err != nil {
				return fmt.Errorf("解析 log_level %q: %w", value, err)
			}
			cfg.LogLevel = level
		case "api_key":
			cfg.APIKey = value
		}
	}
	return nil
}

// RedactSecret 只保留密钥末四位；短密钥完全隐藏。
func RedactSecret(value string) string {
	if value == "" {
		return "<empty>"
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}
