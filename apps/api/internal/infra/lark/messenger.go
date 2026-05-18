// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"context"
	"encoding/json"
	"fmt"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"go.uber.org/zap"

	domain "matrix/api/domain/lark"
)

// messenger 实现 domain.Messenger。
type messenger struct {
	client *Client
	logger *zap.Logger
}

// compile-time interface check
var _ domain.Messenger = (*messenger)(nil)

// NewMessenger 创建消息发送器。
func NewMessenger(client *Client, logger *zap.Logger) domain.Messenger {
	return &messenger{client: client, logger: logger}
}

// SendText 发送文本消息给指定用户（通过 user_id）。
func (m *messenger) SendText(ctx context.Context, userID string, text string) (string, error) {
	content := buildTextContent(text)
	body := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(userID).
		MsgType("text").
		Content(content).
		Build()

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("user_id").
		Body(body).
		Build()

	resp, err := m.client.Im.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("messenger: send text to %s: %w", userID, err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("messenger: send text to %s: code=%d, msg=%s", userID, resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil {
		return "", fmt.Errorf("messenger: send text to %s: empty message_id", userID)
	}
	m.logger.Debug("messenger: text sent", zap.String("user_id", userID), zap.String("message_id", *resp.Data.MessageId))
	return *resp.Data.MessageId, nil
}

// SendCard 发送卡片消息给指定用户（通过 user_id）。
func (m *messenger) SendCard(ctx context.Context, userID string, cardJSON string) (string, error) {
	content := buildCardContent(cardJSON)
	body := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(userID).
		MsgType("interactive").
		Content(content).
		Build()

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("user_id").
		Body(body).
		Build()

	resp, err := m.client.Im.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("messenger: send card to %s: %w", userID, err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("messenger: send card to %s: code=%d, msg=%s", userID, resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil {
		return "", fmt.Errorf("messenger: send card to %s: empty message_id", userID)
	}
	m.logger.Debug("messenger: card sent", zap.String("user_id", userID), zap.String("message_id", *resp.Data.MessageId))
	return *resp.Data.MessageId, nil
}

// ReplyText 回复文本消息。
func (m *messenger) ReplyText(ctx context.Context, messageID string, text string) (string, error) {
	content := buildTextContent(text)
	body := larkim.NewReplyMessageReqBodyBuilder().
		MsgType("text").
		Content(content).
		Build()

	req := larkim.NewReplyMessageReqBuilder().
		MessageId(messageID).
		Body(body).
		Build()

	resp, err := m.client.Im.Message.Reply(ctx, req)
	if err != nil {
		return "", fmt.Errorf("messenger: reply text to %s: %w", messageID, err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("messenger: reply text to %s: code=%d, msg=%s", messageID, resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil {
		return "", fmt.Errorf("messenger: reply text to %s: empty message_id", messageID)
	}
	m.logger.Debug("messenger: text replied", zap.String("message_id", messageID), zap.String("reply_id", *resp.Data.MessageId))
	return *resp.Data.MessageId, nil
}

// ReplyCard 回复卡片消息。
func (m *messenger) ReplyCard(ctx context.Context, messageID string, cardJSON string) (string, error) {
	content := buildCardContent(cardJSON)
	body := larkim.NewReplyMessageReqBodyBuilder().
		MsgType("interactive").
		Content(content).
		Build()

	req := larkim.NewReplyMessageReqBuilder().
		MessageId(messageID).
		Body(body).
		Build()

	resp, err := m.client.Im.Message.Reply(ctx, req)
	if err != nil {
		return "", fmt.Errorf("messenger: reply card to %s: %w", messageID, err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("messenger: reply card to %s: code=%d, msg=%s", messageID, resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil {
		return "", fmt.Errorf("messenger: reply card to %s: empty message_id", messageID)
	}
	m.logger.Debug("messenger: card replied", zap.String("message_id", messageID), zap.String("reply_id", *resp.Data.MessageId))
	return *resp.Data.MessageId, nil
}

// buildTextContent 将文本包装为飞书文本消息 content JSON。
func buildTextContent(text string) string {
	type textMsg struct {
		Text string `json:"text"`
	}
	b, _ := json.Marshal(textMsg{Text: text})
	return string(b)
}

// buildCardContent 卡片内容直接透传。
func buildCardContent(cardJSON string) string {
	return cardJSON
}
