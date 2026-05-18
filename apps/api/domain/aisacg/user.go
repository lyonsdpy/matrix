package aisacg

import "context"

// UserManager 亚信 ACG 用户管理能力。
type UserManager interface {
	// ListUsers 获取所有用户列表。
	ListUsers(ctx context.Context) ([]User, error)

	// GetUser 查询单个用户。
	GetUser(ctx context.Context, name string) (*User, error)

	// CreateUser 新建用户。
	CreateUser(ctx context.Context, req *UserUpdate) (*UserUpdate, error)

	// UpdateUser 更新用户信息。
	UpdateUser(ctx context.Context, name string, req *UserUpdate) (*UserUpdate, error)

	// DeleteUser 删除用户。
	DeleteUser(ctx context.Context, name string) error
}
