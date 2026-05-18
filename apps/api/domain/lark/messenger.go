package lark

import "context"

// Messenger 消息发送能力。
type Messenger interface {
	// SendText 发送文本消息给指定用户（通过 user_id）。
	SendText(ctx context.Context, userID string, text string) (string, error)

	// SendCard 发送卡片消息给指定用户（通过 user_id）。
	// cardJSON 为飞书卡片 JSON 字符串。
	SendCard(ctx context.Context, userID string, cardJSON string) (string, error)

	// ReplyText 回复文本消息。
	ReplyText(ctx context.Context, messageID string, text string) (string, error)

	// ReplyCard 回复卡片消息。
	ReplyCard(ctx context.Context, messageID string, cardJSON string) (string, error)
}
