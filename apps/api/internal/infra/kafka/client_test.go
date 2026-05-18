package kafka_test

import (
	"errors"
	"testing"

	"matrix/api/internal/infra/kafka"
)

func TestNewClient_EmptyBrokers_ReturnsError(t *testing.T) {
	tests := []struct {
		name    string
		brokers []string
	}{
		{"nil brokers", nil},
		{"empty slice", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := kafka.NewClient(tt.brokers)
			if !errors.Is(err, kafka.ErrEmptyBrokers) {
				t.Errorf("error = %v, want ErrEmptyBrokers", err)
			}
		})
	}
}
