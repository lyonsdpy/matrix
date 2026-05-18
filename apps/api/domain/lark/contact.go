package lark

import "context"

// ContactFetcher 通讯录全量拉取能力。
type ContactFetcher interface {
	// FetchAllDepartments 从根部门递归获取所有部门。
	FetchAllDepartments(ctx context.Context) ([]Department, error)

	// FetchDepartmentUsers 获取指定部门的直属用户列表（分页全量）。
	FetchDepartmentUsers(ctx context.Context, departmentID string) ([]User, error)

	// FetchAllUsers 递归获取所有部门下的所有用户（去重）。
	FetchAllUsers(ctx context.Context) ([]User, error)

	// GetUser 获取单个用户详情。
	GetUser(ctx context.Context, userID string) (*User, error)

	// GetDepartment 获取单个部门详情。
	GetDepartment(ctx context.Context, departmentID string) (*Department, error)
}
