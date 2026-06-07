package pg_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Role 角色记录。is_system=true 的不可删除（系统内置）。
type Role struct {
	ID          string
	Code        string
	Name        string
	Description string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RoleUser 角色下的用户记录，用于角色管理页右栏展示。
type RoleUser struct {
	UserID     string
	Username   string
	LarkOpenID string // 空字符串表示非飞书用户（如本地 admin）
	GrantedAt  time.Time
}

type RoleRepository interface {
	List(ctx context.Context) ([]Role, error)
	Get(ctx context.Context, id string) (*Role, error)
	GetByCode(ctx context.Context, code string) (*Role, error)
	Create(ctx context.Context, code, name, description string) (*Role, error)
	Update(ctx context.Context, id, name, description string) (*Role, error)
	Delete(ctx context.Context, id string) error

	// ── 权限码绑定 ────────────────────────────────────────────────────────
	ListPermissionCodes(ctx context.Context, roleID string) ([]string, error)
	// SetPermissionCodes 覆盖式：先清空再写入；空切片代表无权限
	SetPermissionCodes(ctx context.Context, roleID string, codes []string) error

	// ── 用户绑定 ──────────────────────────────────────────────────────────
	ListUsers(ctx context.Context, roleID string, limit int) ([]RoleUser, error)
	CountUsers(ctx context.Context, roleID string) (int, error)
	AddUsers(ctx context.Context, roleID string, userIDs []string, grantedBy string) error
	// AddUsersByLarkOpenIDs 通过飞书 open_id 批量加成员。
	// 内部 JOIN users 表把 open_id → user_id；不在 PG users 白名单的 open_id 静默忽略。
	// 返回实际加入的人数（命中白名单的 open_id 数）。
	AddUsersByLarkOpenIDs(ctx context.Context, roleID string, openIDs []string, grantedBy string) (int, error)
	RemoveUser(ctx context.Context, roleID, userID string) error
}

// ErrRoleNotFound 角色不存在
var ErrRoleNotFound = errors.New("role not found")

// ErrRoleCodeConflict 角色 code 冲突
var ErrRoleCodeConflict = errors.New("role code already exists")

type roleRepo struct {
	db *sqlx.DB
}

func NewRoleRepository(db *sqlx.DB) RoleRepository {
	return &roleRepo{db: db}
}

type roleRow struct {
	ID          string    `db:"id"`
	Code        string    `db:"code"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	IsSystem    bool      `db:"is_system"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (r *roleRepo) List(ctx context.Context) ([]Role, error) {
	var rows []roleRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, code, name, description, is_system, created_at, updated_at
		 FROM roles ORDER BY is_system DESC, created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("role repo: list: %w", err)
	}
	out := make([]Role, len(rows))
	for i, row := range rows {
		out[i] = toRole(row)
	}
	return out, nil
}

func (r *roleRepo) Get(ctx context.Context, id string) (*Role, error) {
	var row roleRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, code, name, description, is_system, created_at, updated_at
		 FROM roles WHERE id = $1`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("role repo: get: %w", err)
	}
	role := toRole(row)
	return &role, nil
}

func (r *roleRepo) GetByCode(ctx context.Context, code string) (*Role, error) {
	var row roleRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, code, name, description, is_system, created_at, updated_at
		 FROM roles WHERE code = $1`,
		code,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("role repo: get by code: %w", err)
	}
	role := toRole(row)
	return &role, nil
}

