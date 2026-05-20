package desktop

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// DeviceFetcher 亚信桌面管理设备数据读取能力。
// 所有方法均为只读操作，仅返回安装了 Agent 的设备（TDevice.AgentInstalled=1）。
type DeviceFetcher interface {
	// ListDevices 返回所有安装了 Agent 的设备基础信息（TDevice + TComputer）。
	ListDevices(ctx context.Context) ([]Device, error)

	// ListInstalledSoftware 返回指定设备的已安装软件列表（AgentInstalled=1 过滤由 ListDevices 保证，此处按 deviceID 查询）。
	ListInstalledSoftware(ctx context.Context, deviceID int) ([]InstalledSoftware, error)

	// ListHardwareComponents 返回指定设备的硬件组件详情列表（AgentInstalled=1 过滤由 ListDevices 保证，此处按 deviceID 查询）。
	ListHardwareComponents(ctx context.Context, deviceID int) ([]HardwareComponent, error)
}

// 编译期断言：deviceFetcher 实现了 DeviceFetcher 接口。
var _ DeviceFetcher = (*deviceFetcher)(nil)

// deviceFetcher 使用 sqlx 从桌面管理 MySQL 数据库读取设备信息。
type deviceFetcher struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewDeviceFetcher 创建 DeviceFetcher 实现，持有只读数据库连接和结构化日志器。
func NewDeviceFetcher(db *sqlx.DB, logger *zap.Logger) DeviceFetcher {
	return &deviceFetcher{db: db, logger: logger}
}

// deviceRow 是 ListDevices SQL 查询结果的内部映射行。
// TDevice 的 DevName/ComputerName/Mac/UserName/LastTime 允许 NULL（Risk R2），
// 字符串字段在 SQL 层 COALESCE 为 ”，LastTime 用 sql.NullTime 处理。
type deviceRow struct {
	DeviceID     int          `db:"DeviceID"`
	IP           string       `db:"IP"`
	AllIP        string       `db:"AllIP"`
	AllMAC       string       `db:"AllMAC"`
	DevName      string       `db:"DevName"`
	ComputerName string       `db:"ComputerName"`
	MAC          string       `db:"Mac"`
	DeviceType   int          `db:"Type"`
	Online       bool         `db:"Online"`
	UserName     string       `db:"UserName"`
	LastTime     sql.NullTime `db:"LastTime"`
	OSName       string       `db:"OSName"`
	OSVersion    string       `db:"OSVersion"`
	CPU          string       `db:"CPU"`
	MemoryMB     int          `db:"MemoryMB"`
	DiskSizeGB   int          `db:"DiskSizeGB"`
	AgentVersion string       `db:"AgentVersion"`
	UUID         string       `db:"UUID"`
}

// softwareRow 是 ListInstalledSoftware SQL 查询结果的内部映射行。
type softwareRow struct {
	DeviceID    int    `db:"DeviceID"`
	DisplayName string `db:"DisplayName"`
	Version     string `db:"Version"`
	Vendor      string `db:"Vendor"`
	InstallDate string `db:"InstallDate"`
	SoftSize    string `db:"SoftSize"`
}

// hardwareRow 是 ListHardwareComponents SQL 查询结果的内部映射行。
type hardwareRow struct {
	DeviceID int    `db:"DeviceID"`
	Name     string `db:"Name"`
	Type     string `db:"Type"`
	Prop     string `db:"Prop"`
	Vendor   string `db:"Vendor"`
}

const listDevicesSQL = `
SELECT
    d.DeviceID,
    d.IP,
    COALESCE(d.DevName,      '') AS DevName,
    COALESCE(d.ComputerName, '') AS ComputerName,
    COALESCE(d.Mac,          '') AS Mac,
    d.Type,
    d.Online,
    COALESCE(d.UserName,     '') AS UserName,
    d.LastTime,
    COALESCE(c.AllIP,    '')  AS AllIP,
    COALESCE(c.AllMac,   '')  AS AllMAC,
    COALESCE(c.OSName,   '')  AS OSName,
    COALESCE(c.OSVersion,'')  AS OSVersion,
    COALESCE(c.CPU,      '')  AS CPU,
    COALESCE(c.Memory,   0)   AS MemoryMB,
    COALESCE(c.DiskSize, 0)   AS DiskSizeGB,
    COALESCE(c.AgtVer,   '')  AS AgentVersion,
    COALESCE(c.UUID,     '')  AS UUID
FROM TDevice d
LEFT JOIN TComputer c ON c.DeviceID = d.DeviceID
WHERE d.AgentInstalled = 1
ORDER BY d.DeviceID`

