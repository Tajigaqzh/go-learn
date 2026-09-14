package go29_cli_logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name                   string
		args                   []string
		wantCode               int
		wantStdout, wantStderr string
	}{
		{"成功", []string{"user", "-name", "小林", "7"}, 0, "id=7 name=小林", ""},
		{"参数错误", []string{"user", "zero"}, 2, "", "必须是正整数"},
		{"业务错误", []string{"user", "404"}, 1, "", "user not found"},
		{"未知命令", []string{"missing"}, 2, "", "未知子命令"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if got := Run(tt.args, &stdout, &stderr); got != tt.wantCode {
				t.Errorf("退出码：期望 %d，实际 %d", tt.wantCode, got)
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout=%q，期望包含 %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr=%q，期望包含 %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestLoadConfigPrecedenceAndValidation(t *testing.T) {
	cfg, err := LoadConfig(map[string]string{"host": "file", "port": "7000"}, map[string]string{"host": "env", "port": "8000"}, map[string]string{"port": "9000"})
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Host != "env" || cfg.Port != 9000 {
		t.Errorf("优先级错误: %+v", cfg)
	}
	if _, err := LoadConfig(nil, map[string]string{"port": "0"}, nil); err == nil {
		t.Error("非法端口应当返回错误")
	}
}

func TestLoggerFromContextAndLevel(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := WithRequestID(context.Background(), "req-test")
	LoggerFromContext(ctx, logger).Debug("hidden")
	LoggerFromContext(ctx, logger).Info("done")
	got := output.String()
	if strings.Contains(got, "hidden") || !strings.Contains(got, "request_id=req-test") {
		t.Errorf("日志过滤或上下文字段错误: %q", got)
	}
}

func TestRotatingWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := NewRotatingWriter(path, 6)
	if err != nil {
		t.Fatalf("创建写入器失败: %v", err)
	}
	if _, err := w.Write([]byte("first\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("second\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != "second\n" {
		t.Errorf("当前日志不对: %q, %v", current, err)
	}
	backup, err := os.ReadFile(path + ".1")
	if err != nil || string(backup) != "first\n" {
		t.Errorf("备份日志不对: %q, %v", backup, err)
	}
}

func TestRedactSecret(t *testing.T) {
	for input, want := range map[string]string{"": "<empty>", "abc": "****", "sk-12345678": "****5678"} {
		if got := RedactSecret(input); got != want {
			t.Errorf("RedactSecret(%q)：期望 %q，实际 %q", input, want, got)
		}
	}
}
