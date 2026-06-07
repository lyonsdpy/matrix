package pg_repo

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// UserRoleRepository 用户角色绑定关系的双向访问 + 权限决策核心查询。
// 角色 -> 用户 的查询能力在 RoleRepository（ListUsers），此处只关心用户视角。
type UserRoleRepository interface {
	// ListRolesByUserID 取用户绑定的所有角色（按 granted_at 排序）
	ListRolesByUserID(ctx context.Context, userID string) ([]Role, error)
	// ListPermissionCodesByUserID 取用户拥有的所有权限码（去重，所有角色 union）
	ListPermissionCodesByUserID(ctx context.Context, userID string) ([]string, error)
	// SetRolesByUserID 覆盖式：清空用户的所有角色后写入新集合
	SetRolesByUserID(ctx context.Context, userID string, roleIDs []string, grantedBy string) error
	// HasRoleCode 用户是否拥有指定 code 的角色（admin 特判用，单条查询）
	HasRoleCode(ctx context.Context, userID, roleCode string) (bool, error)
	// HasAnyPermission 用户是否拥有 codes 中的任意一个权限码（中间件核心决策）
	HasAnyPermission(ctx context.Context, userID string, codes []string) (bool, error)
}

type userRoleRepo struct {
	db *sqlx.DB
}

func NewUserRoleRepository(db *sqlx.DB) UserRoleRepository {
	return &userRoleRepo{db: db}
}

func (r *userRoleRepo) ListRolesByUserID(ctx context.Context, userID string) ([]Role, error) {
	var rows []roleRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT r.id, r.code, r.name, r.description, r.is_system, r.created_at, r.updated_at
		 FROM user_roles ur
		 JOIN roles r ON r.id = ur.role_id
		 WHERE ur.user_id = $1
		 ORDER BY ur.granted_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("user_role repo: list roles by user: %w", err)
	}
	out := make([]Role, len(rows))
	for i, row := range rows {
		out[i] = toRole(row)
	}
	return out, nil
}

func (r *userRoleRepo) ListPermissionCodesByUserID(ctx context.Context, userID string) ([]string, error) {
	var codes []string
	err := r.db.SelectContext(ctx, &codes,
		`SELECT DISTINCT rp.permission_code
		 FROM user_roles ur
		 JOIN role_permissions rp ON rp.role_id = ur.role_id
		 WHERE ur.user_id = $1
		 ORDER BY rp.permission_code`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("user_role repo: list permissions by user: %w", err)
	}
	return codes, nil
}

func (r *userRoleRepo) SetRolesByUserID(ctx context.Context, userID string, roleIDs []string, grantedBy string) error {
	var grantedByArg any
	if grantedBy != "" {
		grantedByArg = grantedBy
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("user_role repo: set roles: begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM user_roles WHERE user_id = $1`, userID,
	); err != nil {
		return fmt.Errorf("user_role repo: set roles: clear: %w", err)
	}

	for _, rid := range roleIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO user_roles (user_id, role_id, granted_by) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`,
			userID, rid, grantedByArg,
		); err != nil {
			return fmt.Errorf("user_role repo: set roles: insert %s: %w", rid, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("user_role repo: set roles: commit: %w", err)
	}
	return nil
}

func (r *userRoleRepo) HasRoleCode(ctx context.Context, userID, roleCode string) (bool, error) {
	var ok bool
	err := r.db.GetContext(ctx, &ok,
		`SELECT EXISTS(
		   SELECT 1 FROM user_roles ur
		   JOIN roles r ON r.id = ur.role_id
		   WHERE ur.user_id = $1 AND r.code = $2
		 )`,
		userID, roleCode,
	)
	if err != nil {
		return false, fmt.Errorf("user_role repo: has role code: %w", err)
	}
	return ok, nil
}

func (r *userRoleRepo) HasAnyPermission(ctx context.Context, userID string, codes []string) (bool, error) {
	if len(codes) == 0 {
		return false, nil
	}
	var ok bool
	err := r.db.GetContext(ctx, &ok,
		`SELECT EXISTS(
		   SELECT 1 FROM user_roles ur
		   JOIN role_permissions rp ON rp.role_id = ur.role_id
		   WHERE ur.user_id = $1 AND rp.permission_code = ANY($2)
		 )`,
		userID, pq.Array(codes),
	)
	if err != nil {
		return false, fmt.Errorf("user_role repo: has any permission: %w", err)
	}
	return ok, nil
}
