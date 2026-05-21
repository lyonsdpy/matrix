package service

import (
	"context"
	"fmt"

	"matrix/api/domain"
	"matrix/api/internal/repository/pg_repo"
)

// EmployeeView 跨源聚合视图：PG 同步数据 + 图中 User 节点。
// 仅用于读接口的响应，不存储。
type EmployeeView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	FeishuID string `json:"feishu_id"`
	// UserID 是图中对应 User 节点的 ID。
	// 若图同步尚未完成此字段为空，不报错。
	UserID string `json:"user_id,omitempty"`
}

// employeeSyncRepo 定义从 PG 同步层需要的能力（接口定义在使用方，Go 惯用法）。
type employeeSyncRepo interface {
	FindByID(ctx context.Context, id string) (*pg_repo.Employee, error)
}

// userGraphRepo 定义从图层需要的能力。
type userGraphRepo interface {
	FindByFeishuID(ctx context.Context, feishuID string) (*domain.User, error)
}

// EmployeeService 演示跨源聚合：PG 同步数据 + 图中 User 节点。
// Service 层是两个数据源的唯一汇合点。
type EmployeeService struct {
	syncRepo  employeeSyncRepo
	graphRepo userGraphRepo
}

func NewEmployeeService(sync employeeSyncRepo, graph userGraphRepo) *EmployeeService {
	return &EmployeeService{syncRepo: sync, graphRepo: graph}
}

// Get 按员工 ID 查询，返回跨源聚合视图。
//
// 数据流：
//  1. 从 PG 取飞书同步的员工记录（名字、邮箱、飞书 ID）
//  2. 用飞书 ID 在图中查找对应的 User 节点
//  3. 手工 join，返回聚合视图
func (s *EmployeeService) Get(ctx context.Context, id string) (*EmployeeView, error) {
	emp, err := s.syncRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("employee.Get: pg lookup: %w", err)
	}
	if emp == nil {
		return nil, fmt.Errorf("employee %s: not found", id)
	}

	user, err := s.graphRepo.FindByFeishuID(ctx, emp.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("employee.Get: graph lookup: %w", err)
	}

	view := &EmployeeView{
		ID:       emp.ID,
		Name:     emp.Name,
		Email:    emp.Email,
		FeishuID: emp.ExternalID,
	}
	if user != nil {
		view.UserID = user.ID
	}
	return view, nil
}
