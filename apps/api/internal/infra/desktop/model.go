// Package desktop 定义亚信桌面管理（AIS Desktop）的领域模型和数据读取接口。
// 本包为纯 Go 包，零外部依赖（仅使用标准库 time），可由任意上层服务直接引用。
package desktop

import "time"

// 设备类型常量，对应 TDeviceType 表中的 TypeID。
const (
	DeviceTypeWindowsPC = 101
	DeviceTypeAndroid   = 103
	DeviceTypeMac       = 110
)

// Device 桌面管理中安装了 Agent 的终端设备。
// 字段来自 TDevice（基础信息）+ TComputer（系统与硬件摘要）的 LEFT JOIN。
// UserName / DepartID 来自桌面管理，不可信，以飞书数据为准。
type Device struct {
	DeviceID     int       // TDevice.DeviceID，桌面管理侧主键
	IP           string    // 当前主 IP（TDevice.IP）
	AllIP        string    // 所有 IP，逗号分隔（TComputer.AllIP）
	AllMAC       string    // 所有 MAC，逗号分隔（TComputer.AllMac）
	DevName      string    // 桌面管理设备名（TDevice.DevName）
	ComputerName string    // 计算机名（含域/WORKGROUP 前缀，TDevice.ComputerName）
	MAC          string    // 主 MAC（TDevice.Mac）
	DeviceType   int       // 设备类型（101=WinPC / 103=Android / 110=Mac）
	Online       bool      // 是否在线（TDevice.Online=1）
	UserName     string    // 桌面管理登录用户名（不可信，以飞书为准）
	LastTime     time.Time // 最后上报时间（TDevice.LastTime）
	OSName       string    // 操作系统名称（TComputer.OSName）
	OSVersion    string    // OS 版本号（TComputer.OSVersion）
	CPU          string    // CPU 型号（TComputer.CPU）
	MemoryMB     int       // 内存 MB（TComputer.Memory）
	DiskSizeGB   int       // 磁盘 GB（TComputer.DiskSize）
	AgentVersion string    // Agent 版本（TComputer.AgtVer）
	UUID         string    // 系统 UUID（TComputer.UUID，可用于与飞书设备比对）
}

// HardwareComponent 终端设备的单个硬件组件。
type HardwareComponent struct {
	DeviceID int    // 所属桌面管理设备 ID
	Name     string // 组件描述，如 "物理内存 Ramaxel Technology 16384 MB 5600 MHz"
	Type     string // 组件类型（CPU/Memory/Disk/Video/Network/BaseBoard/BIOS 等）
	Prop     string // 额外属性（如容量、频率），可为空
	Vendor   string // 厂商，可为空
}

// InstalledSoftware 终端设备上已安装的软件。
type InstalledSoftware struct {
	DeviceID    int    // 所属桌面管理设备 ID
	DisplayName string // 软件显示名称
	Version     string // 版本号
	Vendor      string // 发布商
	InstallDate string // 安装日期（格式 "2006-01-02"，NULL 时为空字符串）
	SoftSize    string // 安装大小，可为空
}
