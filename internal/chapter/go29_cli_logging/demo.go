// Package go29_cli_logging 演示可测试的命令行、结构化日志与分层配置。
package go29_cli_logging

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Demo 运行第 29 章的全部演示。
func Demo() {
	fmt.Println("\n========== go29_cli_logging: 命令行工具、日志与配置 ==========")
	demoArgsAndFlag()
	demoSubcommandsAndExitCode()
	demoStructuredLogging()
	demoContextAndLevel()
	demoRotation()
	demoConfigPrecedence()
	demoSecretRedaction()
	fmt.Println("========== 命令行工具、日志与配置演示结束 ==========")
	fmt.Println()
}

func demoArgsAndFlag() {
	fmt.Println("--- 29.1 os.Args、flag 与可测试边界 ---")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"user", "-name", "小林", "-verbose", "7"}, &stdout, &stderr)
	fmt.Printf("stdout: %s", stdout.String())
	fmt.Printf("stderr: %s", stderr.String())
	fmt.Printf("exit code: %d\n\n", code)
}

func demoSubcommandsAndExitCode() {
	fmt.Println("--- 29.2 子命令、输出分流与退出码 ---")
	for _, args := range [][]string{{"version"}, {"unknown"}, {"user", "404"}} {
		var stdout, stderr bytes.Buffer
		code := Run(args, &stdout, &stderr)
		fmt.Printf("args=%q code=%d stdout=%q stderr=%q\n", args, code, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	fmt.Println()
}

func demoStructuredLogging() {
	fmt.Println("--- 29.3 slog 结构化日志与 Handler ---")
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{ReplaceAttr: removeTime}))
	logger.Info("订单已创建", "order_id", 42, "amount", 99.5)
	fmt.Print(output.String())
	fmt.Println()
}

func demoContextAndLevel() {
	fmt.Println("--- 29.4 日志级别与请求上下文 ---")
	var output bytes.Buffer
	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: level, ReplaceAttr: removeTime}))
	ctx := WithRequestID(context.Background(), "req-29")
	LoggerFromContext(ctx, logger).Debug("不会出现")
	LoggerFromContext(ctx, logger).Info("请求完成", "status", 200)
	fmt.Print(output.String())
	fmt.Println()
}

func demoRotation() {
	fmt.Println("--- 29.5 日志落盘与大小轮转 ---")
	dir, err := os.MkdirTemp("", "go29-log-")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n\n", err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "app.log")
	w, err := NewRotatingWriter(path, 12)
	if err != nil {
		fmt.Printf("打开轮转日志失败: %v\n\n", err)
		return
	}
	_, _ = w.Write([]byte("first-line\n"))
	_, _ = w.Write([]byte("second-line\n"))
	_ = w.Close()
	current, backup := LogFileNames(path)
	fmt.Printf("超过 12 字节后: 当前=%s 备份=%s\n\n", current, backup)
}

func demoConfigPrecedence() {
	fmt.Println("--- 29.6 配置优先级与校验 ---")
	cfg, err := LoadConfig(map[string]string{"host": "file.local", "port": "7000"}, map[string]string{"port": "8000", "log_level": "warn"}, map[string]string{"port": "9000"})
	if err != nil {
		fmt.Printf("加载配置失败: %v\n\n", err)
		return
	}
	fmt.Printf("host=%s port=%d level=%s\n", cfg.Host, cfg.Port, cfg.LogLevel)
	_, err = LoadConfig(nil, map[string]string{"port": "70000"}, nil)
	fmt.Printf("非法配置: %v\n\n", err)
}

func demoSecretRedaction() {
	fmt.Println("--- 29.7 敏感信息脱敏 ---")
	fmt.Printf("api_key=%s\n", RedactSecret("sk-demo-12345678"))
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: removeTime}))
	logger.Info("连接外部服务", "api_key", RedactSecret("sk-demo-12345678"), "timeout", 2*time.Second)
	fmt.Println()
}

func removeTime(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return attr
}