const listInstalledSoftwareSQL = `
SELECT
    DeviceID,
    DisplayName,
    SoftVersion                                     AS Version,
    COALESCE(SoftVendor, '')                        AS Vendor,
    COALESCE(DATE_FORMAT(InstallDate, '%Y-%m-%d'),'') AS InstallDate,
    COALESCE(SoftSize, '')                          AS SoftSize
FROM TSoftware_AgentInstalled
WHERE DeviceID = ?
ORDER BY DisplayName`

const listHardwareComponentsSQL = `
SELECT
    hr.DeviceID,
    hi.Name,
    hi.Type,
    COALESCE(hi.Prop,   '') AS Prop,
    COALESCE(hi.Vendor, '') AS Vendor
FROM THardRelation hr
JOIN TDevHardInfo hi ON hi.ID = hr.HardID
WHERE hr.DeviceID = ?
ORDER BY hi.Type, hi.Name`

// ListDevices 返回所有安装了 Agent 的设备基础信息（TDevice + TComputer LEFT JOIN）。
func (f *deviceFetcher) ListDevices(ctx context.Context) ([]Device, error) {
	var rows []deviceRow
	if err := f.db.SelectContext(ctx, &rows, listDevicesSQL); err != nil {
		return nil, fmt.Errorf("desktop: ListDevices query: %w", err)
	}

	devices := make([]Device, 0, len(rows))
	for _, r := range rows {
		devices = append(devices, toDevice(r))
	}
	return devices, nil
}

// ListInstalledSoftware 返回指定设备的已安装软件列表。
func (f *deviceFetcher) ListInstalledSoftware(ctx context.Context, deviceID int) ([]InstalledSoftware, error) {
	var rows []softwareRow
	if err := f.db.SelectContext(ctx, &rows, listInstalledSoftwareSQL, deviceID); err != nil {
		return nil, fmt.Errorf("desktop: ListInstalledSoftware(deviceID=%d) query: %w", deviceID,
			err)
	}

	result := make([]InstalledSoftware, 0, len(rows))
	for _, r := range rows {
		result = append(result, toInstalledSoftware(r))
	}
	return result, nil
}

// ListHardwareComponents 返回指定设备的硬件组件详情列表。
func (f *deviceFetcher) ListHardwareComponents(ctx context.Context, deviceID int) ([]HardwareComponent, error) {
	var rows []hardwareRow
	if err := f.db.SelectContext(ctx, &rows, listHardwareComponentsSQL, deviceID); err != nil {
		return nil, fmt.Errorf("desktop: ListHardwareComponents(deviceID=%d) query: %w", deviceID,
			err)
	}

	result := make([]HardwareComponent, 0, len(rows))
	for _, r := range rows {
		result = append(result, toHardwareComponent(r))
	}
	return result, nil
}

// toDevice 将 deviceRow 转换为领域模型 Device。
// LastTime 为 sql.NullTime，NULL 时映射为 time.Time 零值。
func toDevice(r deviceRow) Device {
	var lastTime time.Time
	if r.LastTime.Valid {
		lastTime = r.LastTime.Time
	}
	return Device{
		DeviceID:     r.DeviceID,
		IP:           r.IP,
		AllIP:        r.AllIP,
		AllMAC:       r.AllMAC,
		DevName:      r.DevName,
		ComputerName: r.ComputerName,
		MAC:          r.MAC,
		DeviceType:   r.DeviceType,
		Online:       r.Online,
		UserName:     r.UserName,
		LastTime:     lastTime,
		OSName:       r.OSName,
		OSVersion:    r.OSVersion,
		CPU:          r.CPU,
		MemoryMB:     r.MemoryMB,
		DiskSizeGB:   r.DiskSizeGB,
		AgentVersion: r.AgentVersion,
		UUID:         r.UUID,
	}
}

// toInstalledSoftware 将 softwareRow 转换为领域模型 InstalledSoftware。
func toInstalledSoftware(r softwareRow) InstalledSoftware {
	return InstalledSoftware{
		DeviceID:    r.DeviceID,
		DisplayName: r.DisplayName,
		Version:     r.Version,
		Vendor:      r.Vendor,
		InstallDate: r.InstallDate,
		SoftSize:    r.SoftSize,
	}
}

// toHardwareComponent 将 hardwareRow 转换为领域模型 HardwareComponent。
func toHardwareComponent(r hardwareRow) HardwareComponent {
	return HardwareComponent{
		DeviceID: r.DeviceID,
		Name:     r.Name,
		Type:     r.Type,
		Prop:     r.Prop,
		Vendor:   r.Vendor,
	}
}
