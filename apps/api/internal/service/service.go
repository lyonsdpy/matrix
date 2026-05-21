package service

import "matrix/api/internal/repository"

// Services 聚合所有业务逻辑实例。
//
// 新增业务模块的步骤：
//  1. 新建 internal/service/xxx.go，定义所需的 repo 接口 + XxxService 结构体
//  2. 在此处加 Xxx *XxxService 字段
//  3. 在 New() 里加 Xxx: NewXxxService(...)，注入 repos.Graph 或 repos.Sync 中的对应接口
type Services struct {
	Hello    HelloService
	Task     TaskService
	Device   *DeviceService
	Employee *EmployeeService
}

// New 初始化所有 Service，由 main 调用一次，注入到 Handler 层。
// 依赖方向：main → Handler → Service → Repository，单向，不允许反向依赖。
func New(repos *repository.Repositories) *Services {
	return &Services{
		Hello:  &helloSvc{repo: repos.Hello},
		Task:   newTaskSvc(repos.Task),
		Device: NewDeviceService(repos.Graph.Device),
		// Employee 演示跨源聚合：PG 同步层 + 图层各提供一个接口
		Employee: NewEmployeeService(repos.Sync.Employee, repos.Graph.User),
	}
}
