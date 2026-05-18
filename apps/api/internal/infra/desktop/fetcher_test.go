package desktop_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"matrix/api/internal/infra/desktop"
	"matrix/api/internal/testutil"
)

// requireDSN 从 testdata/config.toml 读取 Desktop MySQL DSN，配置不存在时 skip。
func requireDSN(t *testing.T) string {
	t.Helper()
	cfg, err := testutil.LoadConfig()
	if err != nil || cfg == nil || cfg.Desktop.IsEmpty() {
		t.Skip("no desktop config in testdata/config.toml, skipping integration test")
	}
	return cfg.Desktop.DSN()
}

func TestDeviceFetcher_ListDevices(t *testing.T) {
	dsn := requireDSN(t)

	db, err := desktop.NewDB(dsn)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("db.Close: %v", closeErr)
		}
	})

	fetcher := desktop.NewDeviceFetcher(db, zap.NewNop())
	ctx := context.Background()

	devices, err := fetcher.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("expected at least one device, got none")
	}

	for _, d := range devices {
		if d.DeviceID <= 0 {
			t.Errorf("device %+v: DeviceID must be > 0", d)
		}
		if d.IP == "" {
			t.Errorf("device %d: IP must not be empty", d.DeviceID)
		}
		if d.OSName == "" {
			t.Errorf("device %d: OSName must not be empty (LEFT JOIN verification)", d.DeviceID)
		}
		t.Logf("device %+v", d)
	}
}

func TestDeviceFetcher_ListInstalledSoftware(t *testing.T) {
	dsn := requireDSN(t)

	db, err := desktop.NewDB(dsn)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("db.Close: %v", closeErr)
		}
	})

	fetcher := desktop.NewDeviceFetcher(db, zap.NewNop())
	ctx := context.Background()

	devices, err := fetcher.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices available, skipping software test")
	}

	targetID := devices[0].DeviceID
	software, err := fetcher.ListInstalledSoftware(ctx, targetID)
	if err != nil {
		t.Fatalf("ListInstalledSoftware(deviceID=%d): %v", targetID, err)
	}
	if len(software) == 0 {
		t.Fatalf("expected at least one installed software entry for deviceID=%d", targetID)
	}

	for _, s := range software {
		if s.DisplayName == "" {
			t.Errorf("software entry for device %d: DisplayName must not be empty", targetID)
		}
		if s.DeviceID != targetID {
			t.Errorf("software entry: DeviceID=%d, expected %d", s.DeviceID, targetID)
		}
	}
}

func TestDeviceFetcher_ListHardwareComponents(t *testing.T) {
	dsn := requireDSN(t)

	db, err := desktop.NewDB(dsn)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("db.Close: %v", closeErr)
		}
	})

	fetcher := desktop.NewDeviceFetcher(db, zap.NewNop())
	ctx := context.Background()

	devices, err := fetcher.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices available, skipping hardware test")
	}

	targetID := devices[0].DeviceID
	components, err := fetcher.ListHardwareComponents(ctx, targetID)
	if err != nil {
		t.Fatalf("ListHardwareComponents(deviceID=%d): %v", targetID, err)
	}
	if len(components) == 0 {
		t.Fatalf("expected at least one hardware component for deviceID=%d", targetID)
	}

	for _, c := range components {
		if c.Type == "" {
			t.Errorf("hardware component for device %d: Type must not be empty", targetID)
		}
		if c.DeviceID != targetID {
			t.Errorf("hardware component: DeviceID=%d, expected %d", c.DeviceID, targetID)
		}
	}
}
