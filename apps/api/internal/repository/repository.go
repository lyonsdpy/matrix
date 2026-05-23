package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"matrix/api/domain"
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
}

type GroupGraphRepo interface {
	GetGroup(ctx context.Context, id string) (*domain.Group, error)
	ListGroups(_ context.Context, first int, _ string) ([]*domain.Group, bool, string, error)
	GreateGroup(_ context.Context, name string) (*domain.Group, error)
	GetUserGroups(_ context.Context, userID string) ([]*domain.UserGroupLink, error)
	GetGroupMembers(_ context.Context, groupID string) ([]*domain.UserGroupLink, error)
	GetGroupChildren(_ context.Context, parentID string) ([]*domain.GroupGroupLink, error)
	AddUserToGroup(_ context.Context, userID, groupID string) error
	AddGroupToGroup(_ context.Context, parentID, childID string) error
}

// GraphRepos 图数据库中的 domain 数据。
type GraphRepos struct {
	Device DeviceGraphRepo
	User   *neo4j_repo.UserGraphRepo
	Group  GroupGraphRepo
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

	userRepo := neo4j_repo.NewUserGraphRepo()
	return &Repositories{
		Graph: &GraphRepos{
			Device: deviceRepo,
			User:   userRepo,
			Group:  neo4j_repo.NewGroupGraphRepo(userRepo),
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
