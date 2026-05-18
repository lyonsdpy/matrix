package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// Handler 消息处理函数签名。
type Handler func(ctx context.Context, msg Message) error

// Consumer Kafka 消息消费者，支持按消息类型路由到不同 Handler。
type Consumer struct {
	client   *kgo.Client
	handlers map[Type]Handler
	logger   *zap.Logger
}

// NewConsumer 创建 Kafka 消费者。
// 传入已配置 consumer group 和 topics 的 kgo.Client。
func NewConsumer(client *kgo.Client, logger *zap.Logger) *Consumer {
	return &Consumer{
		client:   client,
		handlers: make(map[Type]Handler),
		logger:   logger,
	}
}

// Register 注册消息类型对应的处理函数。
func (c *Consumer) Register(msgType Type, handler Handler) {
	c.handlers[msgType] = handler
}

// Start 启动消费循环（阻塞，直到 ctx 取消）。
// 流程：
//  1. PollFetches 获取消息批次
//  2. 反序列化 Message 信封
//  3. 按 Type 查找 Handler，未注册类型记录 warn 日志后跳过
//  4. 调用 Handler 处理，处理失败记录 error 日志（不中断消费）
//  5. 提交 offset
func (c *Consumer) Start(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fetchErr := range errs {
				if errors.Is(fetchErr.Err, context.Canceled) {
					return nil
				}
				c.logger.Error("consumer: fetch error",
					zap.String("topic", fetchErr.Topic),
					zap.Error(fetchErr.Err))
			}
		}
		fetches.EachRecord(func(record *kgo.Record) {
			c.processRecord(ctx, record)
		})
		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.logger.Error("consumer: commit offsets", zap.Error(err))
		}
	}
}

// processRecord 处理单条 Kafka 记录。
func (c *Consumer) processRecord(ctx context.Context, record *kgo.Record) {
	var msg Message
	if err := json.Unmarshal(record.Value, &msg); err != nil {
		c.logger.Error("consumer: unmarshal message",
			zap.String("topic", record.Topic),
			zap.Error(fmt.Errorf("unmarshal: %w", err)))
		return
	}
	c.logger.Debug("consumer: message received",
		zap.String("type", string(msg.Type)),
		zap.String("topic", record.Topic),
		zap.String("trace_id", msg.TraceID),
		zap.String("source", msg.Source))
	handler, ok := c.handlers[msg.Type]
	if !ok {
		c.logger.Warn("consumer: unregistered message type",
			zap.String("type", string(msg.Type)))
		return
	}
	if err := handler(ctx, msg); err != nil {
		c.logger.Error("consumer: handler error",
			zap.String("type", string(msg.Type)),
			zap.Error(err))
	}
}
