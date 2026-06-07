-- 权限码主表：代码权威列表（pkg/perm/codes.go），启动时 UPSERT 同步入库
-- UI 只读，按 module 分组、page→action 树形展示
CREATE TABLE IF NOT EXISTS permissions (
    code         TEXT PRIMARY KEY,                          -- 如 endpoint:read / endpoint:delete
    name         TEXT NOT NULL,                             -- 显示名："查看终端列表"
    module       TEXT NOT NULL,                             -- 分组键："endpoint"
    kind         TEXT NOT NULL,                             -- 'page' | 'action'
    parent_code  TEXT REFERENCES permissions(code) ON DELETE SET NULL,
    sort         INT  NOT NULL DEFAULT 0,
    CONSTRAINT permissions_kind_chk CHECK (kind IN ('page', 'action'))
);

CREATE INDEX IF NOT EXISTS idx_permissions_module ON permissions (module);
CREATE INDEX IF NOT EXISTS idx_permissions_parent ON permissions (parent_code);
