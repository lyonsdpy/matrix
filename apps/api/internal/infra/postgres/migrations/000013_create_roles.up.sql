-- 角色表：管理员可在 UI 自由创建/编辑/删除（is_system=true 的不可删）
CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        TEXT UNIQUE NOT NULL,            -- 机器名（小写下划线，仅创建时指定，不可改）
    name        TEXT NOT NULL,                   -- 显示名（可改）
    description TEXT NOT NULL DEFAULT '',
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,  -- admin/viewer 内置，不可删
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_roles_code ON roles (code);
