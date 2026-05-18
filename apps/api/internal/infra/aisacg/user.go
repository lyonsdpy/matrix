package aisacg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	domain "matrix/api/domain/aisacg"

	"go.uber.org/zap"
)

// 编译期 interface 合规检查。
var _ domain.UserManager = (*userManager)(nil)

type userManager struct {
	client *Client
	logger *zap.Logger
}

// NewUserManager 构造 UserManager 实现。
func NewUserManager(client *Client, logger *zap.Logger) domain.UserManager {
	return &userManager{client: client, logger: logger}
}

// ListUsers 获取所有用户列表（GET /AuthUser）。
func (m *userManager) ListUsers(ctx context.Context) ([]domain.User, error) {
	data, err := m.client.do(ctx, http.MethodGet, "/AuthUser", nil)
	if err != nil {
		return nil, fmt.Errorf("aisacg: ListUsers: %w", err)
	}
	var users []domain.User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("aisacg: ListUsers unmarshal: %w", err)
	}
	return users, nil
}

// GetUser 查询单个用户（GET /AuthUser/name/{name}）。
func (m *userManager) GetUser(ctx context.Context, name string) (*domain.User, error) {
	data, err := m.client.do(ctx, http.MethodGet, "/AuthUser/name/"+name, nil)
	if err != nil {
		return nil, fmt.Errorf("aisacg: GetUser: %w", err)
	}
	// API 返回的是数组，取第一个元素
	var users []domain.User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("aisacg: GetUser unmarshal: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("aisacg: GetUser: user %q not found", name)
	}
	return &users[0], nil
}

// CreateUser 新建用户（POST /AuthUser）。
func (m *userManager) CreateUser(ctx context.Context, req *domain.UserUpdate) (*domain.UserUpdate, error) {
	data, err := m.client.do(ctx, http.MethodPost, "/AuthUser", req)
	if err != nil {
		return nil, fmt.Errorf("aisacg: CreateUser: %w", err)
	}
	result, err := unmarshalUserUpdate(data)
	if err != nil {
		return nil, fmt.Errorf("aisacg: CreateUser unmarshal: %w", err)
	}
	return result, nil
}

// UpdateUser 更新用户信息（PUT /AuthUser/name/{name}）。
func (m *userManager) UpdateUser(ctx context.Context, name string, req *domain.UserUpdate) (*domain.UserUpdate, error) {
	data, err := m.client.do(ctx, http.MethodPut, "/AuthUser/name/"+name, req)
	if err != nil {
		return nil, fmt.Errorf("aisacg: UpdateUser: %w", err)
	}
	result, err := unmarshalUserUpdate(data)
	if err != nil {
		return nil, fmt.Errorf("aisacg: UpdateUser unmarshal: %w", err)
	}
	return result, nil
}

// unmarshalUserUpdate 兼容 API 返回单对象（{...}）或数组（[{...}]）两种格式。
// 真机观测：CreateUser/UpdateUser 返回单对象而非数组。
func unmarshalUserUpdate(data json.RawMessage) (*domain.UserUpdate, error) {
	// 尝试数组格式
	var items []domain.UserUpdate
	if err := json.Unmarshal(data, &items); err == nil {
		if len(items) == 0 {
			return nil, fmt.Errorf("empty array response")
		}
		return &items[0], nil
	}
	// 回退到单对象格式
	var item domain.UserUpdate
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteUser 删除用户（DELETE /AuthUser/name/{name}）。
func (m *userManager) DeleteUser(ctx context.Context, name string) error {
	_, err := m.client.do(ctx, http.MethodDelete, "/AuthUser/name/"+name, nil)
	if err != nil {
		return fmt.Errorf("aisacg: DeleteUser: %w", err)
	}
	return nil
}
