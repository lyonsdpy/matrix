package lark

import "context"

// ContactEventHandler 通讯录事件处理器。
// 由 service 层实现，注入到 infra 层的事件分发器中。
type ContactEventHandler interface {
	OnUserCreated(ctx context.Context, user *User) error
	OnUserDeleted(ctx context.Context, user *User) error
	OnUserUpdated(ctx context.Context, user *User) error
	OnDeptCreated(ctx context.Context, dept *Department) error
	OnDeptDeleted(ctx context.Context, dept *Department) error
	OnDeptUpdated(ctx context.Context, dept *Department) error
}

// DeviceEventHandler 设备变更事件处理器。
type DeviceEventHandler interface {
	OnDeviceChanged(ctx context.Context, device *Device) error
}

// MessageEventHandler 消息事件处理器。
type MessageEventHandler interface {
	OnMessageReceived(ctx context.Context, msg *Message) error
	OnBotChatEntered(ctx context.Context, userID string, chatID string) error
}

// CardEventHandler 卡片交互事件处理器。
type CardEventHandler interface {
	OnCardAction(ctx context.Context, action *CardAction) error
}

// MenuEventHandler 机器人菜单点击事件处理器。
type MenuEventHandler interface {
	OnBotMenuClicked(ctx context.Context, event *BotMenuEvent) error
}
