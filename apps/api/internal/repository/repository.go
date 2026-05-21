package repository

import (
	"matrix/api/internal/repository/neo4j_repo"
	"matrix/api/internal/repository/pg_repo"

	"github.com/jmoiron/sqlx"
)

// GraphRepos 图数据库中的 domain 数据。
// 生产环境使用 neo4j_repo.Neo4jDeviceRepo，当前使用内存存根。
type GraphRepos struct {
	Device *neo4j_repo.DeviceRepo
	User   *neo4j_repo.UserGraphRepo
}

// SyncRepos 从外部系统同步过来、存储在 PostgreSQL 的数据。
// 跨源关联（graph ↔ sync）在 Service 层手工完成。
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
}

// Repositories 聚合所有数据访问实例，由 main 初始化后注入 Service 层。
type Repositories struct {
	Graph *GraphRepos
	Sync  *SyncRepos
	Hello HelloRepository
	Task  TaskRepository
}

func New(db *sqlx.DB) *Repositories {
	return &Repositories{
		Graph: &GraphRepos{
			Device: neo4j_repo.NewDeviceRepo(),
			User:   neo4j_repo.NewUserGraphRepo(),
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
		},
		Hello: newHelloRepo(),
		Task:  newTaskRepo(),
	}
}
