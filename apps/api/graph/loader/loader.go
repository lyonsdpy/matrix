package loader

import (
	"context"
	"matrix/api/domain"

	"github.com/vikstrous/dataloadgen"
)

// ctxKey 是注入 context 的私有键类型，防止和其他包的 key 碰撞。
type ctxKey struct{}

// Key 供 handler 中间件注入 Loaders 到 request context 时使用。
var Key = ctxKey{}

// For 从 context 中取出当前请求的 Loaders。
// 必须在 dataLoaderMiddleware 之后调用，否则会 panic。
func For(ctx context.Context) *Loaders {
	return ctx.Value(ctxKey{}).(*Loaders)
}

// Loaders 聚合当前进程所有模块的 DataLoader。
//
// DataLoader 解决的问题：GraphQL 查询 N 个设备时，每个设备都触发一次 IP 查询，
// 就会产生 N 次数据库调用（N+1 问题）。DataLoader 把同一请求窗口内的多次查询
// 自动合并成一次批量查询。
//
// 新增模块的步骤：
//  1. 新建 graph/loader/xxx.go，定义 XxxBatchProvider 接口 + Key 类型 + 私有 loader 工厂函数
//  2. 在此处加对应字段
//  3. 在 NewLoaders 参数里加 xxx XxxBatchProvider，并在函数体里初始化
type Loaders struct {
	// Device 模块
	IpsByDeviceID      *dataloadgen.Loader[DeviceIPsKey, []*domain.IPv4Addr]
	DevLinksByDeviceID *dataloadgen.Loader[DeviceLinksKey, []*domain.DeviceLink]
}

// NewLoaders 初始化所有 DataLoader，由 dataLoaderMiddleware 在每个请求开始时调用。
//
// 为什么每个请求都要新建？DataLoader 的批量窗口和缓存是请求级别隔离的，
// 如果跨请求复用，A 请求的缓存会污染 B 请求的结果。
//
// 新增模块时在参数里加对应的 BatchProvider：
//
//	func NewLoaders(device DeviceBatchProvider, user UserBatchProvider) *Loaders
func NewLoaders(device DeviceBatchProvider) *Loaders {
	return &Loaders{
		IpsByDeviceID:      newDeviceIPsLoader(device),
		DevLinksByDeviceID: newDeviceLinksLoader(device),
	}
}
