package service

import (
	"time"

	"matrix/api/internal/infra/lark"
	"matrix/api/internal/repository"
	"matrix/api/pkg/config"
	"matrix/api/pkg/log"
)

// Services 聚合所有业务逻辑实例。
//
// 新增业务模块的步骤：
//  1. 新建 internal/service/xxx.go，定义所需的 repo 接口 + XxxService 结构体
//  2. 在此处加 Xxx *XxxService 字段
//  3. 在 New() 里加 Xxx: NewXxxService(...)，注入 repos.Graph 或 repos.Sync 中的对应接口
type Services struct {
	Hello        HelloService
	Task         TaskService
	Device       *DeviceService
	Employee     *EmployeeService
	Auth         *AuthService
	User         *UserService
	Department   *DepartmentService
	Contact      *ContactService
	Endpoint     *EndpointService
	Sync         *SyncService
	EndpointSync *EndpointSyncService
}

// Shutdown 停止所有后台 goroutine，应在服务器 Stop 之前调用。
func (s *Services) Shutdown() {
	s.Task.Stop()
}

// New 初始化所有 Service，由 main 调用一次，注入到 Handler 层。
// 依赖方向：main → Handler → Service → Repository，单向，不允许反向依赖。
func New(repos *repository.Repositories, jwtCfg config.JWT, larkCfg config.Lark) *Services {
	expiry := time.Duration(jwtCfg.ExpiryHours) * time.Hour
	if expiry <= 0 {
		expiry = 24 * time.Hour
	}

	// lark 客户端构造一次，OAuth / ContactFetcher / DeviceFetcher 共用，避免 token 重复刷新
	var (
		larkOAuth      *lark.OAuthProvider
		contactFetcher lark.ContactFetcher
		deviceFetcher  lark.DeviceFetcher
	)
	if larkCfg.AppID != "" {
		larkClient, err := lark.NewClient(lark.Config{
			AppID:     larkCfg.AppID,
			AppSecret: larkCfg.AppSecret,
		})
		if err == nil {
			larkOAuth = lark.NewOAuthProvider(larkClient, larkCfg.AppID, larkCfg.OAuthRedirect)
			contactFetcher = lark.NewContactFetcher(larkClient, log.Log)
			deviceFetcher = lark.NewDeviceFetcher(larkClient, log.Log)
		}
	}

	return &Services{
		Hello:      &helloSvc{repo: repos.Hello},
		Task:       newTaskSvc(repos.Task),
		Device:     NewDeviceService(repos.Graph.Device),
		Employee:   NewEmployeeService(repos.Sync.Employee, repos.Graph.User),
		Auth:       NewAuthService(repos.Sync.AuthUser, larkOAuth, jwtCfg.Secret, expiry),
		User:       NewUserService(repos.Graph.User),
		Department: NewDepartmentService(repos.Graph.Department),
		Contact:    NewContactService(repos.Graph.User, repos.Graph.Department),
		Endpoint:   NewEndpointService(repos.Graph.Endpoint),
		Sync: NewSyncService(
			contactFetcher,
			repos.Graph.User,
			repos.Graph.Department,
		),
		EndpointSync: NewEndpointSyncService(deviceFetcher, repos.Graph.Endpoint),
	}
}
