//go:build integration

package kafka_test

import (
	"context"
	"testing"
	"time"

	"matrix/api/internal/infra/kafka"
	"matrix/api/internal/testutil"
)

// TestNewClient_RealBroker_Ping 验证本地 Kafka 真实连接。
// 运行方式：go test -tags integration ./internal/infra/kafka/...
func TestNewClient_RealBroker_Ping(t *testing.T) {
	cfg, err := testutil.LoadConfig()
	if err != nil || cfg == nil || cfg.Kafka.IsEmpty() {
		t.Skip("no kafka config in testdata/config.toml, skipping")
	}

	client, err := kafka.NewClient(cfg.Kafka.Brokers)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v，请确认本地 Kafka 已在配置的 broker 地址启动", err)
	}
}
