package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"matrix/api/domain"
	"matrix/api/internal/infra/lark"
	"matrix/api/internal/repository/neo4j_repo"
	"matrix/api/internal/repository/pg_repo"
)

// DeviceGraphRepo 是设备图节点的完整访问接口。
// 同时覆盖 service.DeviceRepository 和 loader.DeviceBatchProvider 所需的方法，
// 这样 GraphRepos.Device 既能注入 Service，又能注入 DataLoader，无需两个字段。
type DeviceGraphRepo interface {
	GetDevice(ctx context.Context, id string) (*domain.Device, error)
	ListDevices(ctx context.Context, first int, after string) ([]*domain.Device, bool, string, error)
	CreateDevice(ctx context.Context, name, deviceType, mip string) (*domain.Device, error)
	UpdateDevice(ctx context.Context, id string, name, deviceType, mip *string) (*domain.Device, error)
	DeleteDevice(ctx context.Context, id string) (bool, error)
	BatchConnectionsByDeviceIDs(ctx context.Context, ids []string, limit int) (map[string][]*domain.DeviceLink, error)
	BatchIPsByDeviceIDs(ctx context.Context, ids []string, limit int) (map[string][]*domain.IPv4Addr, error)
	CreateConnection(ctx context.Context, fromID, toID string) (*domain.DeviceLink, error)
}

type UserGraphRepo interface {
	FindByFeishuID(_ context.Context, feishuID string) (*domain.User, error)
	GetUser(ctx context.Context, id string) (*domain.User, error)
	ListUsers(ctx context.Context, first int, after string) ([]*domain.User, bool, string, error)
	CreateUser(_ context.Context, name, feishuID string) (*domain.User, error)
	// SearchSyncedUsers 按 name/email 模糊搜索已同步用户，游标分页（cursor 为 feishu_id）。
	// 返回完整飞书字段，供用户管理列表展示。
	SearchSyncedUsers(ctx context.Context, search, cursor string, limit int) ([]*domain.SyncedUser, bool, string, error)
	// GetSyncedUserDetail 按 open_id 取用户详情：基本字段 + 所属部门(带完整路径)。
	GetSyncedUserDetail(ctx context.Context, openID string) (*domain.UserDetail, error)
	// GetLeaders 上级领导：所属部门沿 PARENT_OF 链向上，跳过自己当 leader 的部门后取首个非自己的 leader，去重。
	GetLeaders(ctx context.Context, openID string, limit int) ([]*domain.SyncedUser, error)
	// GetManagedDepartments 管理部门：leader_user_id == 本人 user_id 的部门列表。
	GetManagedDepartments(ctx context.Context, openID string) ([]*domain.DepartmentRef, error)
	// SearchUsersByKeyword name/email 模糊匹配，供联合搜索使用。
	SearchUsersByKeyword(ctx context.Context, q string, limit int) ([]*domain.SyncedUser, error)

	// ── sync 同步用：批量写、全 ID 列表、批量删 ──
	BulkUpsertSyncedUsers(ctx context.Context, batch []lark.User) error
	ListAllFeishuIDs(ctx context.Context) (map[string]struct{}, error)
	DeleteByFeishuIDs(ctx context.Context, ids []string) (int, error)
}

// DepartmentGraphRepo Department 节点的图能力：写入 + 树/路径/递归人数/直属成员等图特性查询。
type DepartmentGraphRepo interface {
	Upsert(ctx context.Context, d lark.Department) error
	LinkParent(ctx context.Context, deptID, parentID string) error
	LinkUser(ctx context.Context, userFeishuID string, deptIDs []string) error

	ListChildren(ctx context.Context, parentID string) ([]*domain.DepartmentNode, error)
	GetDetail(ctx context.Context, deptID string, memberLimit int) (*domain.DepartmentDetail, error)
	GetPath(ctx context.Context, deptID string) ([]*domain.DepartmentRef, error)
	RecursiveMemberCount(ctx context.Context, deptID string) (int, error)
	ListDirectMembers(ctx context.Context, deptID string, limit int) ([]*domain.SyncedUser, error)
	SearchByName(ctx context.Context, q string, limit int) ([]*domain.DepartmentNode, error)

	// ── sync 同步用：批量 ──
	BulkUpsert(ctx context.Context, batch []lark.Department) error
	BulkLinkParents(ctx context.Context, batch []lark.Department) error
	BulkLinkMembers(ctx context.Context, batch []lark.User) error
	ListAllIDs(ctx context.Context) (map[string]struct{}, error)
	DeleteByIDs(ctx context.Context, ids []string) (int, error)
}

// EndpointGraphRepo 终端节点图能力。
// Endpoint 与 User 通过两种边关联：
//   - (Endpoint)-[:CURRENT_LOGIN]->(User)  当前登录用户
//   - (Endpoint)-[:LATEST_LOGIN]->(User)   最近一次登录用户
// 语义都是"登录"而非"归属"；归属由后续资产管理系统维护。
type EndpointGraphRepo interface {
	BulkUpsert(ctx context.Context, batch []lark.Device) error
	BulkLinkLogin(ctx context.Context, batch []lark.Device) error
	ListAllFeishuDeviceIDs(ctx context.Context) (map[string]struct{}, error)
	DeleteByFeishuDeviceIDs(ctx context.Context, ids []string) (int, error)

	// Search 终端列表：q=设备名/序列号模糊；userQ=按关联用户(current 或 latest)姓名/邮箱筛选；
	// typeFilter/osFilter=按物理形态/操作系统精确匹配（空字符串视为不筛）；cursor=feishu_device_id。
	Search(ctx context.Context, q, userQ, typeFilter, osFilter, cursor string, limit int) ([]*domain.SyncedEndpoint, bool, string, error)
	// GetDetail 按节点 id 取终端 + 关联用户。
	GetDetail(ctx context.Context, id string) (*domain.SyncedEndpoint, error)
}

