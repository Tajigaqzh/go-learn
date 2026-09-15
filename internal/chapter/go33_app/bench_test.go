package go33_app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BenchmarkCreateUser 度量单次创建请求的完整开销（handler → service → repo）。
// 数据库用内存 SQLite、logger 落空，排除外部依赖的抖动。
func BenchmarkCreateUser(b *testing.B) {
	db := openDemoDB()
	if err := Migrate(b.Context(), db); err != nil {
		b.Fatal(err)
	}
	app := NewApp(
		NewUserService(NewSQLiteUserRepo(db)),
		newSlog(&strings.Builder{}),
		WithRequestIDGen(counter()),
	)
	h := app.Handler()
	body := `{"name":"张三","email":"zhang@example.com"}`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated && rec.Code != http.StatusConflict {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}
