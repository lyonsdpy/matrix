package service

import (
	"context"
	"fmt"

	"matrix/api/domain"
)

// syncedEndpointRepo 终端管理列表/详情所需的图层能力（接口定义在使用方）。
type syncedEndpointRepo interface {
	Search(ctx context.Context, q, userQ, typeFilter, osFilter, cursor string, limit int) ([]*domain.SyncedEndpoint, bool, string, error)
	GetDetail(ctx context.Context, id string) (*domain.SyncedEndpoint, error)
}

// EndpointService 终端管理（用户终端 ↔ 飞书同步设备）。
// 与 SyncService 解耦：Sync 负责写入，Endpoint 负责查询。
type EndpointService struct {
	graphRepo syncedEndpointRepo
}

func NewEndpointService(graph syncedEndpointRepo) *EndpointService {
	return &EndpointService{graphRepo: graph}
}

// EndpointList 终端列表查询结果，含游标分页信息。字段命名与 SyncedUserList 对齐。
type EndpointList struct {
	Endpoints []*domain.SyncedEndpoint `json:"endpoints"`
	HasNext   bool                     `json:"has_next"`
	EndCursor string                   `json:"end_cursor"`
}

const (
	defaultEndpointPageSize = 20
	maxEndpointPageSize     = 100
)

// List 按搜索词 / 关联用户 / 类型 / 系统 / 游标分页查询终端。
// limit 越界时归一到合理范围；typeFilter / osFilter 空字符串表示不筛。
func (s *EndpointService) List(ctx context.Context, q, userQ, typeFilter, osFilter, cursor string, limit int) (*EndpointList, error) {
	if limit <= 0 {
		limit = defaultEndpointPageSize
	}
	if limit > maxEndpointPageSize {
		limit = maxEndpointPageSize
	}
	endpoints, hasNext, endCursor, err := s.graphRepo.Search(ctx, q, userQ, typeFilter, osFilter, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("endpoint.List: %w", err)
	}
	if endpoints == nil {
		endpoints = []*domain.SyncedEndpoint{}
	}
	return &EndpointList{Endpoints: endpoints, HasNext: hasNext, EndCursor: endCursor}, nil
}

// GetDetail 按节点 id 查终端详情；返回 nil 表示不存在。
func (s *EndpointService) GetDetail(ctx context.Context, id string) (*domain.SyncedEndpoint, error) {
	if id == "" {
		return nil, nil
	}
	ep, err := s.graphRepo.GetDetail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("endpoint.GetDetail: %w", err)
	}
	return ep, nil
}
