-- 弃用 users.roles 字段：角色关系已迁至 user_roles 表
-- 字段保留过渡两版后由后续迁移移除，避免和正在运行的 JWT 兼容代码冲突
COMMENT ON COLUMN users.roles IS 'DEPRECATED: 角色关系已迁至 user_roles 表；本字段保留过渡两版后移除';
