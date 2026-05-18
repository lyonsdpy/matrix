package aisacg

import (
	"time"

	"github.com/google/uuid"
)

// BlacklistItem 软件黑名单规则。
type BlacklistItem struct {
	ID        uuid.UUID // 主键
	Name      string    // 黑名单软件名称（用于子串匹配）
	CreatedAt time.Time // 创建时间
}

// BlacklistHit 软件黑名单命中记录。
type BlacklistHit struct {
	ID              uuid.UUID // 主键
	EmployeeID      string    // 员工 ID（关联失败时为空）
	EmployeeName    string    // 员工姓名
	DeviceID        string    // 设备 ID
	DeviceName      string    // 设备名称
	SoftwareName    string    // 命中的软件名称
	SoftwareVersion string    // 软件版本
	DetectedAt      time.Time // 检测时间
}
