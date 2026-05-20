package repository

// Repositories 聚合所有数据访问实例。
// 新增模块时在此添加字段并在 New() 中初始化。
type Repositories struct {
	Hello  HelloRepository
	Task   TaskRepository
	Device *DeviceRepo
}

// New 初始化所有 Repository，由 main 调用
func New() *Repositories {
	return &Repositories{
		Hello:  newHelloRepo(),
		Task:   newTaskRepo(),
		Device: newDeviceRepo(),
	}
}
