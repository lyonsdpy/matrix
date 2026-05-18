package kafka

import (
	"encoding/json"
	"time"
)

type Type string

const (
	TypeOrgSync      Type = "lark.org_sync"        // 飞书组织架构全量同步
	TypeDeviceSync   Type = "lark.device_sync"     // 飞书设备信息同步
	TypeOnlineStatus Type = "aisacg.online_status" // 亚信在线用户快照

	// 通讯录增量事件
	TypeUserCreated Type = "lark.user.created"
	TypeUserUpdated Type = "lark.user.updated"
	TypeUserDeleted Type = "lark.user.deleted"
	TypeDeptCreated Type = "lark.dept.created"
	TypeDeptUpdated Type = "lark.dept.updated"
	TypeDeptDeleted Type = "lark.dept.deleted"

	// 设备增量事件
	TypeDeviceChanged Type = "lark.device.changed"

	// 机器人交互事件
	TypeBotChatEntered Type = "lark.bot.chat_entered"
	TypeBotMessage     Type = "lark.bot.message"
	TypeCardAction     Type = "lark.card.action"
	TypeBotMenu        Type = "lark.bot.menu" // 机器人菜单点击事件

	// 桌面管理同步事件
	TypeDesktopDeviceSync   Type = "desktop.device_sync"
	TypeDesktopSoftwareSync Type = "desktop.software_sync"
	TypeDesktopHardwareSync Type = "desktop.hardware_sync"

	// ACG 配置同步事件
	TypeACGWhitelistSync Type = "aisacg.whitelist_sync"
	TypeACGUserSync      Type = "aisacg.user_sync"
)

// Message Kafka 统一消息信封。
// 所有写入 Kafka 的消息都使用此结构序列化为 JSON。
type Message struct {
	Type      Type            `json:"type"`      // 消息类型
	Source    string          `json:"source"`    // 来源标识（如 "collector"）
	Timestamp time.Time       `json:"timestamp"` // 消息产生时间
	TraceID   string          `json:"trace_id"`  // 追踪 ID（UUID）
	Payload   json.RawMessage `json:"payload"`   // 业务载荷（延迟解析）
}
