package domain

import (
	"context"
	"time"
)

// NodeVersionRepository 节点版本快照的存储接口。
// 实现可选择写入 Neo4j（作为 :NodeVersion 节点）或 PostgreSQL。
type NodeVersionRepository interface {
	// SaveNodeVersion 在节点变更时记录旧版本快照。
	SaveNodeVersion(ctx context.Context, v *NodeVersion) error

	// ListNodeVersions 返回指定节点的所有历史版本，按 version asc 排序。
	ListNodeVersions(ctx context.Context, nodeID string) ([]*NodeVersion, error)

	// GetNodeAt 返回节点在给定系统时间点的快照；若该时刻节点尚不存在则返回 nil。
	GetNodeAt(ctx context.Context, nodeID string, at time.Time) (*NodeVersion, error)
}

// EdgeVersionRepository 边版本快照的存储接口，与 NodeVersionRepository 对称。
type EdgeVersionRepository interface {
	SaveEdgeVersion(ctx context.Context, v *EdgeVersion) error
	ListEdgeVersions(ctx context.Context, edgeID string) ([]*EdgeVersion, error)
	GetEdgeAt(ctx context.Context, edgeID string, at time.Time) (*EdgeVersion, error)
}

// AuditEventRepository 审计事件的存储接口（append-only）。
type AuditEventRepository interface {
	// Append 写入一条审计事件，实现层需保证原子性且禁止修改已写入记录。
	Append(ctx context.Context, e *AuditEvent) error

	// ListByEntity 按实体 ID 查询该实体上发生的所有审计事件，时间正序。
	ListByEntity(ctx context.Context, entityID string) ([]*AuditEvent, error)

	// ListByOperator 查询某操作人在 [from, to] 时间段内的所有操作。
	ListByOperator(ctx context.Context, operatorID string, from, to time.Time) ([]*AuditEvent, error)
}
