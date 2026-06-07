-- 账号密码登录失败计数 + 软锁定记录
-- key 用 (ip, username) 联合，避免单 IP 锁定整个账号（导致 DoS）以及
-- 单账号被同 IP 慢速暴破。PG 存而非内存：cmd/server -unlock-admin CLI 工具需共享状态。
CREATE TABLE IF NOT EXISTS login_attempts (
    key            TEXT PRIMARY KEY,           -- 形如 "ip|username"
    username       TEXT NOT NULL,              -- 冗余，便于按 username 解锁（跨 IP）
    failed_count   INT  NOT NULL DEFAULT 0,
    locked_until   TIMESTAMPTZ,                -- NULL 表示未锁定
    last_failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_login_attempts_username ON login_attempts (username);
CREATE INDEX IF NOT EXISTS idx_login_attempts_locked_until ON login_attempts (locked_until)
    WHERE locked_until IS NOT NULL;
