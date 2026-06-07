# docs/authz/

Matrix 系统的角色权限管理（RBAC）设计文档。本文档是系统行为的规格说明，`apps/api` 的实现必须与此保持一致；新增资源/操作时，先更新本文档，再修改代码。

---

## 1. 目标与范围

构建一套**管理员可视化运维**的角色权限体系，支持：

- 在 UI 中自由创建/编辑/删除**角色**（系统内置角色除外）
- 从**已同步的飞书用户**中手工选择并授予角色（不做部门自动绑定）
- 第一期权限粒度到**页面**，但数据结构与命名规范**为更细粒度（按钮/动作级）做好准备**——未来扩展无需重构表结构

**不在本期范围**：数据范围过滤（ABAC，如"只看本部门"）、组织架构自动派权、跨租户隔离。

---

## 2. 核心概念

```
User ──user_roles──▶ Role ──role_permissions──▶ Permission
                                                 │
                                                 ├─ kind=page    （页面入口，第一期使用）
                                                 └─ kind=action  （操作粒度，预留）
```

- **User**：`users` 表中的本地账号（与飞书 `open_id` 关联）
- **Role**：可在 UI 自由创建的命名集合（`admin`/`viewer` 为系统内置不可删）
- **Permission**：权限码（代码维护权威列表，UI 只读），格式 `<module>:<scope>` 或 `<module>:<sub>:<action>`
- **Permission.parent_code**：构建权限码层级树，UI 树形展示及"勾父全选子"操作的基础

---

## 3. 数据模型

### 3.1 新增 4 张表

迁移文件位于 `apps/api/internal/infra/postgres/migrations/`，编号续接已有最大序号（000011）。

#### `000012_create_permissions`

```sql
-- 权限码主表。代码维护权威列表（pkg/perm/codes.go），启动时 UPSERT 入库
-- UI 只读，按 module 分组、page→action 树形展示
CREATE TABLE IF NOT EXISTS permissions (
    code         TEXT PRIMARY KEY,                          -- 如 endpoint:read / endpoint:delete
    name         TEXT NOT NULL,                             -- 显示名："查看终端列表"
    module       TEXT NOT NULL,                             -- 分组键："endpoint"
    kind         TEXT NOT NULL,                             -- 'page' | 'action'
    parent_code  TEXT REFERENCES permissions(code) ON DELETE SET NULL,
    sort         INT  NOT NULL DEFAULT 0,
    CONSTRAINT permissions_kind_chk CHECK (kind IN ('page','action'))
);
CREATE INDEX IF NOT EXISTS idx_permissions_module ON permissions(module);
CREATE INDEX IF NOT EXISTS idx_permissions_parent ON permissions(parent_code);
```

#### `000013_create_roles`

