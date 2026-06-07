package service

import (
	"context"
	"fmt"

	"matrix/api/domain"
)

// syncedEndpointRepo 终端管理列表/详情所需的图层能力（接口定义在使用方）。
type syncedEndpointRepo interface {
	Search(ctx context.Context, q, userQ, typeFilter, osFilter string, offset, limit int) ([]*domain.SyncedEndpoint, int64, error)
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

// EndpointList 终端列表查询结果，含分页元信息。
// 改用 page/page_size + total 的主流分页协议，便于前端做"跳到第 N 页"。
type EndpointList struct {
	Endpoints []*domain.SyncedEndpoint `json:"endpoints"`
	Total     int64                    `json:"total"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"page_size"`
}

const (
	defaultEndpointPageSize = 20
	maxEndpointPageSize     = 200
)

// List 按搜索词 / 关联用户 / 类型 / 系统 / 页码查询终端。
// page 从 1 起；pageSize 越界时归一到合理范围；typeFilter / osFilter 空字符串表示不筛。
func (s *EndpointService) List(ctx context.Context, q, userQ, typeFilter, osFilter string, page, pageSize int) (*EndpointList, error) {
	if pageSize <= 0 {
		pageSize = defaultEndpointPageSize
	}
	if pageSize > maxEndpointPageSize {
		pageSize = maxEndpointPageSize
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	endpoints, total, err := s.graphRepo.Search(ctx, q, userQ, typeFilter, osFilter, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("endpoint.List: %w", err)
	}
	if endpoints == nil {
		endpoints = []*domain.SyncedEndpoint{}
	}
	return &EndpointList{Endpoints: endpoints, Total: total, Page: page, PageSize: pageSize}, nil
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
