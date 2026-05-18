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
var _ domain.UserOnlineFetcher = (*userOnlineFetcher)(nil)

type userOnlineFetcher struct {
	client *Client
	logger *zap.Logger
}

// NewUserOnlineFetcher 构造 UserOnlineFetcher 实现。
func NewUserOnlineFetcher(client *Client, logger *zap.Logger) domain.UserOnlineFetcher {
	return &userOnlineFetcher{client: client, logger: logger}
}

// GetOnlineTotal 获取在线用户总数统计（GET /UserOnline）。
// API 返回 data 为数组，取第一个元素。
func (f *userOnlineFetcher) GetOnlineTotal(ctx context.Context) (*domain.OnlineTotal, error) {
	data, err := f.client.do(ctx, http.MethodGet, "/UserOnline", nil)
	if err != nil {
		return nil, fmt.Errorf("aisacg: GetOnlineTotal: %w", err)
	}
	var items []domain.OnlineTotal
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("aisacg: GetOnlineTotal unmarshal: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("aisacg: GetOnlineTotal: empty response")
	}
	return &items[0], nil
}

// ListOnlineUsers 根据组织 path 获取在线用户明细（POST /UserOnline）。
// API 在叶节点（仅 1 个在线用户）时返回单个对象，多用户时返回数组，两种情况均兼容。
func (f *userOnlineFetcher) ListOnlineUsers(ctx context.Context, path string) ([]domain.UserOnline, error) {
	body := map[string]string{"path": path}
	data, err := f.client.do(ctx, http.MethodPost, "/UserOnline", body)
	if err != nil {
		return nil, fmt.Errorf("aisacg: ListOnlineUsers: %w", err)
	}
	// 叶节点单用户时 API 返回 {} 而非 [{}]，需兼容两种格式
	if len(data) > 0 && data[0] == '{' {
		var single domain.UserOnline
		if err := json.Unmarshal(data, &single); err != nil {
			return nil, fmt.Errorf("aisacg: ListOnlineUsers unmarshal single: %w", err)
		}
		return []domain.UserOnline{single}, nil
	}
	var users []domain.UserOnline
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("aisacg: ListOnlineUsers unmarshal: %w", err)
	}
	return users, nil
}

// GetOnlineTree 获取在线用户顶层组织树（GET /UserOnline/tree）。
func (f *userOnlineFetcher) GetOnlineTree(ctx context.Context) ([]domain.OnlineTreeNode, error) {
	data, err := f.client.do(ctx, http.MethodGet, "/UserOnline/tree", nil)
	if err != nil {
		return nil, fmt.Errorf("aisacg: GetOnlineTree: %w", err)
	}
	var nodes []domain.OnlineTreeNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("aisacg: GetOnlineTree unmarshal: %w", err)
	}
	return nodes, nil
}

// GetOnlineTreeByPath 按 path 查询在线用户组织树（POST /UserOnline/tree）。
func (f *userOnlineFetcher) GetOnlineTreeByPath(ctx context.Context, path string) ([]domain.OnlineTreeNode, error) {
	body := map[string]string{"path": path}
	data, err := f.client.do(ctx, http.MethodPost, "/UserOnline/tree", body)
	if err != nil {
		return nil, fmt.Errorf("aisacg: GetOnlineTreeByPath: %w", err)
	}
	var nodes []domain.OnlineTreeNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("aisacg: GetOnlineTreeByPath unmarshal: %w", err)
	}
	return nodes, nil
}