```sql
-- 角色：管理员可在 UI 自由创建/编辑/删除（is_system=true 的不可删）
CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        TEXT UNIQUE NOT NULL,            -- 机器名（小写下划线，仅创建时指定）
    name        TEXT NOT NULL,                   -- 显示名（可改）
    description TEXT NOT NULL DEFAULT '',
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,  -- admin/viewer 内置，不可删
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `000014_create_role_permissions`

```sql
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_code TEXT NOT NULL REFERENCES permissions(code) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_code)
);
CREATE INDEX IF NOT EXISTS idx_role_perms_perm ON role_permissions(permission_code);
```

#### `000015_create_user_roles`

```sql
-- 一人多角色；独立表而非 users.roles JSONB
-- 为什么：要支持"反查角色下的用户"（角色管理右栏）+ "授权时间审计"
CREATE TABLE IF NOT EXISTS user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),       -- 谁授的，便于审计
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);
```

### 3.2 旧字段处理

`users.roles JSONB` 字段：迁移 `000016` 加 deprecated 注释，**保留过渡两版**，新代码统一从 `user_roles` 取角色。`daipengyuan` 通过 seed 写入 `user_roles` 绑定 `admin`。

---

## 4. 权限码命名规范

### 4.1 格式

```
<module>:<scope>             # 页面级，如 endpoint:read
<module>:<sub>:<action>      # 动作级，如 user:role:assign
```

**冒号分层**，全小写下划线。`parent_code` 字段构建层级关系（树）。

### 4.2 权威列表

代码位置：`apps/api/pkg/perm/codes.go`。系统启动时 `UPSERT` 同步入 `permissions` 表，UI 只读。

```go
// apps/api/pkg/perm/codes.go
var Codes = []Permission{
    // 模块 endpoint（终端管理）
    {Code: "endpoint:read",   Module: "endpoint", Kind: "page",   Name: "终端管理-查看"},
    {Code: "endpoint:write",  Module: "endpoint", Kind: "action", Parent: "endpoint:read", Name: "终端-编辑"},
    {Code: "endpoint:delete", Module: "endpoint", Kind: "action", Parent: "endpoint:read", Name: "终端-删除"},
    {Code: "endpoint:sync",   Module: "endpoint", Kind: "action", Parent: "endpoint:read", Name: "终端-触发同步"},

    // 模块 contact（飞书通讯录）
    {Code: "contact:read",    Module: "contact",  Kind: "page",   Name: "通讯录-查看"},
    {Code: "contact:sync",    Module: "contact",  Kind: "action", Parent: "contact:read", Name: "通讯录-触发同步"},

    // 模块 user（系统用户授权管理，不是飞书用户）
    {Code: "user:read",       Module: "user",     Kind: "page",   Name: "用户授权-查看"},
    {Code: "user:role:assign",Module: "user",     Kind: "action", Parent: "user:read", Name: "用户授权-分配角色"},

    // 模块 role（角色管理）
    {Code: "role:read",       Module: "role",     Kind: "page",   Name: "角色管理-查看"},
    {Code: "role:write",      Module: "role",     Kind: "action", Parent: "role:read", Name: "角色-创建编辑"},
    {Code: "role:delete",     Module: "role",     Kind: "action", Parent: "role:read", Name: "角色-删除"},
    {Code: "role:assign",     Module: "role",     Kind: "action", Parent: "role:read", Name: "角色-绑定权限/用户"},

    // 其他页面（待补：cloud_network / dumb_terminal / network_device 同模式）
}
```

### 4.3 第一期实际使用范围

| 阶段 | UI 展示 | 后端校验 |
|---|---|---|
| 第一期 | 仅 `kind=page` 节点（action 节点隐藏，避免配置死权限） | 仅页面级路由（`*:read`）+ 角色管理本身的 action |
| 后续扩展 | `kind=action` 节点同步展示 | 给对应路由加 `RequirePermission("xxx:delete")` 即可 |

**前端、后端、数据库都不用改结构**——这是 `kind` 字段的核心价值。

---

## 5. 后端实现

### 5.1 三层结构（沿用既有模式）

```
apps/api/
├── pkg/perm/codes.go                   # 权限码权威列表 + 启动同步
├── internal/repository/pg_repo/
│   ├── role.go                          # roles CRUD
│   ├── permission.go                    # permissions 读取 + UPSERT 同步
│   └── user_role.go                     # user_roles 关联管理
├── internal/service/
│   ├── role.go                          # 角色业务逻辑
│   └── permission.go                    # 含 UserHasAny / UserPermissions
├── internal/middleware/
│   └── perm.go                          # RequirePermission(codes...)
└── internal/handler/
    ├── role.go
    ├── permission.go
    └── handler.go                       # 路由注册 + 全局挂中间件
