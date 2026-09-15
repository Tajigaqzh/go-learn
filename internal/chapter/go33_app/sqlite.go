package go33_app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// Migrate 建表。IF NOT EXISTS 保证幂等，可反复执行——这是「迁移」的最小可靠形态。
// 真实项目里迁移通常拆成带版本的多个文件，由工具按序应用。
func Migrate(ctx context.Context, db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	email      TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// SQLiteUserRepo 是 UserRepository 的 SQLite 实现。
type SQLiteUserRepo struct {
	db *sql.DB
}

// NewSQLiteUserRepo 包装一个 *sql.DB。
// 内存模式（:memory:）每个连接各有一个独立库，调用方必须限定单一连接才能共享数据。
func NewSQLiteUserRepo(db *sql.DB) *SQLiteUserRepo {
	return &SQLiteUserRepo{db: db}
}

// Create 插入用户；email 唯一冲突时产出 already_exists。
func (r *SQLiteUserRepo) Create(ctx context.Context, name, email string) (User, error) {
	res, err := r.db.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", name, email)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, conflict("邮箱 %q 已被占用", email)
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("last insert id: %w", err)
	}
	return User{ID: id, Name: name, Email: email}, nil
}

// Get 查询单条；无结果时归一成 not_found。
func (r *SQLiteUserRepo) Get(ctx context.Context, id int64) (User, error) {
	var u User
	row := r.db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = ?", id)
	if err := row.Scan(&u.ID, &u.Name, &u.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, notFound("用户 %d 不存在", id)
		}
		return User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return u, nil
}

// List 查询全部用户，按 id 升序保证输出稳定。
func (r *SQLiteUserRepo) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, email FROM users ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// isUniqueViolation 通过驱动暴露的错误码判断约束冲突，而不是比对错误字符串。
func isUniqueViolation(err error) bool {
	var se sqlite3.Error
	return errors.As(err, &se) && se.Code == sqlite3.ErrConstraint
}
