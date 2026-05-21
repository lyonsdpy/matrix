package domain

import "time"

// AuditAction 审计操作类型枚举。
type AuditAction string

const (
	AuditCreate  AuditAction = "create"
	AuditUpdate  AuditAction = "update"
	AuditDelete  AuditAction = "delete"  // 软删除
	AuditRestore AuditAction = "restore" // 从软删除恢复
)

// TemporalMeta 时态元数据，嵌入节点和边结构体。
//
// 时间维度说明：
//   - CreatedAt / UpdatedAt / DeletedAt：系统时间（transaction time），记录数据库层面的写入时刻
//   - ValidFrom / ValidTo：业务时间（valid time），建模现实世界中该事实的有效区间；
//     两者均为 nil 时表示无明确业务时效限制（永久有效）
//
// 软删除：DeletedAt 非 nil 时表示已删除，查询默认过滤此类记录，
// 但时态查询（AsOf）仍可访问已删除的历史状态。
type TemporalMeta struct {
	Version   int        `json:"version"` // 单调递增，每次变更 +1；初始值为 1
	CreateBy  string     `json:"create_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdateBy  string     `json:"update_by"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"` // nil 表示未删除
	ValidFrom *time.Time `json:"valid_from,omitempty"` // 业务有效开始时间
	ValidTo   *time.Time `json:"valid_to,omitempty"`   // 业务有效结束时间，nil 表示无限期
}

// IsDeleted 返回该记录是否处于软删除状态。
func (m *TemporalMeta) IsDeleted() bool { return m.DeletedAt != nil }

// IsValidAt 检查在给定时间 t，该记录的业务有效期是否覆盖 t。
// 仅考虑 ValidFrom/ValidTo，不考虑软删除状态。
func (m *TemporalMeta) IsValidAt(t time.Time) bool {
	if m.ValidFrom != nil && t.Before(*m.ValidFrom) {
		return false
	}
	if m.ValidTo != nil && !t.Before(*m.ValidTo) {
		return false
	}
	return true
}

// NodeVersion 节点的一次历史版本快照。
//
// 写入策略：每次更新节点属性前，将旧版本属性整体写入此结构，
// 并将上一版本的 ValidTo 设为本次变更时刻，形成不重叠的版本区间链。
// 当前版本的 ValidTo 为 nil。
type NodeVersion struct {
	NodeID    string         `json:"node_id"`
	NodeLabel string         `json:"node_label"` // Neo4j label，例如 "Device"
	Version   int            `json:"version"`
	ValidFrom time.Time      `json:"valid_from"` // 该版本生效的系统时间
	ValidTo   *time.Time     `json:"valid_to"`   // nil 表示此版本仍为当前版本
	Props     map[string]any `json:"props"`      // 该版本的完整属性快照（不含时态字段）
	EventID   string         `json:"event_id"`   // 产生该版本的审计事件 ID
}

// EdgeVersion 边的历史版本快照，与 NodeVersion 对称设计。
//
// 边的 ID 在 Neo4j 中使用 elementId；FromID/ToID 记录端点，
// 以便在端点节点被删除后仍能还原历史图结构。
type EdgeVersion struct {
	EdgeID    string         `json:"edge_id"`
	EdgeType  string         `json:"edge_type"` // 关系类型，例如 "CONNECTED_TO"
	FromID    string         `json:"from_id"`
	ToID      string         `json:"to_id"`
	Version   int            `json:"version"`
	ValidFrom time.Time      `json:"valid_from"`
	ValidTo   *time.Time     `json:"valid_to"`
	Props     map[string]any `json:"props"`
	EventID   string         `json:"event_id"`
}

// AuditEvent 图变更的不可变审计记录。
//
// 约束：写入后不允许修改或删除（append-only）。
// 推荐落库到 PostgreSQL 的 audit_events 表，而非 Neo4j，
// 便于跨实体的时序查询与合规审计导出。
type AuditEvent struct {
	ID         string         `json:"id"`
	Action     AuditAction    `json:"action"`
	EntityType string         `json:"entity_type"` // "node" 或 "edge"
	EntityID   string         `json:"entity_id"`
	EntityKind string         `json:"entity_kind"`      // 例如 "Device"、"CONNECTED_TO"
	Before     map[string]any `json:"before,omitempty"` // 变更前快照；create 时为 nil
	After      map[string]any `json:"after,omitempty"`  // 变更后快照；delete 时记录删除时刻
	OperatorID string         `json:"operator_id"`      // 操作人（飞书 open_id 或系统账号）
	OccurredAt time.Time      `json:"occurred_at"`
	Reason     string         `json:"reason,omitempty"` // 变更原因，由调用方填写
}
