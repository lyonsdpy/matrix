package service

import (
	"context"
	"fmt"

	"matrix/api/domain"
)

// departmentGraphRepo Department 节点的图查询能力（接口定义在使用方）。
type departmentGraphRepo interface {
	ListChildren(ctx context.Context, parentID string) ([]*domain.DepartmentNode, error)
	GetDetail(ctx context.Context, deptID string, memberLimit int) (*domain.DepartmentDetail, error)
}

// DepartmentService 通讯录的部门读路径：树懒加载 + 详情。
type DepartmentService struct {
	repo departmentGraphRepo
}

func NewDepartmentService(repo departmentGraphRepo) *DepartmentService {
	return &DepartmentService{repo: repo}
}

const defaultDirectMemberLimit = 50

// ListChildren 返回某部门的直接子部门，供前端树懒加载。
// parentID 为 "" 或 "0" 时返回顶级部门（无 PARENT_OF 入边的节点）。
func (s *DepartmentService) ListChildren(ctx context.Context, parentID string) ([]*domain.DepartmentNode, error) {
	nodes, err := s.repo.ListChildren(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("department.ListChildren: %w", err)
	}
	if nodes == nil {
		nodes = []*domain.DepartmentNode{}
	}
	return nodes, nil
}

// GetDetail 部门详情：基本信息 + 路径 + 子部门 + 直属成员(前 N) + 递归人数。
// 返回 nil 表示部门不存在。
func (s *DepartmentService) GetDetail(ctx context.Context, deptID string) (*domain.DepartmentDetail, error) {
	if deptID == "" {
		return nil, nil
	}
	detail, err := s.repo.GetDetail(ctx, deptID, defaultDirectMemberLimit)
	if err != nil {
		return nil, fmt.Errorf("department.GetDetail: %w", err)
	}
	return detail, nil
}
