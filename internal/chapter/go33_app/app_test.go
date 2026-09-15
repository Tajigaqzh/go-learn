package go33_app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeRepo 是内存版 UserRepository，把 service 的校验与数据来源解耦。
type fakeRepo struct {
	users  []User
	nextID int64
}

func (f *fakeRepo) Create(_ context.Context, name, email string) (User, error) {
	f.nextID++
	u := User{ID: f.nextID, Name: name, Email: email}
	f.users = append(f.users, u)
	return u, nil
}

func (f *fakeRepo) Get(_ context.Context, id int64) (User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, notFound("用户 %d 不存在", id)
}

func (f *fakeRepo) List(_ context.Context) ([]User, error) {
	return f.users, nil
}

func TestUserServiceCreate(t *testing.T) {
	svc := NewUserService(&fakeRepo{})

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"张三", "zhang@example.com", false},
		{"", "a@b.com", true},
		{"张三", "bad-email", true},
	}
	for _, tt := range tests {
		_, err := svc.Create(context.Background(), tt.name, tt.email)
		if (err != nil) != tt.wantErr {
			t.Fatalf("Create(%q, %q) err=%v, wantErr=%v", tt.name, tt.email, err, tt.wantErr)
		}
	}
}

func TestUserServiceGet(t *testing.T) {
	svc := NewUserService(&fakeRepo{})
	if _, err := svc.Get(context.Background(), 0); err == nil {
		t.Fatal("id=0 应当被拒绝")
	}
	if _, err := svc.Get(context.Background(), 42); err == nil {
		t.Fatal("不存在的 id 应当报 not_found")
	}
}

func TestConfigLoadFrom(t *testing.T) {
	none := func(string) (string, bool) { return "", false }
	cfg, err := LoadConfigFrom(none)
	if err != nil {
		t.Fatalf("默认配置不应报错: %v", err)
	}
	if cfg.Addr != ":8080" || cfg.DSN != ":memory:" || cfg.LogLevel != "info" {
		t.Fatalf("默认值不符: %+v", cfg)
	}

	env := map[string]string{"APP_ADDR": "127.0.0.1:9999", "APP_LOG_LEVEL": "debug"}
	cfg, err = LoadConfigFrom(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil {
		t.Fatalf("覆盖配置不应报错: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9999" || cfg.LogLevel != "debug" {
		t.Fatalf("环境变量未生效: %+v", cfg)
	}

	if _, err := LoadConfigFrom(func(k string) (string, bool) {
		return "verbose", k == "APP_LOG_LEVEL"
	}); err == nil {
		t.Fatal("非法日志级别应当报错")
	}
}

func TestResponseOf(t *testing.T) {
	if status, code, _ := responseOf(notFound("x")); status != http.StatusNotFound || code != CodeNotFound {
		t.Fatalf("AppError 映射错误: %d %s", status, code)
	}
	// 非 *AppError 一律走 500/internal，避免泄露内部细节。
	if status, code, _ := responseOf(errors.New("boom")); status != http.StatusInternalServerError || code != CodeInternal {
		t.Fatalf("普通 error 应归为 500/internal: %d %s", status, code)
	}
}

// newTestApp 建一个在内存 SQLite 上装配好、带自增请求 ID 的应用。
func newTestApp() *App {
	db := openDemoDB()
	if err := Migrate(context.Background(), db); err != nil {
		panic(err)
	}
	return NewApp(NewUserService(NewSQLiteUserRepo(db)), newSlog(&strings.Builder{}), WithRequestIDGen(counter()))
}

func doTestRequest(t *testing.T, h http.Handler, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

func TestAppCreateGetListFlow(t *testing.T) {
	h := newTestApp().Handler()

	status, _ := doTestRequest(t, h, http.MethodPost, "/api/v1/users", `{"name":"张三","email":"zhang@example.com"}`)
	if status != http.StatusCreated {
		t.Fatalf("创建失败: %d", status)
	}

	// 重复邮箱触发唯一约束 -> 409 already_exists。
	status, body := doTestRequest(t, h, http.MethodPost, "/api/v1/users", `{"name":"李四","email":"zhang@example.com"}`)
	if status != http.StatusConflict || !strings.Contains(body, "already_exists") {
		t.Fatalf("重复邮箱应 409: %d %s", status, body)
	}

	status, body = doTestRequest(t, h, http.MethodGet, "/api/v1/users/1", "")
	if status != http.StatusOK || !strings.Contains(body, `"zhang@example.com"`) {
		t.Fatalf("查询失败: %d %s", status, body)
	}

	status, body = doTestRequest(t, h, http.MethodGet, "/api/v1/users", "")
	if status != http.StatusOK {
		t.Fatalf("列表失败: %d %s", status, body)
	}
	var list []User
	if err := json.Unmarshal([]byte(body), &list); err != nil || len(list) != 1 {
		t.Fatalf("列表内容不符: %s", body)
	}
}

func TestAppErrorPaths(t *testing.T) {
	h := newTestApp().Handler()

	status, _ := doTestRequest(t, h, http.MethodPost, "/api/v1/users", `{"name":"","email":"a@b.com"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("空 name 应 400: %d", status)
	}

	status, _ = doTestRequest(t, h, http.MethodGet, "/api/v1/users/999", "")
	if status != http.StatusNotFound {
		t.Fatalf("不存在用户应 404: %d", status)
	}

	status, _ = doTestRequest(t, h, http.MethodGet, "/api/v1/users/abc", "")
	if status != http.StatusBadRequest {
		t.Fatalf("非法 id 应 400: %d", status)
	}
}

func TestMiddlewareRequestID(t *testing.T) {
	gen := counter()
	app := NewApp(NewUserService(&fakeRepo{}), newSlog(&strings.Builder{}), WithRequestIDGen(gen))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "caller-1")
	app.Handler().ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "caller-1" {
		t.Fatalf("应沿用调用方 ID: %q", got)
	}
	// 没带 ID 时用生成器。
	rec = httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if got := rec.Header().Get("X-Request-ID"); got != "req-1" {
		t.Fatalf("应使用生成器 ID: %q", got)
	}
}

func TestMiddlewarePanicRecovery(t *testing.T) {
	app := newTestApp()
	app.Mux().HandleFunc("GET /boom", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	status, body := doTestRequest(t, app.Handler(), http.MethodGet, "/boom", "")
	if status != http.StatusInternalServerError || !strings.Contains(body, "internal") {
		t.Fatalf("panic 应被转成 500/internal: %d %s", status, body)
	}
}
