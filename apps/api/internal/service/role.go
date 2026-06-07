package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"matrix/api/internal/repository/pg_repo"
	"matrix/api/pkg/perm"
)

// ErrRoleNotFound 角色不存在
var ErrRoleNotFound = errors.New("role not found")

// ErrRoleCodeConflict 角色 code 冲突
var ErrRoleCodeConflict = errors.New("role code already exists")

// ErrSystemRoleProtected 系统内置角色不可删/不可改 code
var ErrSystemRoleProtected = errors.New("system role is protected")

// ErrInvalidRoleCode 角色 code 不合法（仅允许小写字母、数字、下划线）
var ErrInvalidRoleCode = errors.New("invalid role code")

// ErrUnknownPermissionCode 引用的权限码不存在
var ErrUnknownPermissionCode = errors.New("unknown permission code")

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// RoleView 角色视图（列表 / 详情共用）
type RoleView struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	UserCount   int       `json:"user_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleDetail 角色详情：基础信息 + 已绑权限码（admin 角色为空，由前端特殊渲染）
type RoleDetail struct {
	RoleView
	PermissionCodes []string `json:"permission_codes"`
}

// RoleUserView 角色下用户视图
type RoleUserView struct {
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	LarkOpenID string    `json:"lark_open_id"`
	GrantedAt  time.Time `json:"granted_at"`
}

// roleRepoIface RoleService 所需的 role repo 接口（窄）
type roleRepoIface interface {
	List(ctx context.Context) ([]pg_repo.Role, error)
	Get(ctx context.Context, id string) (*pg_repo.Role, error)
	GetByCode(ctx context.Context, code string) (*pg_repo.Role, error)
	Create(ctx context.Context, code, name, description string) (*pg_repo.Role, error)
	Update(ctx context.Context, id, name, description string) (*pg_repo.Role, error)
	Delete(ctx context.Context, id string) error
	ListPermissionCodes(ctx context.Context, roleID string) ([]string, error)
	SetPermissionCodes(ctx context.Context, roleID string, codes []string) error
	ListUsers(ctx context.Context, roleID string, limit int) ([]pg_repo.RoleUser, error)
	CountUsers(ctx context.Context, roleID string) (int, error)
	AddUsers(ctx context.Context, roleID string, userIDs []string, grantedBy string) error
	AddUsersByLarkOpenIDs(ctx context.Context, roleID string, openIDs []string, grantedBy string) (int, error)
	RemoveUser(ctx context.Context, roleID, userID string) error
}

// urAssignIface RoleService 给用户授角色用的窄接口
type urAssignIface interface {
	SetRolesByUserID(ctx context.Context, userID string, roleIDs []string, grantedBy string) error
	ListRolesByUserID(ctx context.Context, userID string) ([]pg_repo.Role, error)
}

// authUserLookupIface RoleService 用来反查用户名做"内置账号保护"判断
type authUserLookupIface interface {
	FindByID(ctx context.Context, id string) (*pg_repo.AuthUser, error)
}

type RoleService struct {
	roleRepo     roleRepoIface
	urRepo       urAssignIface
	permRepo     permissionRepoIface
	authUserRepo authUserLookupIface
}

func NewRoleService(roleRepo roleRepoIface, urRepo urAssignIface, permRepo permissionRepoIface, authUserRepo authUserLookupIface) *RoleService {
	return &RoleService{roleRepo: roleRepo, urRepo: urRepo, permRepo: permRepo, authUserRepo: authUserRepo}
}

// List 角色列表（含每个角色的用户数）
func (s *RoleService) List(ctx context.Context) ([]RoleView, error) {
	roles, err := s.roleRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RoleView, len(roles))
	for i, r := range roles {
		// 用户数：内联 count，列表角色数量级很小（< 20），N+1 可以接受
		n, _ := s.roleRepo.CountUsers(ctx, r.ID)
		out[i] = toRoleView(r, n)
	}
	return out, nil
}

// Get 角色详情（含已绑权限码）
func (s *RoleService) Get(ctx context.Context, id string) (*RoleDetail, error) {
	r, err := s.roleRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	codes, err := s.roleRepo.ListPermissionCodes(ctx, id)
	if err != nil {
		return nil, err
	}
	n, _ := s.roleRepo.CountUsers(ctx, id)
	return &RoleDetail{
		RoleView:        toRoleView(*r, n),
		PermissionCodes: codes,
	}, nil
}

// Create 创建自定义角色（不允许复用 system code）
func (s *RoleService) Create(ctx context.Context, code, name, description string) (*RoleView, error) {
	code = strings.TrimSpace(code)
	if !roleCodePattern.MatchString(code) {
		return nil, ErrInvalidRoleCode
	}
	if code == perm.RoleCodeAdmin || code == perm.RoleCodeViewer {
		return nil, ErrRoleCodeConflict
	}
	r, err := s.roleRepo.Create(ctx, code, strings.TrimSpace(name), strings.TrimSpace(description))
	if err != nil {
		if errors.Is(err, pg_repo.ErrRoleCodeConflict) {
			return nil, ErrRoleCodeConflict
		}
		return nil, err
	}
	v := toRoleView(*r, 0)
	return &v, nil
}

// Update 改角色名称/描述（code 不可改；is_system 也允许改名以便本地化）
func (s *RoleService) Update(ctx context.Context, id, name, description string) (*RoleView, error) {
	r, err := s.roleRepo.Update(ctx, id, strings.TrimSpace(name), strings.TrimSpace(description))
	if err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	n, _ := s.roleRepo.CountUsers(ctx, id)
	v := toRoleView(*r, n)
	return &v, nil
}

// Delete 删除角色，is_system=true 的拒绝
func (s *RoleService) Delete(ctx context.Context, id string) error {
	r, err := s.roleRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if r.IsSystem {
		return ErrSystemRoleProtected
	}
	if err := s.roleRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	return nil
}

// SetPermissions 覆盖式设置角色权限码集合
// 校验：admin 角色不允许通过此接口修改（走中间件特判，无需绑权限）
func (s *RoleService) SetPermissions(ctx context.Context, roleID string, codes []string) error {
	r, err := s.roleRepo.Get(ctx, roleID)
	if err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if r.Code == perm.RoleCodeAdmin {
		// admin 走特判，禁止配置
		return ErrSystemRoleProtected
	}

	// 校验所有 code 都在 permissions 表中（防止前端传错）
	allPerms, err := s.permRepo.ListAll(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(allPerms))
	for _, p := range allPerms {
		known[p.Code] = struct{}{}
	}
	for _, c := range codes {
		if _, ok := known[c]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownPermissionCode, c)
		}
	}

	return s.roleRepo.SetPermissionCodes(ctx, roleID, codes)
}

// ListUsers 列出角色下的用户
func (s *RoleService) ListUsers(ctx context.Context, roleID string, limit int) ([]RoleUserView, error) {
	users, err := s.roleRepo.ListUsers(ctx, roleID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]RoleUserView, len(users))
	for i, u := range users {
		out[i] = RoleUserView{
			UserID:     u.UserID,
			Username:   u.Username,
			LarkOpenID: u.LarkOpenID,
			GrantedAt:  u.GrantedAt,
		}
	}
	return out, nil
}

// AddUsers 批量给角色添加用户（grantedBy 为操作者 user_id，可空）
func (s *RoleService) AddUsers(ctx context.Context, roleID string, userIDs []string, grantedBy string) error {
	if _, err := s.roleRepo.Get(ctx, roleID); err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	return s.roleRepo.AddUsers(ctx, roleID, userIDs, grantedBy)
}

// AddUsersByLarkOpenIDs 通过飞书 open_id 批量加成员。
// 返回实际加入人数；不在 PG users 白名单的 open_id 静默忽略（前端可用差值提示）
func (s *RoleService) AddUsersByLarkOpenIDs(ctx context.Context, roleID string, openIDs []string, grantedBy string) (int, error) {
	if _, err := s.roleRepo.Get(ctx, roleID); err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return 0, ErrRoleNotFound
		}
		return 0, err
	}
	return s.roleRepo.AddUsersByLarkOpenIDs(ctx, roleID, openIDs, grantedBy)
}

// RemoveUser 解绑某用户。
// 兜底防护：admin 本地账号不可从 admin 角色解绑——避免系统失去管理员入口
func (s *RoleService) RemoveUser(ctx context.Context, roleID, userID string) error {
	role, err := s.roleRepo.Get(ctx, roleID)
	if err != nil {
		if errors.Is(err, pg_repo.ErrRoleNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if role.Code == perm.RoleCodeAdmin {
		user, err := s.authUserRepo.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		if user != nil && user.Username == "admin" {
			return ErrSystemRoleProtected
		}
	}
	return s.roleRepo.RemoveUser(ctx, roleID, userID)
}

// SetUserRoles 用户管理页：覆盖式设置某用户的角色集合
func (s *RoleService) SetUserRoles(ctx context.Context, userID string, roleIDs []string, grantedBy string) error {
	// 校验所有 roleID 均存在
	for _, rid := range roleIDs {
		if _, err := s.roleRepo.Get(ctx, rid); err != nil {
			if errors.Is(err, pg_repo.ErrRoleNotFound) {
				return fmt.Errorf("%w: %s", ErrRoleNotFound, rid)
			}
			return err
		}
	}
	return s.urRepo.SetRolesByUserID(ctx, userID, roleIDs, grantedBy)
}

// GetUserRoles 用户管理页：取某用户绑定的角色集合
func (s *RoleService) GetUserRoles(ctx context.Context, userID string) ([]RoleView, error) {
	roles, err := s.urRepo.ListRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]RoleView, len(roles))
	for i, r := range roles {
		// 这里不查 CountUsers，user 视角下用户数无意义
		out[i] = toRoleView(r, 0)
	}
	return out, nil
}

func toRoleView(r pg_repo.Role, userCount int) RoleView {
	return RoleView{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		UserCount:   userCount,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
