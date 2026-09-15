package go33_app

import (
	"context"
	"net/mail"
)

// User 是用户资源；JSON 只暴露 id / name / email，不含内部字段。
type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserRepository 定义数据访问契约，handler / service 只依赖它。
type UserRepository interface {
	Create(ctx context.Context, name, email string) (User, error)
	Get(ctx context.Context, id int64) (User, error)
	List(ctx context.Context) ([]User, error)
}

// UserService 是业务层：参数校验 + 调用仓储，底层错误统一成 *AppError。
type UserService struct {
	repo UserRepository
}

// NewUserService 注入 UserRepository；替换实现即可替换整个数据来源。
func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Create 先做入参校验再落库。
func (s *UserService) Create(ctx context.Context, name, email string) (User, error) {
	if name == "" {
		return User{}, badRequest("name 不能为空")
	}
	// 用标准库 net/mail 做一个轻量校验；严格场景再上更完整的规则。
	if _, err := mail.ParseAddress(email); err != nil {
		return User{}, badRequest("email 格式错误: %q", email)
	}
	return s.repo.Create(ctx, name, email)
}

// Get 按 ID 查询；仓储层已把「找不到」归一成 *AppError，这里直接透传。
func (s *UserService) Get(ctx context.Context, id int64) (User, error) {
	if id <= 0 {
		return User{}, badRequest("id 必须为正整数: %d", id)
	}
	return s.repo.Get(ctx, id)
}

// List 返回全部用户。
func (s *UserService) List(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}
