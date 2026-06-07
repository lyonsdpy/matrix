-- 用户与角色的多对多关联（一人可多角色）
-- 独立表而非用 users.roles JSONB：要支持"角色→用户反查"（管理页右栏）+ 授权时间审计
CREATE TABLE IF NOT EXISTS user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),       -- 授权人，便于审计；不级联，授权人删除时置 NULL
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles (role_id);
