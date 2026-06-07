package pg_repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// AuthUser 本地鉴权用户，存储在 PostgreSQL users 表。
// 与 domain.User（图节点）无关，是系统自身的账号体系。
type AuthUser struct {
	ID           string
	Username     string
	PasswordHash string
	LarkOpenID   string // 飞书 open_id，空字符串表示非飞书用户
	Roles        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuthUserRepository interface {
	FindByUsername(ctx context.Context, username string) (*AuthUser, error)
	FindByLarkOpenID(ctx context.Context, openID string) (*AuthUser, error)
	FindByID(ctx context.Context, id string) (*AuthUser, error)
	Create(ctx context.Context, username, passwordHash string, roles []string) (*AuthUser, error)
	// CreateLarkUser 创建飞书扫码用户，password_hash 设为空字符串（不允许密码登录）。
	CreateLarkUser(ctx context.Context, username, larkOpenID string, roles []string) (*AuthUser, error)
	UpdatePassword(ctx context.Context, username, newPasswordHash string) error
}

type authUserRepo struct {
	db *sqlx.DB
}

func NewAuthUserRepository(db *sqlx.DB) AuthUserRepository {
	return &authUserRepo{db: db}
}

type authUserRow struct {
	ID           string         `db:"id"`
	Username     string         `db:"username"`
	PasswordHash string         `db:"password_hash"`
	LarkOpenID   sql.NullString `db:"lark_open_id"`
	Roles        string         `db:"roles"` // JSONB → JSON string
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func (r *authUserRepo) FindByUsername(ctx context.Context, username string) (*AuthUser, error) {
	var row authUserRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, username, password_hash, lark_open_id, roles, created_at, updated_at FROM users WHERE username = $1`,
		username,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth user repo: find by username: %w", err)
	}
	return toAuthUser(row)
}

func (r *authUserRepo) FindByID(ctx context.Context, id string) (*AuthUser, error) {
	var row authUserRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, username, password_hash, lark_open_id, roles, created_at, updated_at FROM users WHERE id = $1`,
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth user repo: find by id: %w", err)
	}
	return toAuthUser(row)
}

func (r *authUserRepo) FindByLarkOpenID(ctx context.Context, openID string) (*AuthUser, error) {
	var row authUserRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, username, password_hash, lark_open_id, roles, created_at, updated_at FROM users WHERE lark_open_id = $1`,
		openID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth user repo: find by lark open_id: %w", err)
	}
	return toAuthUser(row)
}

func (r *authUserRepo) Create(ctx context.Context, username, passwordHash string, roles []string) (*AuthUser, error) {
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, fmt.Errorf("auth user repo: marshal roles: %w", err)
	}
	var row authUserRow
	err = r.db.QueryRowxContext(ctx,
		`INSERT INTO users (username, password_hash, roles)
		 VALUES ($1, $2, $3::jsonb)
		 RETURNING id, username, password_hash, lark_open_id, roles, created_at, updated_at`,
		username, passwordHash, string(rolesJSON),
	).StructScan(&row)
	if err != nil {
		return nil, fmt.Errorf("auth user repo: create: %w", err)
	}
	return toAuthUser(row)
}

func (r *authUserRepo) CreateLarkUser(ctx context.Context, username, larkOpenID string, roles []string) (*AuthUser, error) {
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, fmt.Errorf("auth user repo: marshal roles: %w", err)
	}
	var row authUserRow
	// password_hash 为空字符串，bcrypt 对空 hash 的校验天然失败，确保飞书用户无法密码登录
	err = r.db.QueryRowxContext(ctx,
		`INSERT INTO users (username, password_hash, lark_open_id, roles)
		 VALUES ($1, '', $2, $3::jsonb)
		 RETURNING id, username, password_hash, lark_open_id, roles, created_at, updated_at`,
		username, larkOpenID, string(rolesJSON),
	).StructScan(&row)
	if err != nil {
		return nil, fmt.Errorf("auth user repo: create lark user: %w", err)
	}
	return toAuthUser(row)
}

func (r *authUserRepo) UpdatePassword(ctx context.Context, username, newPasswordHash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE username = $2`,
		newPasswordHash, username,
	)
	if err != nil {
		return fmt.Errorf("auth user repo: update password: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("auth user repo: user %q not found", username)
	}
	return nil
}

func toAuthUser(row authUserRow) (*AuthUser, error) {
	var roles []string
	if err := json.Unmarshal([]byte(row.Roles), &roles); err != nil {
		return nil, fmt.Errorf("auth user repo: unmarshal roles: %w", err)
	}
	if roles == nil {
		roles = []string{}
	}
	larkOpenID := ""
	if row.LarkOpenID.Valid {
		larkOpenID = row.LarkOpenID.String
	}
	return &AuthUser{
		ID:           row.ID,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		LarkOpenID:   larkOpenID,
		Roles:        roles,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}
