package service

import (
	"context"
	"fmt"
	"time"

	"matrix/api/internal/repository/pg_repo"
)

// 暴破防护参数：失败 5 次进入软锁，窗口 15 分钟。
// 滑动窗口语义：last_failed_at + window 内未续命，计数归零（不算入新阈值）。
// 为什么写常量而非 yaml 配置：内部系统、运维不太可能改、配置项越少越好。
const (
	loginFailWindow  = 15 * time.Minute
	loginFailMax     = 5
	loginLockDur     = 15 * time.Minute
)

// LoginGuard 登录暴破防护，组合 (ip|username) 联合计数，避免：
//   - 单 IP 锁定整个账号（DoS：攻击者用别人账号触发锁，挡住合法登录）
//   - 单账号被同 IP 慢速暴破
type LoginGuard struct {
	repo pg_repo.LoginAttemptRepository
}

func NewLoginGuard(repo pg_repo.LoginAttemptRepository) *LoginGuard {
	return &LoginGuard{repo: repo}
}

// CheckAllowed 登录前查软锁定状态。
// 注意：未锁定时 returnsallowed=true 且不增加任何计数（计数只在 RecordFailure 加）。
// 锁定但已过 locked_until 时间 → 允许（视为锁定自然到期，下次失败会重新累计）。
func (g *LoginGuard) CheckAllowed(ctx context.Context, ip, username string) (bool, error) {
	a, err := g.repo.Get(ctx, keyOf(ip, username))
	if err != nil {
		return false, fmt.Errorf("login guard: check: %w", err)
	}
	if a == nil {
		return true, nil
	}
	if !a.LockedUntil.IsZero() && a.LockedUntil.After(time.Now()) {
		return false, nil
	}
	return true, nil
}

// RecordFailure 失败时调用。返回新的 failed_count；达到阈值时自动写入软锁定截止时间。
func (g *LoginGuard) RecordFailure(ctx context.Context, ip, username string) error {
	key := keyOf(ip, username)
	count, err := g.repo.RecordFailure(ctx, key, username, loginFailWindow)
	if err != nil {
		return err
	}
	if count >= loginFailMax {
		// 软锁定：阻断该 (IP, username) 组合 loginLockDur 时间
		// 不影响其他 IP 或同 IP 的其他账号（admin 在别处仍可登录）
		if err := g.repo.Lock(ctx, key, time.Now().Add(loginLockDur)); err != nil {
			return err
		}
	}
	return nil
}

// RecordSuccess 成功登录时调用，清空该 (IP, username) 的计数与锁定
func (g *LoginGuard) RecordSuccess(ctx context.Context, ip, username string) error {
	return g.repo.Reset(ctx, keyOf(ip, username))
}

// UnlockUsername 跨 IP 解锁整个账号（CLI -unlock-admin 用）。
// 返回清理的记录数（便于 CLI 反馈结果）。
func (g *LoginGuard) UnlockUsername(ctx context.Context, username string) (int, error) {
	return g.repo.ResetByUsername(ctx, username)
}

func keyOf(ip, username string) string {
	return ip + "|" + username
}
