// 已同步的飞书用户，对应后端 domain.SyncedUser。
export interface SyncedUser {
  id: string;
  name: string;
  feishu_id: string; // open_id
  user_id: string;
  email: string;
  status: number;
  department_ids: string[];
  department_names: string[]; // 通过 MEMBER_OF 边取到的部门中文名
}

// 用户列表查询响应，对应后端 service.SyncedUserList。
export interface SyncedUserList {
  users: SyncedUser[];
  has_next: boolean;
  end_cursor: string;
}

// 通讯录-部门最小引用(用于路径/面包屑)
export interface DepartmentRef {
  department_id: string;
  name: string;
}

// 通讯录-部门树节点(列表/树共用)，has_children 用于树懒加载判断
export interface DepartmentNode {
  department_id: string;
  name: string;
  parent_id: string;
  member_count: number;
  has_children: boolean;
}

// 通讯录-部门详情面板用
export interface DepartmentDetail {
  department_id: string;
  name: string;
  parent_id: string;
  leader_user_id: string;
  member_count: number;
  recursive_member_count: number;
  path: DepartmentRef[];
  children: DepartmentNode[];
  direct_members: SyncedUser[];
}

// 通讯录-用户的某个部门归属，带从顶级到该部门的完整路径
export interface UserDepartment {
  department_id: string;
  name: string;
  path: DepartmentRef[];
}

// 通讯录-用户详情面板用
export interface UserDetailData extends SyncedUser {
  departments: UserDepartment[];
  leaders: SyncedUser[]; // 上级领导：所属部门沿父链向上首个非自己的 leader
  managed_departments: DepartmentRef[]; // 管理部门：本人作为 leader 的部门
}

// 通讯录-联合搜索结果(按类型分组，前端按类型路由到对应卡片组件)
export interface ContactSearchResult {
  users: SyncedUser[];
  departments: DepartmentNode[];
}

// EntityRef 详情抽屉接受的实体引用，type 决定渲染哪个详情组件(registry 模式扩展点)
export type EntityType = "user" | "department" | "endpoint";
export interface EntityRef {
  type: EntityType;
  id: string; // user: feishu_id; department: department_id; endpoint: node id (SHA1 派生)
}

// 终端管理-飞书同步过来的设备（含当前/最近登录用户），对应后端 domain.SyncedEndpoint。
// 飞书原始字段：https://open.feishu.cn/document/security_and_compliance-v1/security_and_compliance-v2/device_record/list
export interface SyncedEndpoint {
  id: string;
  feishu_device_id: string;
  name: string;
  type: string;   // 物理形态：PC | LAPTOP | PRINTER | TV | ATTENDANCE | PHONE | TABLET | OTHER
  os: string;     // 操作系统：WINDOWS | MACOS | LINUX | IOS | ANDROID | HARMONYOS | OTHER
  status: string; // ACTIVE | INACTIVE | LOST | RETIRED
  platform_code: string; // 飞书原始 terminal_type（1=Win/2=Mac/3=Linux/4=iOS/5=Android/6=HarmonyOS）

  // 硬件标识
  serial_number: string;
  disk_serial_number: string;
  board_uuid: string;
  mac_address: string;
  model: string;

  // 操作系统/版本
  os_code: string; // device_system 原始编号
  version: string;

  // 合规与归属
  ownership: string;     // device_ownership 原始编号
  trust_level: string;   // device_status 原始编号
  certification: string; // certification_level 原始编号

  // MDM
  is_managed: boolean;
  mdm_device_id: string;
  mdm_provider: string;

  // 关联用户
  current_user?: SyncedUser;
  latest_user?: SyncedUser;
}

// 终端列表响应，对应后端 service.EndpointList。
// page 从 1 起；total 为命中总数（用于分页器渲染"共 N 条 / 共 M 页"和跳页）。
export interface EndpointList {
  endpoints: SyncedEndpoint[];
  total: number;
  page: number;
  page_size: number;
}

// ─── RBAC 角色权限 ────────────────────────────────────────────────

// 权限码视图，对应后端 service.PermissionView
export interface Permission {
  code: string;
  name: string;
  module: string;
  kind: "page" | "action";
  parent_code: string; // 空字符串表示顶级
  sort: number;
}

// 角色视图（列表/详情共用），对应后端 service.RoleView
export interface Role {
  id: string;
  code: string;
  name: string;
  description: string;
  is_system: boolean;
  user_count: number;
  created_at: string;
  updated_at: string;
}

// 角色详情，对应后端 service.RoleDetail
export interface RoleDetail extends Role {
  permission_codes: string[]; // admin 角色为空（走中间件特判）
}

// 角色下用户视图
export interface RoleUser {
  user_id: string;
  username: string;
  lark_open_id: string;
  granted_at: string;
}

// 当前登录者的权限码集合（给前端按钮/菜单显隐用）
export interface MePermissions {
  user_id: string;
  username: string;
  is_admin: boolean;
  permissions: string[];
}
