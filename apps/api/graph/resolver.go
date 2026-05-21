package graph

// 此文件由开发者手写，gqlgen generate 只在文件不存在时创建，不会覆盖。

// ── Resolver 的定位 ───────────────────────────────────────────────────────────
//
// Resolver 是 GraphQL 层的依赖根，所有 xxxResolver 都嵌入它来共享依赖。
//
// 持有 Service 而不是 Repository 的原因：
//   Resolver 是 GraphQL 协议层，不应该直接碰数据库。
//   它只负责：解包 GraphQL 参数 → 调 Service → 把结果转成 GraphQL 类型。
//   业务规则统一在 Service 层扩展，Resolver 不需要改动。
//
// DataLoader 不在此处持有——它是 per-request 的，通过 context 传递（见 handler.go）。

import "matrix/api/internal/service"

type Resolver struct {
	deviceSvc *service.DeviceService
}

func NewResolver(deviceSvc *service.DeviceService) *Resolver {
	return &Resolver{deviceSvc: deviceSvc}
}
