/*
可接入网络的终端设备，包含：电脑、打印机
*/

package domain

type Endpoint struct {
	ID               string         `json:"id"`
	FeishuDeviceID   string         `json:"feishu_device_id"`   // 飞书 device_record_id，业务唯一键
	Name             string         `json:"name"`               // 设备名称
	Type             EndpointType   `json:"type"`               // 设备类型
	Status           EndpointStatus `json:"status"`             // 设备状态
	AssetNumber      string         `json:"asset_number"`       // 资产编码
	SerialNumber     string         `json:"serial_number"`      // 生产序列号
	DiskSerialNumber string         `json:"disk_serial_number"` // 硬盘序列号
	Vendor           string         `json:"vendor"`             // 制造商
	Model            string         `json:"model"`              // 设备型号
	OSName           string         `json:"os_name"`            // 操作系统
	TemporalMeta                    // 时态元数据（版本、时间戳、软删除）
}

type EndpointType string

const (
	EndpointTypePC         EndpointType = "PC"         // 台式机
	EndpointTypeLaptop     EndpointType = "LAPTOP"     // 便携笔记本
	EndpointTypePrinter    EndpointType = "PRINTER"    // 打印机
	EndpointTypeTV         EndpointType = "TV"         // 智能电视
	EndpointTypeAttendance EndpointType = "ATTENDANCE" // 考勤机
	EdnpointTypePhone      EndpointType = "PHONE"      // 手机
	EndpointTypeTablet     EndpointType = "TABLET"     // 平板
	EndpointTypeOther      EndpointType = "OTHER"
)

type EndpointStatus string

const (
	EndpointStatusActive   EndpointStatus = "ACTIVE"
	EndpointStatusInactive EndpointStatus = "INACTIVE"
	EndpointStatusLost     EndpointStatus = "LOST"
	EndpointStatusRetired  EndpointStatus = "RETIRED"
)

// EndpointOS 终端操作系统枚举。
// 飞书 device_terminal_type: 1=Win/2=Mac/3=Linux/4=iOS/5=Android/6=HarmonyOS。
type EndpointOS string

const (
	EndpointOSWindows   EndpointOS = "WINDOWS"
	EndpointOSMacOS     EndpointOS = "MACOS"
	EndpointOSLinux     EndpointOS = "LINUX"
	EndpointOSIOS       EndpointOS = "IOS"
	EndpointOSAndroid   EndpointOS = "ANDROID"
	EndpointOSHarmonyOS EndpointOS = "HARMONYOS"
	EndpointOSOther     EndpointOS = "OTHER"
)

// SyncedEndpoint 用户终端管理列表/详情用：在 Endpoint 基础上挂当前/最近登录的 User。
// 字段命名贴合飞书 device_record/list 返回的语义，前端按需展示。
// CurrentUser/LatestUser 可能为空——飞书 device 引用但 User 节点尚未同步、或飞书未上报该字段。
type SyncedEndpoint struct {
	ID             string         `json:"id"`
	FeishuDeviceID string         `json:"feishu_device_id"`
	Name           string         `json:"name"`
	Type           EndpointType   `json:"type"` // 物理形态（PC/PHONE/...）
	OS             EndpointOS     `json:"os"`   // 操作系统（WINDOWS/MACOS/...）
	Status         EndpointStatus `json:"status"`
	PlatformCode   string         `json:"platform_code"` // 飞书原始 terminal_type 编号（1~6）

	// 硬件标识
	SerialNumber     string `json:"serial_number"`
	DiskSerialNumber string `json:"disk_serial_number"`
	BoardUUID        string `json:"board_uuid"`
	MACAddress       string `json:"mac_address"`
	Model            string `json:"model"`

	// 操作系统 / 版本
	OSCode  string `json:"os_code"` // device_system 原始编号
	Version string `json:"version"`

	// 合规与归属
	Ownership     string `json:"ownership"`     // device_ownership 原始编号
	TrustLevel    string `json:"trust_level"`   // device_status 原始编号
	Certification string `json:"certification"` // certification_level 原始编号

	// MDM
	IsManaged   bool   `json:"is_managed"`
	MDMDeviceID string `json:"mdm_device_id"`
	MDMProvider string `json:"mdm_provider"`

	// 关联用户
	CurrentUser *SyncedUser `json:"current_user,omitempty"`
	LatestUser  *SyncedUser `json:"latest_user,omitempty"`
}
