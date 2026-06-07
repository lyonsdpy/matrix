package service

import (
	"context"
	"fmt"

	"matrix/api/internal/repository/pg_repo"
	"matrix/api/pkg/perm"
)

// PermissionView 权限码视图，给前端按 module 分组展示。
type PermissionView struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Module     string `json:"module"`
	Kind       string `json:"kind"` // page | action
	ParentCode string `json:"parent_code"`
	Sort       int    `json:"sort"`
}

// PermissionChecker 中间件用的最小决策接口。
// 单独定义而非复用 PermissionService 完整接口，遵循"接口定义在使用方"。
type PermissionChecker interface {
	UserHasAny(ctx context.Context, userID string, codes []string) (bool, error)
	IsAdmin(ctx context.Context, userID string) (bool, error)
}

// permissionRepo 是 PermissionService 所需的 repo 接口（窄）
type permissionRepoIface interface {
	ListAll(ctx context.Context) ([]pg_repo.Permission, error)
}

// userRoleRepoIface 权限决策与"当前用户权限码"查询所需的窄接口
type userRoleRepoIface interface {
	ListPermissionCodesByUserID(ctx context.Context, userID string) ([]string, error)
	HasRoleCode(ctx context.Context, userID, roleCode string) (bool, error)
	HasAnyPermission(ctx context.Context, userID string, codes []string) (bool, error)
}

type PermissionService struct {
	permRepo permissionRepoIface
	urRepo   userRoleRepoIface
}

func NewPermissionService(permRepo permissionRepoIface, urRepo userRoleRepoIface) *PermissionService {
	return &PermissionService{permRepo: permRepo, urRepo: urRepo}
}

// ListAll 列出所有权限码，按 sort 升序，给前端构建权限树
func (s *PermissionService) ListAll(ctx context.Context) ([]PermissionView, error) {
	rows, err := s.permRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PermissionView, len(rows))
	for i, r := range rows {
		out[i] = PermissionView{
			Code:       r.Code,
			Name:       r.Name,
			Module:     r.Module,
			Kind:       r.Kind,
			ParentCode: r.ParentCode,
			Sort:       r.Sort,
		}
	}
	return out, nil
}

// MyPermissions 当前用户拥有的所有权限码集合。
// admin 角色用户：返回全部权限码（与中间件特判行为一致，前端按钮显隐才一致）
func (s *PermissionService) MyPermissions(ctx context.Context, userID string) ([]string, bool, error) {
	isAdmin, err := s.urRepo.HasRoleCode(ctx, userID, perm.RoleCodeAdmin)
	if err != nil {
		return nil, false, fmt.Errorf("permission svc: check admin: %w", err)
	}
	if isAdmin {
		// admin 走特判：返回所有代码维护的权限码（库里同步过的应一致，以代码为准）
		out := make([]string, 0, len(perm.Codes))
		for _, p := range perm.Codes {
			out = append(out, p.Code)
		}
		return out, true, nil
	}
	codes, err := s.urRepo.ListPermissionCodesByUserID(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("permission svc: list user perms: %w", err)
	}
	return codes, false, nil
}

// UserHasAny 用户是否拥有 codes 中的任意一个权限码（admin 直接放行）
func (s *PermissionService) UserHasAny(ctx context.Context, userID string, codes []string) (bool, error) {
	if len(codes) == 0 {
		return true, nil
	}
	isAdmin, err := s.urRepo.HasRoleCode(ctx, userID, perm.RoleCodeAdmin)
	if err != nil {
		return false, fmt.Errorf("permission svc: check admin: %w", err)
	}
	if isAdmin {
		return true, nil
	}
	return s.urRepo.HasAnyPermission(ctx, userID, codes)
}

// IsAdmin 当前用户是否拥有 admin 角色。中间件特判逻辑通过 UserHasAny 已覆盖，
// 此方法给 handler 层做 UI 特殊渲染判断（如"系统管理员"占位）
func (s *PermissionService) IsAdmin(ctx context.Context, userID string) (bool, error) {
	return s.urRepo.HasRoleCode(ctx, userID, perm.RoleCodeAdmin)
}
