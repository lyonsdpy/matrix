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
	Roles        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuthUserRepository interface {
	FindByUsername(ctx context.Context, username string) (*AuthUser, error)
	Create(ctx context.Context, username, passwordHash string, roles []string) (*AuthUser, error)
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
	Roles        string         `db:"roles"` // JSONB → JSON string
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func (r *authUserRepo) FindByUsername(ctx context.Context, username string) (*AuthUser, error) {
	var row authUserRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, username, password_hash, roles, created_at, updated_at FROM users WHERE username = $1`,
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

func (r *authUserRepo) Create(ctx context.Context, username, passwordHash string, roles []string) (*AuthUser, error) {
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, fmt.Errorf("auth user repo: marshal roles: %w", err)
	}
	var row authUserRow
	err = r.db.QueryRowxContext(ctx,
		`INSERT INTO users (username, password_hash, roles)
		 VALUES ($1, $2, $3::jsonb)
		 RETURNING id, username, password_hash, roles, created_at, updated_at`,
		username, passwordHash, string(rolesJSON),
	).StructScan(&row)
	if err != nil {
		return nil, fmt.Errorf("auth user repo: create: %w", err)
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
	return &AuthUser{
		ID:           row.ID,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		Roles:        roles,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}
