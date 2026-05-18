package service

import "matrix/api/internal/repository"

// Services 聚合所有业务逻辑实例。
// 新增业务模块时，在此添加对应的 Service 字段并在 New() 中初始化。
type Services struct {
	Hello HelloService
	Task  TaskService
}

// New 初始化所有 Service，由 main 调用，注入到 Handler 层
func New(repos *repository.Repositories) *Services {
	return &Services{
		Hello: &helloSvc{repo: repos.Hello},
		Task:  newTaskSvc(repos.Task),
	}
}
