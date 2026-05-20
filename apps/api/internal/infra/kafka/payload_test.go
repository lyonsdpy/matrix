package kafka_test

import (
	"encoding/json"
	"testing"
	"time"

	"matrix/api/internal/infra/aisacg"
	desktop "matrix/api/internal/infra/desktop"
	kafka "matrix/api/internal/infra/kafka"
)

// TestNewPayloadsJSONRoundTrip 验证 5 个新 Payload struct 的 JSON 序列化/反序列化一致性。
func TestNewPayloadsJSONRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload any
		target  func() any // 返回一个指向空目标的指针
	}{
		{
			name: "DesktopDeviceSyncPayload",
			payload: kafka.DesktopDeviceSyncPayload{
				Devices: []desktop.Device{
					{
						DeviceID:   1,
						IP:         "192.168.1.10",
						DevName:    "PC-001",
						DeviceType: desktop.DeviceTypeWindowsPC,
						Online:     true,
						LastTime:   time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC),
					},
				},
			},
			target: func() any { return &kafka.DesktopDeviceSyncPayload{} },
		},
		{
			name: "DesktopSoftwareSyncPayload",
			payload: kafka.DesktopSoftwareSyncPayload{
				DeviceID:   1,
				DeviceName: "PC-001",
				Softwares: []desktop.InstalledSoftware{
					{
						DeviceID:    1,
						DisplayName: "Go 1.25",
						Version:     "1.25.0",
						Vendor:      "Google",
					},
				},
			},
			target: func() any { return &kafka.DesktopSoftwareSyncPayload{} },
		},
		{
			name: "DesktopHardwareSyncPayload",
			payload: kafka.DesktopHardwareSyncPayload{
				DeviceID: 1,
				Components: []desktop.HardwareComponent{
					{
						DeviceID: 1,
						Name:     "Intel Core i7-12700",
						Type:     "CPU",
					},
				},
			},
			target: func() any { return &kafka.DesktopHardwareSyncPayload{} },
		},
		{
			name: "ACGWhitelistSyncPayload",
			payload: kafka.ACGWhitelistSyncPayload{
				ACGDevice: "acg-01",
				Entries: []aisacg.WhitelistEntry{
					{
						Enable: true,
						Name:   "allow-github",
						Desc:   "GitHub 访问白名单",
						Addrs:  []string{"140.82.112.0/20", "192.30.252.0/22"},
					},
				},
			},
			target: func() any { return &kafka.ACGWhitelistSyncPayload{} },
		},
		{
			name: "ACGUserSyncPayload",
			payload: kafka.ACGUserSyncPayload{
				ACGDevice: "acg-01",
				Users: []aisacg.User{
					{
						Path:   "/org/it",
						Name:   "alice",
						Enable: "true",
					},
				},
			},
			target: func() any { return &kafka.ACGUserSyncPayload{} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			got := tc.target()
			if err := json.Unmarshal(data, got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// 二次序列化后与第一次结果相等，即 round-trip 一致
			data2, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("second Marshal failed: %v", err)
			}
			if string(data) != string(data2) {
				t.Errorf("round-trip mismatch:\n  first:  %s\n  second: %s", data, data2)
			}
		})
	}
}
