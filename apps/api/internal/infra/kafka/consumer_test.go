package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

func makeTestRecord(t *testing.T, msg Message) *kgo.Record {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &kgo.Record{Value: data}
}

func newTestConsumer() *Consumer {
	return &Consumer{
		handlers: make(map[Type]Handler),
		logger:   zap.NewNop(),
	}
}

func TestConsumer_processRecord_RouteCorrect(t *testing.T) {
	c := newTestConsumer()

	var called bool
	var calledMsg Message
	c.Register(TypeOrgSync, func(_ context.Context, msg Message) error {
		called = true
		calledMsg = msg
		return nil
	})

	msg := Message{
		Type:      TypeOrgSync,
		Source:    "test",
		Timestamp: time.Now(),
		TraceID:   "trace-route",
		Payload:   json.RawMessage(`{}`),
	}

	c.processRecord(context.Background(), makeTestRecord(t, msg))

	if !called {
		t.Fatal("handler was not called")
	}
	if calledMsg.Type != TypeOrgSync {
		t.Errorf("calledMsg.Type = %q, want %q", calledMsg.Type, TypeOrgSync)
	}
	if calledMsg.TraceID != "trace-route" {
		t.Errorf("calledMsg.TraceID = %q, want %q", calledMsg.TraceID, "trace-route")
	}
}

func TestConsumer_processRecord_UnregisteredTypeSkipped(t *testing.T) {
	c := newTestConsumer()

	var called bool
	c.Register(TypeOrgSync, func(_ context.Context, _ Message) error {
		called = true
		return nil
	})

	// DeviceSync 未注册，应跳过
	msg := Message{
		Type:      TypeDeviceSync,
		Source:    "test",
		Timestamp: time.Now(),
		TraceID:   "trace-skip",
		Payload:   json.RawMessage(`{}`),
	}

	c.processRecord(context.Background(), makeTestRecord(t, msg))

	if called {
		t.Error("handler should not be called for unregistered type")
	}
}

func TestConsumer_processRecord_HandlerErrorNotInterrupt(t *testing.T) {
	c := newTestConsumer()

	callCount := 0
	c.Register(TypeOrgSync, func(_ context.Context, _ Message) error {
		callCount++
		return errors.New("simulated handler error")
	})

	msg := Message{
		Type:      TypeOrgSync,
		Source:    "test",
		Timestamp: time.Now(),
		TraceID:   "trace-err",
		Payload:   json.RawMessage(`{}`),
	}
	record := makeTestRecord(t, msg)

	// 第一次调用出错，不应阻止第二次调用
	c.processRecord(context.Background(), record)
	c.processRecord(context.Background(), record)

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}
