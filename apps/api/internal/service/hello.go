package service

import "matrix/api/internal/repository"

// HelloService 打招呼业务逻辑接口
type HelloService interface {
	// SayHello 根据用户名生成问候语，封装业务规则（如名称校验、国际化等）
	SayHello(name string) string
}

// helloSvc 是 HelloService 的具体实现，持有 Repository 引用
type helloSvc struct {
	repo repository.HelloRepository // 依赖数据访问层，而非具体实现（便于测试 mock）
}

// SayHello 业务层调用 Repository 获取数据，此处可叠加业务规则
func (s *helloSvc) SayHello(name string) string {
	// 此处可添加：参数校验、权限判断、日志埋点、缓存读取等业务逻辑
	return s.repo.GetGreeting(name)
}