type GroupGraphRepo interface {
	GetGroup(ctx context.Context, id string) (*domain.Group, error)
	ListGroups(_ context.Context, first int, _ string) ([]*domain.Group, bool, string, error)
	CreateGroup(_ context.Context, name string) (*domain.Group, error)
	GetUserGroups(_ context.Context, userID string) ([]*domain.UserGroupLink, error)
	GetGroupMembers(_ context.Context, groupID string) ([]*domain.UserGroupLink, error)
	GetGroupChildren(_ context.Context, parentID string) ([]*domain.GroupGroupLink, error)
	AddUserToGroup(_ context.Context, userID, groupID string) error
	AddGroupToGroup(_ context.Context, parentID, childID string) error
}

// GraphRepos 图数据库中的 domain 数据。
type GraphRepos struct {
	Device     DeviceGraphRepo
	User       UserGraphRepo
	Group      GroupGraphRepo
	Department DepartmentGraphRepo
	Endpoint   EndpointGraphRepo
}

// SyncRepos 从外部系统同步过来、存储在 PostgreSQL 的数据，以及本地 auth 账号体系。
type SyncRepos struct {
	Employee     pg_repo.EmployeeRepository
	Department   pg_repo.DepartmentRepository
	Device       pg_repo.EmployeeDeviceRepository
	Session      pg_repo.OnlineSessionRepository
	UserBind     pg_repo.UserBindItemRepository
	MACBinding   pg_repo.ObservedMACBindingRepository
	Violation    pg_repo.ViolationRepository
	Blacklist    pg_repo.SoftwareBlacklistRepository
	BlacklistHit pg_repo.BlacklistHitRepository
	AuthUser     pg_repo.AuthUserRepository
}

// Repositories 聚合所有数据访问实例，由 main 初始化后注入 Service 层。
type Repositories struct {
	Graph *GraphRepos
	Sync  *SyncRepos
	Hello HelloRepository
	Task  TaskRepository
}

// New 初始化所有 Repository。
// neo4jDriver 非 nil 时使用真实 Neo4j；为 nil 时降级为内存存根（本地开发无需启动 Neo4j）。
func New(db *sqlx.DB, neo4jDriver neo4j.Driver, neo4jDB string) *Repositories {
	var deviceRepo DeviceGraphRepo
	if neo4jDriver != nil {
		deviceRepo = neo4j_repo.NewNeo4jDeviceRepo(neo4jDriver, neo4jDB)
	} else {
		deviceRepo = neo4j_repo.NewDeviceRepo()
	}

	// Group 仍是内存存根（暂未接 Neo4j），依赖内存版 UserGraphRepo，维持现状不动
	memUserRepo := neo4j_repo.NewUserGraphRepo()
	// GraphRepos.User 走真实 Neo4j：employee.Get 跨源 join 与扫码登录关联都依赖它；无 driver 时降级
	var userRepo UserGraphRepo = memUserRepo
	if neo4jDriver != nil {
		userRepo = neo4j_repo.NewNeo4jUserGraphRepo(neo4jDriver, neo4jDB)
	}
	// Department 走真实 Neo4j：通讯录树/路径/递归人数全靠它；无 driver 时降级为 stub
	var deptRepo DepartmentGraphRepo = neo4j_repo.NewStubDepartmentRepo()
	if neo4jDriver != nil {
		deptRepo = neo4j_repo.NewNeo4jDepartmentRepo(neo4jDriver, neo4jDB)
	}
	// Endpoint：飞书设备同步专用；无 driver 时降级为 stub
	var endpointRepo EndpointGraphRepo = neo4j_repo.NewStubEndpointRepo()
	if neo4jDriver != nil {
		endpointRepo = neo4j_repo.NewNeo4jEndpointRepo(neo4jDriver, neo4jDB)
	}
	return &Repositories{
		Graph: &GraphRepos{
			Device:     deviceRepo,
			User:       userRepo,
			Group:      neo4j_repo.NewGroupGraphRepo(memUserRepo),
			Department: deptRepo,
			Endpoint:   endpointRepo,
		},
		Sync: &SyncRepos{
			Employee:     pg_repo.NewEmployeeRepository(db),
			Department:   pg_repo.NewDepartmentRepository(db),
			Device:       pg_repo.NewDeviceRepository(db),
			Session:      pg_repo.NewOnlineSessionRepository(db),
			UserBind:     pg_repo.NewUserBindItemRepository(db),
			MACBinding:   pg_repo.NewObservedMACBindingRepository(db),
			Violation:    pg_repo.NewViolationRepository(db),
			Blacklist:    pg_repo.NewBlacklistItemRepository(db),
			BlacklistHit: pg_repo.NewBlacklistHitRepository(db),
			AuthUser:     pg_repo.NewAuthUserRepository(db),
		},
		Hello: newHelloRepo(),
		Task:  newTaskRepo(),
	}
}