```

### 5.2 路由清单

| 方法 | 路径 | 中间件 | 用途 |
|---|---|---|---|
| GET | `/api/v1/permissions` | `role:read` | 权限码树（按 module 分组） |
| GET | `/api/v1/roles` | `role:read` | 角色列表 |
| POST | `/api/v1/roles` | `role:write` | 创建角色 |
| GET | `/api/v1/roles/:id` | `role:read` | 详情（含已绑权限码 + 已绑用户数） |
| PUT | `/api/v1/roles/:id` | `role:write` | 改名称/描述 |
| DELETE | `/api/v1/roles/:id` | `role:delete` | 删除（`is_system=true` 拒绝） |
| PUT | `/api/v1/roles/:id/permissions` | `role:assign` | 覆盖式设置该角色的权限码集合 |
| GET | `/api/v1/roles/:id/users` | `role:read` | 该角色下的用户（分页） |
| POST | `/api/v1/roles/:id/users` | `role:assign` | 批量给该角色添加用户 |
| DELETE | `/api/v1/roles/:id/users/:userId` | `role:assign` | 解绑某用户 |
| PUT | `/api/v1/users/:id/roles` | `user:role:assign` | 在用户管理页改某人的角色集合 |
| GET | `/api/v1/me/permissions` | 仅 `Auth` | 当前用户权限码集合（前端按钮显隐用） |

**白名单制**：所有业务接口必须显式挂 `RequirePermission`；未挂中间件的接口默认全员可访问，但**视为漏挂 bug**，code review 强制检查。

### 5.3 中间件

```go
// apps/api/internal/middleware/perm.go
func RequirePermission(svc service.PermissionChecker, codes ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        u, ok := auth.UserFrom(c.Request.Context())
        if !ok {
            c.AbortWithStatusJSON(401, gin.H{"error": "not authenticated"})
            return
        }
        // admin 角色特判，避免每次给 admin 同步全量权限码
        if hasRole(u, "admin") {
            c.Next()
            return
        }
        ok, err := svc.UserHasAny(c.Request.Context(), u.ID, codes)
        if err != nil {
            c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
            return
        }
        if !ok {
            c.AbortWithStatusJSON(403, gin.H{"error": "forbidden", "need": codes})
            return
        }
        c.Next()
    }
}
```

`UserHasAny` 单次 SQL：

```sql
SELECT EXISTS(
    SELECT 1
    FROM user_roles ur
    JOIN role_permissions rp ON rp.role_id = ur.role_id
    WHERE ur.user_id = $1
      AND rp.permission_code = ANY($2)
)
```

PG 索引覆盖，亚毫秒级。第一期**不引入缓存**（10 人级用户量，过度优化）。

### 5.4 admin 角色处理（B 方案：中间件特判）

中间件最前置 `if hasRole(u, "admin") { c.Next(); return }`。

- **为什么选 B 而非 A**：避免新增权限码时需要同时维护 admin 的 `role_permissions`；admin 是"绝对管理员"语义，特判更直接，0 数据漂移风险
- **副作用**：UI 上 admin 角色的"已绑权限"列表为空（因为不走 role_permissions）。处理方式：前端在角色详情页对 `code='admin'` 的角色显示"系统管理员，拥有所有权限"提示，不显示权限勾选框
- **如何判断 hasRole(u, "admin")**：每次中间件查 `user_roles JOIN roles WHERE u.id=? AND roles.code='admin'`；或登录时把"是否 admin"塞进 JWT 作为静态标志（**选前者**，与"JWT 不做权限决策真相源"一致）

### 5.5 JWT 与权限的关系

JWT 只证明"你是谁"（`uid`/`sub`），**不再携带权限信息**。

- **为什么**：角色/权限随时可改，缓存在 JWT 里会导致"撤权要等 token 过期"
- 现有 `Claims.Roles` 字段保留向后兼容**但停止使用**；下一版 JWT 签发可移除
- 中间件每次实时查库做权限决策

---

## 6. 前端实现

### 6.1 `/roles` 页（三栏布局）

```
+---------------------+----------------------------+----------------------+
| 角色列表（左 220）  | 权限勾选树（中，flex-1）   | 该角色下用户（右）   |
| [+ 新建角色]        | 选中"运维"角色后展示：     | [+ 添加用户]         |
|                     |                            |                      |
| · admin (系统)      | ▾ 通讯录                   | 张三 <removeBtn>     |
| · ops (运维)  ←     |   ☑ 查看                   | 李四 <removeBtn>     |
| · viewer            |                            |                      |
| · 自建-门卫         | ▾ 终端                     |                      |
|                     |   ☑ 查看                   |                      |
| [编辑] [删除]       |                            |                      |
|                     | [保存权限]                 |                      |
+---------------------+----------------------------+----------------------+
```

**关键交互**：

- 左栏：角色列表、新建/编辑/删除（删除按钮在 `is_system=true` 时禁用）
- 中栏：树形权限勾选，**第一期只渲染 `kind=page` 节点**；`admin` 角色选中时显示"系统管理员，拥有所有权限"占位，不渲染树
- 右栏：复用 `/contacts` 页的飞书用户搜索能力（姓名/邮箱），点击添加后调 `POST /roles/:id/users`

### 6.2 `/users` 页（用户授权）

- 表格新增"角色"列，多角色以 chips 展示
- 行操作"授权"按钮 → 弹窗多选角色 → `PUT /users/:id/roles`（覆盖式）

### 6.3 全局 `useMe()` Hook

```ts
// apps/web/lib/use-me.ts
// 拉 /me/permissions 缓存到 SWR；提供 hasPerm(code) 给组件用
export function useMe(): { permissions: Set<string>; hasPerm: (code: string) => boolean; isAdmin: boolean }
```

- `SideNav` 用 `hasPerm("endpoint:read")` 决定是否渲染对应入口；`isAdmin` 直接放行所有
- 按钮级（第二期）调 `hasPerm("endpoint:delete")`
- 后端 403 时前端 toast 提示 + 跳转 `/forbidden`

---

## 7. 落地步骤

按依赖顺序执行：

1. **migrations 000012–000015**：4 张新表
2. **migration 000016**：`users.roles` 加 deprecated 注释
3. **`pkg/perm/codes.go`**：权限码权威列表 + 启动 UPSERT 同步入库
4. **`cmd/seed` 扩展**：创建 `admin/viewer` 内置角色 → `daipengyuan` 绑 admin
5. **`pg_repo/role.go` + `permission.go` + `user_role.go`**：接口 + 实现
6. **`service/role.go` + `service/permission.go`**：业务逻辑（含 `UserHasAny`）
7. **`middleware/perm.go`**：`RequirePermission` 实现（含 admin 特判）
8. **`handler/role.go` + `handler/permission.go`**：路由处理函数
9. **`handler/handler.go`**：注册新路由 + **给所有现有路由挂 `RequirePermission`**
10. **`handler/user.go`** 加 `PUT /users/:id/roles`
11. **前端 `/roles` 页**：三栏组件
12. **前端 `/users` 页**：角色列 + 授权弹窗
13. **`useMe()` + `SideNav` 按权限显隐**

---

## 8. 关键设计决策

| # | 决策 | 为什么 |
|---|---|---|
| 1 | 独立 `user_roles` 表，不用 `users.roles JSONB` | 需"角色→用户反查"（管理页右栏）和审计（granted_at/by）；JSONB 反查需 `@>`，索引不友好 |
| 2 | 权限码代码维护，UI 只读 | 系统能力可枚举且与代码强绑定，让 UI 创建会出现"勾了但代码没用"的死权限 |
| 3 | `kind=page/action` 二分 + `parent_code` 父子链 | 极简的"为细粒度铺垫"——不需要 OPA/Casbin，未来扩展只需多注册 action 码、给路由多传 `RequirePermission` 参数 |
| 4 | admin 中间件特判（B 方案） | 避免新增权限码时同步维护 admin 的绑定关系；0 数据漂移风险 |
| 5 | JWT 不携带权限决策信息 | 角色/权限可变；JWT 缓存权限会导致撤权延迟至 token 过期 |
| 6 | 白名单制路由 | 所有接口必须显式挂 `RequirePermission`，避免漏挂裸奔 |
| 7 | 第一期不引入权限缓存 | 用户量级 ~10，单查询亚毫秒，过度优化 |
| 8 | 弃用 `users.roles` 而非立即删除 | 过渡期兼容，避免和正在运行的 JWT 字段冲突 |

---

## 9. 待办与未决

- [ ] 补齐其他模块权限码：`cloud_network` / `dumb_terminal` / `network_device`
- [ ] 第二期是否需要"操作审计日志"（who/when/what）单独建表
- [ ] 数据范围过滤（ABAC）需求出现时，评估接入 Casbin 还是 service 层硬编码
- [ ] 老 `Claims.Roles` 字段下次 JWT 协议升级时移除
