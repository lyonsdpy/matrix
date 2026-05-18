package lark_test

import (
	"errors"
	"testing"

	"matrix/api/internal/infra/lark"
)

func TestNewClient_MissingCredentials(t *testing.T) {
	tests := []struct {
		name      string
		appID     string
		appSecret string
	}{
		{"missing app_id", "", "secret"},
		{"missing app_secret", "id", ""},
		{"both missing", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := lark.Config{AppID: tt.appID, AppSecret: tt.appSecret}
			_, err := lark.NewClient(cfg)
			if !errors.Is(err, lark.ErrMissingCredentials) {
				t.Errorf("error = %v, want ErrMissingCredentials", err)
			}
		})
	}
}

func TestNewClient_ValidCredentials(t *testing.T) {
	cfg := lark.Config{AppID: "test-id", AppSecret: "test-secret"}
	client, err := lark.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}
