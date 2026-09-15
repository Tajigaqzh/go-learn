package go32_engineering

import (
	"errors"
	"testing"
	"time"
)

// TestMemoryStoreGetList 验证内存实现的查询与稳定排序。
func TestMemoryStoreGetList(t *testing.T) {
	s := newMemoryStore()

	u, err := s.Get(1)
	if err != nil || u.Name != "张三" {
		t.Fatalf("Get(1) = %+v, %v，期望 张三", u, err)
	}

	if _, err := s.Get(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(99) 期望 ErrNotFound，得到 %v", err)
	}

	users := s.List()
	if len(users) != 2 {
		t.Fatalf("期望 2 个用户，得到 %d", len(users))
	}
	if users[0].ID != 1 || users[1].ID != 2 {
		t.Fatalf("List 应按 ID 升序，得到 %v", users)
	}
}

// TestUserServiceGetByID 验证 service 层对哨兵错误的包装与 errors.Is 穿透。
func TestUserServiceGetByID(t *testing.T) {
	svc := NewUserService(newMemoryStore())

	if _, err := svc.GetByID(1); err != nil {
		t.Fatalf("GetByID(1) 不应出错，得到 %v", err)
	}

	_, err := svc.GetByID(99)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID(99) 期望能穿透到 ErrNotFound，得到 %v", err)
	}

	if _, err := svc.GetByID(0); err == nil {
		t.Fatal("GetByID(0) 应返回参数非法错误")
	}
}

// TestUserServiceWithFakeStore 验证构造器注入让 service 可测：换用 fakeStore 并观察调用次数。
func TestUserServiceWithFakeStore(t *testing.T) {
	fake := &fakeStore{users: map[int]User{7: {ID: 7, Name: "替身"}}}
	svc := NewUserService(fake)

	u, err := svc.GetByID(7)
	if err != nil || u.Name != "替身" {
		t.Fatalf("GetByID(7) = %+v, %v", u, err)
	}
	if fake.getCalls != 1 {
		t.Fatalf("期望 Get 被调用 1 次，实际 %d", fake.getCalls)
	}

	if _, err := svc.GetByID(8); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID(8) 期望 ErrNotFound，得到 %v", err)
	}
	if fake.getCalls != 2 {
		t.Fatalf("期望 Get 被调用 2 次，实际 %d", fake.getCalls)
	}
}

// TestNewServerDefaults 验证 Functional Options 的默认值。
func TestNewServerDefaults(t *testing.T) {
	s := NewServer(":8080")
	if s.addr != ":8080" || s.timeout != 3*time.Second || s.maxConns != 100 {
		t.Fatalf("默认配置不符合预期：addr=%s timeout=%s maxConns=%d",
			s.addr, s.timeout, s.maxConns)
	}
}

// TestNewServerOptions 验证 Functional Options 按序覆盖默认值。
func TestNewServerOptions(t *testing.T) {
	s := NewServer(":9090", WithTimeout(500*time.Millisecond), WithMaxConns(10))
	if s.timeout != 500*time.Millisecond {
		t.Fatalf("期望 timeout 500ms，得到 %s", s.timeout)
	}
	if s.maxConns != 10 {
		t.Fatalf("期望 maxConns 10，得到 %d", s.maxConns)
	}
}

// TestLoadConfigDefaults 验证无环境变量时回退到默认值。
func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfigFrom(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("不应出错，得到 %v", err)
	}
	if cfg.Addr != "0.0.0.0" || cfg.Port != 8080 || cfg.LogLevel != "info" {
		t.Fatalf("默认配置不符合预期：%+v", cfg)
	}
}

// TestLoadConfigOverride 验证环境变量覆盖默认值。
func TestLoadConfigOverride(t *testing.T) {
	env := map[string]string{"APP_ADDR": "127.0.0.1", "APP_PORT": "9090", "APP_LOG_LEVEL": "debug"}
	cfg, err := LoadConfigFrom(func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	})
	if err != nil {
		t.Fatalf("不应出错，得到 %v", err)
	}
	want := Config{Addr: "127.0.0.1", Port: 9090, LogLevel: "debug"}
	if *cfg != want {
		t.Fatalf("期望 %+v，得到 %+v", want, *cfg)
	}
}

// TestLoadConfigInvalid 验证非法环境变量尽早报错。
func TestLoadConfigInvalid(t *testing.T) {
	_, err := LoadConfigFrom(func(k string) (string, bool) {
		return "not-a-number", k == "APP_PORT"
	})
	if err == nil {
		t.Fatal("非法 APP_PORT 应报错")
	}
	// 校验范围也能兜住越界值。
	if _, err := LoadConfigFrom(func(k string) (string, bool) {
		return "99999", k == "APP_PORT"
	}); err == nil {
		t.Fatal("越界 APP_PORT 应报错")
	}
}
