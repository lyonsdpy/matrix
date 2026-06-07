// Package perm 定义系统权限码权威列表。
//
// 设计原则（参见 docs/authz/README.md）：
//   - 权限码由代码维护，UI 只读。系统能力可枚举且与代码强绑定，
//     避免 UI 创建出"勾了但代码没用"的死权限。
//   - kind=page 控制页面入口（第一期使用）；kind=action 控制按钮/操作（预留）。
//     扩展细粒度时只需在此追加 action 码、给路由加 RequirePermission，
//     表结构、前后端框架均无需改动。
//   - parent_code 构建权限码层级，UI 树形展示及"勾父全选子"操作的基础。
package perm

// Kind 区分页面级和动作级权限。
type Kind string

const (
	KindPage   Kind = "page"
	KindAction Kind = "action"
)

// Permission 权限码定义。
type Permission struct {
	Code   string // 唯一权限码，格式 <module>:<scope> 或 <module>:<sub>:<action>
	Name   string // 显示名（中文）
	Module string // 模块分组键
	Kind   Kind   // page | action
	Parent string // 父权限码，空字符串表示顶级
	Sort   int    // 同级排序键
}

// Codes 是系统权限码权威列表。
// 启动时通过 SyncCatalog 同步至 permissions 表，UI 只读这张表。
//
// 新增权限码：
//  1. 在此处追加条目（page 节点 Parent 为空，action 节点 Parent 指向所属 page 码）
//  2. 给对应路由加 middleware.RequirePermission("xxx:yyy")
//  3. 重启服务，启动同步自动写入 permissions 表
var Codes = []Permission{
	// ── 通讯录（飞书用户/部门） ──────────────────────────────────────────
	{Code: "contact:read", Name: "通讯录-查看", Module: "contact", Kind: KindPage, Sort: 10},
	{Code: "contact:sync", Name: "通讯录-触发同步", Module: "contact", Kind: KindAction, Parent: "contact:read", Sort: 11},

	// ── 终端管理 ────────────────────────────────────────────────────────
	{Code: "endpoint:read", Name: "终端管理-查看", Module: "endpoint", Kind: KindPage, Sort: 20},
	{Code: "endpoint:write", Name: "终端-编辑", Module: "endpoint", Kind: KindAction, Parent: "endpoint:read", Sort: 21},
	{Code: "endpoint:delete", Name: "终端-删除", Module: "endpoint", Kind: KindAction, Parent: "endpoint:read", Sort: 22},
	{Code: "endpoint:sync", Name: "终端-触发同步", Module: "endpoint", Kind: KindAction, Parent: "endpoint:read", Sort: 23},

	// ── 网络设备（Neo4j Device 图） ─────────────────────────────────────
	{Code: "network_device:read", Name: "网络设备-查看", Module: "network_device", Kind: KindPage, Sort: 30},
	{Code: "network_device:write", Name: "网络设备-编辑", Module: "network_device", Kind: KindAction, Parent: "network_device:read", Sort: 31},
	{Code: "network_device:delete", Name: "网络设备-删除", Module: "network_device", Kind: KindAction, Parent: "network_device:read", Sort: 32},

	// ── 哑终端 ──────────────────────────────────────────────────────────
	{Code: "dumb_terminal:read", Name: "哑终端-查看", Module: "dumb_terminal", Kind: KindPage, Sort: 40},

	// ── 云网络 ──────────────────────────────────────────────────────────
	{Code: "cloud_network:read", Name: "云网络-查看", Module: "cloud_network", Kind: KindPage, Sort: 50},

	// ── 用户授权（系统账号管理，不是飞书通讯录） ────────────────────────
	{Code: "user:read", Name: "用户授权-查看", Module: "user", Kind: KindPage, Sort: 60},
	{Code: "user:role:assign", Name: "用户授权-分配角色", Module: "user", Kind: KindAction, Parent: "user:read", Sort: 61},

	// ── 角色管理 ────────────────────────────────────────────────────────
	{Code: "role:read", Name: "角色管理-查看", Module: "role", Kind: KindPage, Sort: 70},
	{Code: "role:write", Name: "角色-创建编辑", Module: "role", Kind: KindAction, Parent: "role:read", Sort: 71},
	{Code: "role:delete", Name: "角色-删除", Module: "role", Kind: KindAction, Parent: "role:read", Sort: 72},
	{Code: "role:assign", Name: "角色-绑定权限/用户", Module: "role", Kind: KindAction, Parent: "role:read", Sort: 73},
}

// RoleCodeAdmin 系统内置管理员角色码。
// 中间件特判此角色绕过所有权限校验（B 方案，见 docs/authz/README.md §5.4）。
const RoleCodeAdmin = "admin"

// RoleCodeViewer 系统内置只读角色码。
const RoleCodeViewer = "viewer"
