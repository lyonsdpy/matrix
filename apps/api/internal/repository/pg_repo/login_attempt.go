package pg_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// LoginAttemptRepository 账号密码登录失败计数 + 软锁定。
// 数据放 PG 而非内存：cmd/server CLI 工具（-unlock-admin / -init-admin-password）
// 需要在 server 进程外清理状态，跨进程必须落库。
type LoginAttemptRepository interface {
	// Get 取当前 (ip|username) 的记录；不存在返回 nil
	Get(ctx context.Context, key string) (*LoginAttempt, error)
	// RecordFailure 失败时调用：window 内 counter++；超出 window 重置为 1
	// 返回更新后的 failed_count，调用方据此决定是否锁定
	RecordFailure(ctx context.Context, key, username string, window time.Duration) (int, error)
	// Lock 写入软锁定截止时间
	Lock(ctx context.Context, key string, until time.Time) error
	// Reset 清空单条记录（成功登录、或对该 (ip|username) 解锁）
	Reset(ctx context.Context, key string) error
	// ResetByUsername 清空某账号的所有 IP 记录（admin 跨 IP 解锁）
	ResetByUsername(ctx context.Context, username string) (int, error)
}

// LoginAttempt 登录失败记录
type LoginAttempt struct {
	Key          string
	Username     string
	FailedCount  int
	LockedUntil  time.Time // 零值表示未锁定
	LastFailedAt time.Time
}

type loginAttemptRepo struct {
	db *sqlx.DB
}

func NewLoginAttemptRepository(db *sqlx.DB) LoginAttemptRepository {
	return &loginAttemptRepo{db: db}
}

type loginAttemptRow struct {
	Key          string       `db:"key"`
	Username     string       `db:"username"`
	FailedCount  int          `db:"failed_count"`
	LockedUntil  sql.NullTime `db:"locked_until"`
	LastFailedAt time.Time    `db:"last_failed_at"`
}

func (r *loginAttemptRepo) Get(ctx context.Context, key string) (*LoginAttempt, error) {
	var row loginAttemptRow
	err := r.db.GetContext(ctx, &row,
		`SELECT key, username, failed_count, locked_until, last_failed_at
		 FROM login_attempts WHERE key = $1`, key,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("login_attempt repo: get: %w", err)
	}
	la := &LoginAttempt{
		Key:          row.Key,
		Username:     row.Username,
		FailedCount:  row.FailedCount,
		LastFailedAt: row.LastFailedAt,
	}
	if row.LockedUntil.Valid {
		la.LockedUntil = row.LockedUntil.Time
	}
	return la, nil
}

// RecordFailure 用单条 UPSERT 完成失败计数：
//   - 已有记录且 last_failed_at + window 未过期 → counter++
//   - 已有记录但 window 已过 → counter 重置为 1（旧窗口的失败不再累计）
//   - 无记录 → 插入 counter=1
//
// 整个判断在 SQL 内完成，避免读-修-写竞态。
func (r *loginAttemptRepo) RecordFailure(ctx context.Context, key, username string, window time.Duration) (int, error) {
	var newCount int
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO login_attempts (key, username, failed_count, last_failed_at)
		 VALUES ($1, $2, 1, NOW())
		 ON CONFLICT (key) DO UPDATE
		   SET failed_count = CASE
		         WHEN login_attempts.last_failed_at + make_interval(secs => $3) < NOW()
		           THEN 1
		         ELSE login_attempts.failed_count + 1
		       END,
		       last_failed_at = NOW(),
		       username = EXCLUDED.username
		 RETURNING failed_count`,
		key, username, int(window.Seconds()),
	).Scan(&newCount)
	if err != nil {
		return 0, fmt.Errorf("login_attempt repo: record failure: %w", err)
	}
	return newCount, nil
}

func (r *loginAttemptRepo) Lock(ctx context.Context, key string, until time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE login_attempts SET locked_until = $2 WHERE key = $1`,
		key, until,
	)
	if err != nil {
		return fmt.Errorf("login_attempt repo: lock: %w", err)
	}
	return nil
}

func (r *loginAttemptRepo) Reset(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM login_attempts WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("login_attempt repo: reset: %w", err)
	}
	return nil
}

func (r *loginAttemptRepo) ResetByUsername(ctx context.Context, username string) (int, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM login_attempts WHERE username = $1`, username)
	if err != nil {
		return 0, fmt.Errorf("login_attempt repo: reset by username: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
