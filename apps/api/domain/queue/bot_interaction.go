// Package queue 定义 bot 交互事件队列的领域模型与持久化接口。
// sink 消费 Kafka 后将机器人交互事件写入队列（Enqueue），
// lark-bot 通过轮询队列（DequeueBatch）取出事件并调用飞书 API 处理。
package queue

import (
	"context"
	"time"
)

// 事件类型常量，对应 Kafka 消息中的 message.Type。
const (
	// BotEventChatEntered 用户进入机器人会话。
	BotEventChatEntered = "bot_chat_entered"
	// BotEventMessage 用户发送消息给机器人。
	BotEventMessage = "bot_message"
	// BotEventCardAction 用户点击卡片按钮。
	BotEventCardAction = "card_action"
	// BotEventMenu 用户点击机器人菜单。
	BotEventMenu = "bot_menu"
)

// 状态常量，描述 BotInteraction 的生命周期。
const (
	// StatusPending sink 写入，等待 lark-bot 处理。
	StatusPending = "pending"
	// StatusProcessing lark-bot 已取出，正在处理中（SKIP LOCKED 保证并发安全）。
	StatusProcessing = "processing"
	// StatusDone 处理完成，飞书消息已发送。
	StatusDone = "done"
	// StatusFailed 处理失败，error_msg 记录原因，可人工介入重试。
	StatusFailed = "failed"
)

// BotInteraction 机器人交互队列记录，对应数据库 bot_interactions 表。
// 生命周期：pending → processing → done | failed。
type BotInteraction struct {
	ID          int64
	EventType   string     // BotEventChatEntered / BotEventMessage / BotEventCardAction
	UserID      string     // 飞书 user_id（事件发起用户）
	Payload     []byte     // JSON 序列化的原始 Kafka Payload struct
	Status      string     // StatusPending / StatusProcessing / StatusDone / StatusFailed
	ErrorMsg    string     // 失败时记录的错误信息
	CreatedAt   time.Time  // 记录写入时间
	ProcessedAt *time.Time // 处理完成时间（nil 表示尚未处理）
}

// BotInteractionRepository bot 交互队列持久化接口。
// 实现位于 infra/postgres/bot_interaction_repo.go。
type BotInteractionRepository interface {
	// Enqueue 写入一条 pending 状态的交互记录。
	Enqueue(ctx context.Context, interaction BotInteraction) error

	// DequeueBatch 原子获取最多 limit 条 pending 记录并更新为 processing。
	// 内部使用事务 + SELECT FOR UPDATE SKIP LOCKED，保证并发安全（不重复消费）。
	// 返回空切片表示当前无待处理记录。
	DequeueBatch(ctx context.Context, limit int) ([]BotInteraction, error)

	// MarkDone 将指定记录更新为 done，设置 processed_at 为当前时间。
	MarkDone(ctx context.Context, id int64) error

	// MarkFailed 将指定记录更新为 failed，记录错误信息。
	MarkFailed(ctx context.Context, id int64, errMsg string) error
}
