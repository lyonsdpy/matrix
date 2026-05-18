package aisdesk_test

import (
	"testing"

	"matrix/api/domain/aisdesk"
)

func TestDeviceTypeConstants(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"DeviceTypeWindowsPC", aisdesk.DeviceTypeWindowsPC, 101},
		{"DeviceTypeAndroid", aisdesk.DeviceTypeAndroid, 103},
		{"DeviceTypeMac", aisdesk.DeviceTypeMac, 110},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestDevice_ZeroValueSafe(t *testing.T) {
	var d aisdesk.Device
	// 零值不应触发 panic，所有字段可正常访问
	_ = d.DeviceID
	_ = d.IP
	_ = d.LastTime
}

func TestInstalledSoftware_InstallDateZeroValue(t *testing.T) {
	var s aisdesk.InstalledSoftware
	if s.InstallDate != "" {
		t.Errorf("InstallDate zero value = %q, want empty string", s.InstallDate)
	}
}
