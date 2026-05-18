// Package kafka 提供 Kafka 生产者与消费者封装。
package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Sender Kafka 消息发送接口，便于测试替换。
type Sender interface {
	Send(ctx context.Context, topic string, msg Message) error
}

// 编译期断言：Producer 实现 Sender 接口。
var _ Sender = (*Producer)(nil)

// Producer Kafka 消息生产者。
type Producer struct {
	client *kgo.Client
}

// NewProducer 创建 Kafka 生产者。
func NewProducer(client *kgo.Client) *Producer {
	return &Producer{client: client}
}

// Send 发送消息到指定 topic。
// 将 Message 序列化为 JSON 后写入 Kafka。
func (p *Producer) Send(ctx context.Context, topic string, msg Message) error {
	record, err := buildRecord(topic, msg)
	if err != nil {
		return err
	}
	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("producer: produce: %w", err)
	}
	return nil
}

// buildRecord 将消息序列化为 kgo.Record。
func buildRecord(topic string, msg Message) (*kgo.Record, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("producer: marshal message: %w", err)
	}
	return &kgo.Record{Topic: topic, Value: data}, nil
}