func (r *roleRepo) Create(ctx context.Context, code, name, description string) (*Role, error) {
	var row roleRow
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO roles (code, name, description, is_system)
		 VALUES ($1, $2, $3, FALSE)
		 RETURNING id, code, name, description, is_system, created_at, updated_at`,
		code, name, description,
	).StructScan(&row)
	if err != nil {
		// 23505 = unique_violation
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrRoleCodeConflict
		}
		return nil, fmt.Errorf("role repo: create: %w", err)
	}
	role := toRole(row)
	return &role, nil
}

func (r *roleRepo) Update(ctx context.Context, id, name, description string) (*Role, error) {
	var row roleRow
	err := r.db.QueryRowxContext(ctx,
		`UPDATE roles SET name = $2, description = $3, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, code, name, description, is_system, created_at, updated_at`,
		id, name, description,
	).StructScan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("role repo: update: %w", err)
	}
	role := toRole(row)
	return &role, nil
}

func (r *roleRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("role repo: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRoleNotFound
	}
	return nil
}

func (r *roleRepo) ListPermissionCodes(ctx context.Context, roleID string) ([]string, error) {
	var codes []string
	err := r.db.SelectContext(ctx, &codes,
		`SELECT permission_code FROM role_permissions WHERE role_id = $1 ORDER BY permission_code`,
		roleID,
	)
	if err != nil {
		return nil, fmt.Errorf("role repo: list permission codes: %w", err)
	}
	return codes, nil
}

func (r *roleRepo) SetPermissionCodes(ctx context.Context, roleID string, codes []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("role repo: set perms: begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM role_permissions WHERE role_id = $1`, roleID,
	); err != nil {
		return fmt.Errorf("role repo: set perms: clear: %w", err)
	}

	for _, code := range codes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO role_permissions (role_id, permission_code) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			roleID, code,
		); err != nil {
			return fmt.Errorf("role repo: set perms: insert %s: %w", code, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("role repo: set perms: commit: %w", err)
	}
	return nil
}

func (r *roleRepo) ListUsers(ctx context.Context, roleID string, limit int) ([]RoleUser, error) {
	if limit <= 0 {
		limit = 500
	}
	type row struct {
		UserID     string         `db:"user_id"`
		Username   string         `db:"username"`
		LarkOpenID sql.NullString `db:"lark_open_id"`
		GrantedAt  time.Time      `db:"granted_at"`
	}
	var rows []row
	err := r.db.SelectContext(ctx, &rows,
		`SELECT u.id AS user_id, u.username, u.lark_open_id, ur.granted_at
		 FROM user_roles ur
		 JOIN users u ON u.id = ur.user_id
		 WHERE ur.role_id = $1
		 ORDER BY ur.granted_at DESC
		 LIMIT $2`,
		roleID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("role repo: list users: %w", err)
	}
	out := make([]RoleUser, len(rows))
	for i, x := range rows {
		out[i] = RoleUser{
			UserID:     x.UserID,
			Username:   x.Username,
			LarkOpenID: nsToString(x.LarkOpenID),
			GrantedAt:  x.GrantedAt,
		}
	}
	return out, nil
}

func (r *roleRepo) CountUsers(ctx context.Context, roleID string) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM user_roles WHERE role_id = $1`, roleID,
	)
	if err != nil {
		return 0, fmt.Errorf("role repo: count users: %w", err)
	}
	return n, nil
}

func (r *roleRepo) AddUsers(ctx context.Context, roleID string, userIDs []string, grantedBy string) error {
	if len(userIDs) == 0 {
		return nil
	}
	var grantedByArg any
	if grantedBy != "" {
		grantedByArg = grantedBy
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("role repo: add users: begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, uid := range userIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO user_roles (user_id, role_id, granted_by) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`,
			uid, roleID, grantedByArg,
		); err != nil {
			return fmt.Errorf("role repo: add user %s: %w", uid, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("role repo: add users: commit: %w", err)
	}
	return nil
}

func (r *roleRepo) AddUsersByLarkOpenIDs(ctx context.Context, roleID string, openIDs []string, grantedBy string) (int, error) {
	if len(openIDs) == 0 {
		return 0, nil
	}
	var grantedByArg any
	if grantedBy != "" {
		grantedByArg = grantedBy
	}
	// JOIN users 表把 open_id 转 user_id，写入 user_roles
	// ON CONFLICT 保证幂等（重复添加不报错也不重复计数）
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO user_roles (user_id, role_id, granted_by)
		 SELECT u.id, $1, $2 FROM users u WHERE u.lark_open_id = ANY($3)
		 ON CONFLICT DO NOTHING`,
		roleID, grantedByArg, pq.Array(openIDs),
	)
	if err != nil {
		return 0, fmt.Errorf("role repo: add users by open_id: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *roleRepo) RemoveUser(ctx context.Context, roleID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_roles WHERE role_id = $1 AND user_id = $2`,
		roleID, userID,
	)
	if err != nil {
		return fmt.Errorf("role repo: remove user: %w", err)
	}
	return nil
}

func toRole(row roleRow) Role {
	return Role{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
