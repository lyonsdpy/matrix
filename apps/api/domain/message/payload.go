package message

import (
	"matrix/api/domain/aisacg"
	"matrix/api/domain/aisdesk"
	"matrix/api/domain/lark"
)

// OrgSyncPayload 组织架构同步载荷。
type OrgSyncPayload struct {
	Departments []lark.Department `json:"departments"`
	Users       []lark.User       `json:"users"`
}

// DeviceSyncPayload 设备同步载荷。
type DeviceSyncPayload struct {
	Devices []lark.Device `json:"devices"`
}

// OnlineStatusPayload 在线用户快照载荷。
type OnlineStatusPayload struct {
	Office string              `json:"office"` // 办公室名称（对应 ACG 设备 Name）
	Users  []aisacg.UserOnline `json:"users"`  // 该设备的在线用户列表
}

// UserEventPayload 用户增量事件载荷。
type UserEventPayload struct {
	User lark.User `json:"user"`
}

// DeptEventPayload 部门增量事件载荷。
type DeptEventPayload struct {
	Department lark.Department `json:"department"`
}

// DeviceChangedPayload 设备变更事件载荷。
type DeviceChangedPayload struct {
	Device lark.Device `json:"device"`
}

// BotChatEnteredPayload 用户进入机器人会话事件载荷。
type BotChatEnteredPayload struct {
	UserID string `json:"user_id"` // 飞书 user_id
	ChatID string `json:"chat_id"` // 会话 ID
}

// BotMessagePayload 用户发送消息给机器人的载荷。
type BotMessagePayload struct {
	Message lark.Message `json:"message"`
}

// CardActionPayload 卡片按钮点击事件载荷。
type CardActionPayload struct {
	Action lark.CardAction `json:"action"`
}

// BotMenuPayload 飞书机器人菜单点击事件载荷。
type BotMenuPayload struct {
	Event lark.BotMenuEvent `json:"event"`
}

// DesktopDeviceSyncPayload 桌面管理设备全量同步载荷。
type DesktopDeviceSyncPayload struct {
	Devices []aisdesk.Device `json:"devices"`
}

// DesktopSoftwareSyncPayload 单台终端已安装软件同步载荷。
type DesktopSoftwareSyncPayload struct {
	DeviceID   int                         `json:"device_id"`
	DeviceName string                      `json:"device_name"`
	Softwares  []aisdesk.InstalledSoftware `json:"softwares"`
}

// DesktopHardwareSyncPayload 单台终端硬件组件同步载荷。
type DesktopHardwareSyncPayload struct {
	DeviceID   int                         `json:"device_id"`
	Components []aisdesk.HardwareComponent `json:"components"`
}

// ACGWhitelistSyncPayload ACG 设备全局白名单同步载荷。
type ACGWhitelistSyncPayload struct {
	ACGDevice string                  `json:"acg_device"`
	Entries   []aisacg.WhitelistEntry `json:"entries"`
}

// ACGUserSyncPayload ACG 设备用户同步载荷。
type ACGUserSyncPayload struct {
	ACGDevice string        `json:"acg_device"`
	Users     []aisacg.User `json:"users"`
}
