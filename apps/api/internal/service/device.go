package service

import (
	"context"
	"fmt"
	"matrix/api/domain"
)

// ── 为什么要有 Service 层？ ──────────────────────────────────────────────────
//
// 没有 Service 层时，resolver 直接调 repo，短期没问题。
// 但随着需求增长，"创建设备时发 Kafka 消息"、"删除前检查是否有关联"这类
// 业务规则没地方放，最终全堆进 resolver，resolver 变成大杂烩。
//
// Service 层的职责边界：
//   Resolver  → 只做 GraphQL 参数解包 + 类型转换，不含业务判断
//   Service   → 业务规则、跨模块协调、错误语义
//   Repository → 只管数据怎么存取，不知道任何业务规则

// DeviceRepository ── 接口定义在使用方，是 Go 的惯用法 ─────────────────────────────────────────
//
// DeviceRepository 定义在 service 包，不在 repository 包。
// 这遵循 Go "accept interfaces, return structs" 原则：
//   - service 声明自己需要什么能力，而不依赖某个具体实现
//   - 将来把 DeviceRepo（内存）换成 Neo4jRepo，只要新实现满足这个接口即可
//   - 测试时可以传入 mock，不需要真实数据库
type DeviceRepository interface {
	GetDevice(ctx context.Context, id string) (*domain.Device, error)
	ListDevices(ctx context.Context, first int, after string) ([]*domain.Device, bool, string, error)
	CreateDevice(ctx context.Context, name, deviceType, mip string) (*domain.Device, error)
	DeleteDevice(ctx context.Context, id string) (bool, error)
	BatchConnectionsByDeviceIDs(ctx context.Context, ids []string, limit int) (map[string][]*domain.DeviceLink, error)
}

// DeviceService 是设备模块的业务逻辑层，持有 Repository 接口而非具体实现。
type DeviceService struct {
	repo DeviceRepository
}

func NewDeviceService(repo DeviceRepository) *DeviceService {
	return &DeviceService{repo: repo}
}

// Get 获取单个设备。
// 错误在此处包装语义："device.Get" 让调用栈里的错误信息更清晰。
func (s *DeviceService) Get(ctx context.Context, id string) (*domain.Device, error) {
	d, err := s.repo.GetDevice(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("device.Get %s: %w", id, err)
	}
	return d, nil
}

// List 返回分页设备列表，业务上封装了"把 (nodes, hasNext, cursor) 组装成 DeviceConnection"。
//
// 为什么这个组装在 Service 而不是 Resolver？
// DeviceConnection 是 domain 类型（分页容器），组装它是业务行为；
// 而 Resolver 只负责把 GraphQL 的 *int32 参数转成 Go 的 int，不做业务组装。
func (s *DeviceService) List(ctx context.Context, first int, after string) (*domain.DeviceConnection, error) {
	nodes, hasNext, endCursor, err := s.repo.ListDevices(ctx, first, after)
	if err != nil {
		return nil, fmt.Errorf("device.List: %w", err)
	}
	conn := &domain.DeviceConnection{Nodes: nodes, HasNextPage: hasNext}
	if endCursor != "" {
		conn.EndCursor = &endCursor
	}
	return conn, nil
}

// Create 创建设备。当前逻辑简单，但 Service 层是扩展点：
// 未来在 repo.CreateDevice 前后可插入：权限校验、重复检测、Kafka 消息、审计日志等，
// 而不需要改 Resolver 或 Repository。
func (s *DeviceService) Create(ctx context.Context, name, deviceType, mip string) (*domain.Device, error) {
	return s.repo.CreateDevice(ctx, name, deviceType, mip)
}

// Topology 用 BFS 遍历设备连接图，最多走 depth 跳。
//
// 为什么 BFS 在 Service 而不是 Resolver？
// "图遍历走几跳"是业务规则，和 GraphQL 协议无关。
// Resolver 不应该关心遍历算法，它只负责把结果转换成 GraphQL 类型。
//
// 为什么返回 ([]*domain.Device, []*domain.DeviceLink) 而不是 *model.GraphResult？
// model.GraphResult 是 gqlgen 生成的 GraphQL 专属类型，Service 层不应该依赖 GraphQL。
// 类型转换（domain → GraphQL model）是 Resolver 的工作。
//
// 为什么用 BatchConnectionsByDeviceIDs 而不是逐个查询？
// BFS 每一层可能有多个节点，批量查一次 vs 每节点查一次，是 N+1 vs 1 的差距。
func (s *DeviceService) Topology(ctx context.Context, id string, depth int) ([]*domain.Device, []*domain.DeviceLink, error) {
	startDev, err := s.repo.GetDevice(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("device.Topology: %w", err)
	}

	visited := map[string]bool{id: true}
	queue := []string{id}
	allDevices := []*domain.Device{startDev}
	var allLinks []*domain.DeviceLink

	for level := 0; level < depth && len(queue) > 0; level++ {
		// 批量拉取当前层所有节点的出向连接，避免 N+1
		linkMap, err := s.repo.BatchConnectionsByDeviceIDs(ctx, queue, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("device.Topology level %d: %w", level, err)
		}
		var next []string
		for _, devID := range queue {
			for _, link := range linkMap[devID] {
				allLinks = append(allLinks, link)
				if !visited[link.Target.ID] {
					visited[link.Target.ID] = true
					next = append(next, link.Target.ID)
					allDevices = append(allDevices, link.Target)
				}
			}
		}
		queue = next
	}
	return allDevices, allLinks, nil
}
