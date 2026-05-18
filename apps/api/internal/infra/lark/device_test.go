package lark

import (
	"fmt"
	"testing"

	larksecurity "github.com/larksuite/oapi-sdk-go/v3/service/security_and_compliance/v2"

	domain "matrix/api/domain/lark"
)

func TestConvertDevice(t *testing.T) {
	tests := []struct {
		name  string
		input *larksecurity.DeviceRecord
		want  domain.Device
	}{
		{
			name:  "nil device",
			input: nil,
			want:  domain.Device{},
		},
		{
			name: "all fields",
			input: &larksecurity.DeviceRecord{
				DeviceRecordId:     ptrStr("device001"),
				DeviceName:         ptrStr("MacBook Pro"),
				DeviceTerminalType: ptrInt(2),
				CurrentUserId:      ptrStr("user123"),
				DeviceStatus:       ptrInt(1),
				DeviceOwnership:    ptrInt(1),
				SerialNumber:       ptrStr("SN123456"),
			},
			want: domain.Device{
				DeviceID:     "device001",
				DeviceName:   "MacBook Pro",
				Platform:     fmt.Sprintf("%d", 2),
				UserID:       "user123",
				TrustLevel:   fmt.Sprintf("%d", 1),
				Status:       fmt.Sprintf("%d", 1),
				SerialNumber: "SN123456",
			},
		},
		{
			name: "nil optional fields",
			input: &larksecurity.DeviceRecord{
				DeviceRecordId: ptrStr("device002"),
			},
			want: domain.Device{
				DeviceID: "device002",
			},
		},
		{
			name: "different terminal type",
			input: &larksecurity.DeviceRecord{
				DeviceRecordId:     ptrStr("device003"),
				DeviceTerminalType: ptrInt(4),
				DeviceStatus:       ptrInt(3),
			},
			want: domain.Device{
				DeviceID:   "device003",
				Platform:   fmt.Sprintf("%d", 4),
				TrustLevel: fmt.Sprintf("%d", 3),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDevice(tt.input)
			if got.DeviceID != tt.want.DeviceID {
				t.Errorf("DeviceID = %q, want %q", got.DeviceID, tt.want.DeviceID)
			}
			if got.DeviceName != tt.want.DeviceName {
				t.Errorf("DeviceName = %q, want %q", got.DeviceName, tt.want.DeviceName)
			}
			if got.Platform != tt.want.Platform {
				t.Errorf("Platform = %q, want %q", got.Platform, tt.want.Platform)
			}
			if got.UserID != tt.want.UserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.want.UserID)
			}
			if got.TrustLevel != tt.want.TrustLevel {
				t.Errorf("TrustLevel = %q, want %q", got.TrustLevel, tt.want.TrustLevel)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if got.SerialNumber != tt.want.SerialNumber {
				t.Errorf("SerialNumber = %q, want %q", got.SerialNumber, tt.want.SerialNumber)
			}
		})
	}
}
