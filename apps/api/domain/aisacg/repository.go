package aisacg

import (
	"context"

	"github.com/google/uuid"
)

// WhitelistRepository ACG 全局白名单持久化仓储。
type WhitelistRepository interface {
	// ReplaceByDevice 原子替换指定 ACG 设备的白名单条目（全量覆盖）。
	ReplaceByDevice(ctx context.Context, acgDevice string, entries []WhitelistEntry) error

	// FindAll 获取所有 ACG 设备的白名单条目。
	FindAll(ctx context.Context) ([]WhitelistEntry, error)
}

// ViolationRepository 违规记录持久化接口。
type ViolationRepository interface {
	// Save 保存违规记录。
	Save(ctx context.Context, record *ViolationRecord) error

	// SaveBatch 批量保存违规记录。
	SaveBatch(ctx context.Context, records []ViolationRecord) error

	// FindByOffice 按办公室查询违规记录。
	FindByOffice(ctx context.Context, office string) ([]ViolationRecord, error)

	// FindAll 获取所有违规记录。
	FindAll(ctx context.Context) ([]ViolationRecord, error)
}

// SoftwareBlacklistRepository 软件黑名单规则持久化接口。
type SoftwareBlacklistRepository interface {
	// FindAll 获取所有黑名单规则，按创建时间排序。
	FindAll(ctx context.Context) ([]BlacklistItem, error)

	// Save 保存黑名单规则（重复名称忽略）。
	Save(ctx context.Context, item *BlacklistItem) error

	// Delete 按 ID 删除黑名单规则。
	Delete(ctx context.Context, id uuid.UUID) error
}

// BlacklistHitRepository 软件黑名单命中记录持久化接口。
type BlacklistHitRepository interface {
	// SaveBatch 批量保存命中记录。
	SaveBatch(ctx context.Context, hits []BlacklistHit) error

	// FindByEmployeeID 按员工 ID 查询命中记录。
	FindByEmployeeID(ctx context.Context, employeeID string) ([]BlacklistHit, error)

	// FindAll 获取所有命中记录。
	FindAll(ctx context.Context) ([]BlacklistHit, error)
}
