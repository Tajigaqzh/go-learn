package go28_database

import (
	"database/sql"
	"errors"
	"testing"
)

// openTestDB 建一个内存 SQLite 测试库。:memory: 每个连接各有独立库，
// 必须 SetMaxOpenConns(1) 把所有操作限定在同一个连接上，才能共享建表和写入。
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open 失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestCRUDAndQuery(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, email TEXT UNIQUE)`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "a@example.com"); err != nil {
		t.Fatalf("插入失败: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "b@example.com"); err != nil {
		t.Fatalf("插入失败: %v", err)
	}

	var name string
	if err := db.QueryRow("SELECT name FROM users WHERE id = ?", 1).Scan(&name); err != nil {
		t.Fatalf("QueryRow 失败: %v", err)
	}
	if name != "Alice" {
		t.Fatalf("id=1 应为 Alice，得到 %q", name)
	}

	rows, err := db.Query("SELECT name FROM users ORDER BY id")
	if err != nil {
		t.Fatalf("Query 失败: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("Scan 失败: %v", err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	if len(names) != 2 || names[0] != "Alice" || names[1] != "Bob" {
		t.Fatalf("多行结果不符: %v", names)
	}
}

// TestPlaceholderPreventsInjection 是 28.4 的核心结论：恶意输入经占位符绑定后
// 只当普通字符串比较，永远无法拼出第二条 WHERE 子句。
func TestPlaceholderPreventsInjection(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if _, err := db.Exec("INSERT INTO products (name) VALUES (?)", "Laptop"); err != nil {
		t.Fatalf("插入失败: %v", err)
	}

	malicious := "Laptop' OR '1'='1"
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM products WHERE name = ?", malicious).Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("占位符应防注入：期望 0 行，得到 %d 行", count)
	}
}

func TestSQLNullString(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT, bio TEXT)`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if _, err := db.Exec("INSERT INTO authors (name, bio) VALUES (?, ?)", "Bob", nil); err != nil {
		t.Fatalf("插入失败: %v", err)
	}

	// NULL 直接扫进 string 会失败（无法转换）
	var bio string
	if err := db.QueryRow("SELECT bio FROM authors WHERE name = ?", "Bob").Scan(&bio); err == nil {
		t.Fatal("NULL 扫描到 string 应当报错")
	}

	var nb sql.NullString
	if err := db.QueryRow("SELECT bio FROM authors WHERE name = ?", "Bob").Scan(&nb); err != nil {
		t.Fatalf("sql.NullString 扫描失败: %v", err)
	}
	if nb.Valid {
		t.Fatalf("bio 应为 NULL，得到 %q", nb.String)
	}
}

func TestTransactionRollback(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`CREATE TABLE accounts (id INTEGER PRIMARY KEY, name TEXT, balance REAL)`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if _, err := db.Exec("INSERT INTO accounts (name, balance) VALUES (?, ?)", "Alice", 100.0); err != nil {
		t.Fatalf("插入失败: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin 失败: %v", err)
	}
	if _, err := tx.Exec("UPDATE accounts SET balance = 9999 WHERE name = ?", "Alice"); err != nil {
		t.Fatalf("UPDATE 失败: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback 失败: %v", err)
	}

	var balance float64
	if err := db.QueryRow("SELECT balance FROM accounts WHERE name = ?", "Alice").Scan(&balance); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if balance != 100.0 {
		t.Fatalf("回滚后余额应为 100，得到 %v", balance)
	}
}

func TestErrNoRows(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	var v string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "missing").Scan(&v)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("期望 sql.ErrNoRows，得到 %v", err)
	}
}
