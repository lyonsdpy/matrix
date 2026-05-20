package lark

// EventType 飞书事件类型。
type EventType string

const (
	// 通讯录事件
	EventUserCreated EventType = "contact.user.created_v3"
	EventUserDeleted EventType = "contact.user.deleted_v3"
	EventUserUpdated EventType = "contact.user.updated_v3"
	EventDeptCreated EventType = "contact.department.created_v3"
	EventDeptDeleted EventType = "contact.department.deleted_v3"
	EventDeptUpdated EventType = "contact.department.updated_v3"

	// 设备事件
	EventDeviceChanged EventType = "security_and_compliance.device_record.device_change_event"

	// 消息事件
	EventMessageReceive EventType = "im.message.receive_v1"
	EventBotChatEntered EventType = "p2p_chat_create" // 用户进入与机器人的会话

	// EventCardAction 卡片事件
	EventCardAction EventType = "card.action.trigger"

	// EventBotMenu 飞书机器人菜单点击事件类型（application.bot.menu_v6）。
	EventBotMenu EventType = "application.bot.menu_v6"
)

// ContactEvent 通讯录变更事件（用户/部门增删改）。
type ContactEvent struct {
	Type       EventType   // 事件类型
	User       *User       // 用户事件时非 nil
	Department *Department // 部门事件时非 nil
}

// DeviceEvent 设备变更事件。
type DeviceEvent struct {
	Type   EventType // EventDeviceChanged
	Device *Device   // 变更后的设备信息
}

// MessageEvent 消息接收事件。
type MessageEvent struct {
	Type    EventType // EventMessageReceive
	Message *Message  // 接收到的消息
}

// BotChatEnteredEvent 用户进入机器人会话事件。
type BotChatEnteredEvent struct {
	Type   EventType // EventBotChatEntered
	UserID string    // 进入会话的用户 user_id
	ChatID string    // 会话 ID
}

// BotMenuEvent 机器人菜单点击事件。
type BotMenuEvent struct {
	Type     EventType // EventBotMenu
	EventKey string    // 菜单项唯一标识，如 "compliance_check"、"device_status"
	UserID   string    // 操作用户 user_id（租户内唯一）
	OpenID   string    // 操作用户 open_id（可为空）
}
