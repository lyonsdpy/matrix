package kafka

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProducer_Send_Serialization(t *testing.T) {
	payload := json.RawMessage(`{"departments":[],"users":[]}`)
	msg := Message{
		Type:      TypeOrgSync,
		Source:    "collector",
		Timestamp: time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC),
		TraceID:   "test-trace-id",
		Payload:   payload,
	}

	record, err := buildRecord("secflow.raw", msg)
	if err != nil {
		t.Fatalf("buildRecord error: %v", err)
	}

	if record.Topic != "secflow.raw" {
		t.Errorf("topic = %q, want %q", record.Topic, "secflow.raw")
	}

	var got Message
	if err := json.Unmarshal(record.Value, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"type", string(got.Type), string(TypeOrgSync)},
		{"source", got.Source, "collector"},
		{"trace_id", got.TraceID, "test-trace-id"},
		{"payload", string(got.Payload), string(payload)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}

	if !got.Timestamp.Equal(msg.Timestamp) {
		t.Errorf("timestamp = %v, want %v", got.Timestamp, msg.Timestamp)
	}
}
