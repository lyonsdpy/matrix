// Package kafka 提供 franz-go kgo.Client 的初始化封装。
package kafka

import (
	"errors"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// ErrEmptyBrokers 当 brokers 为空时返回。
var ErrEmptyBrokers = errors.New("kafka: brokers must not be empty")

// NewClient 创建 franz-go kgo.Client。
// brokers 为 Kafka 地址列表，如 []string{"localhost:9092"}。
// collector 用：不传额外 opts（仅生产者模式）。
// sink 用：通过 opts 追加 ConsumerGroup 和 ConsumeTopics。
func NewClient(brokers []string, opts ...kgo.Opt) (*kgo.Client, error) {
	if len(brokers) == 0 {
		return nil, ErrEmptyBrokers
	}
	baseOpts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		// 支持大型 org sync payload（部门数量多时单条消息可达数十 MB）。
		kgo.ProducerBatchMaxBytes(50 * 1024 * 1024),
	}
	baseOpts = append(baseOpts, opts...)
	client, err := kgo.NewClient(baseOpts...)
	if err != nil {
		return nil, fmt.Errorf("kafka: new client: %w", err)
	}
	return client, nil
}
