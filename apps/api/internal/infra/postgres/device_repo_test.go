package postgres

import (
	"context"
	"testing"

	"matrix/api/domain/identity"
)

func TestDeviceRepo_Save_FindByExternalID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	dev := &identity.EmployeeDevice{
		ID:           "de011111-0000-0000-0000-000000000001",
		EmployeeID:   "e0011111-0000-0000-0000-000000000001",
		ExternalID:   "dev-ext-001",
		DeviceName:   "MacBook Pro",
		Platform:     "Mac",
		SerialNumber: "SN001",
		Status:       "active",
		TrustLevel:   "trusted",
	}

	if err := repo.Save(ctx, dev); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "dev-ext-001")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got == nil {
		t.Fatal("expected device, got nil")
		return
	}
	if got.DeviceName != dev.DeviceName {
		t.Errorf("DeviceName: want %q, got %q", dev.DeviceName, got.DeviceName)
	}
	if got.EmployeeID != dev.EmployeeID {
		t.Errorf("EmployeeID: want %q, got %q", dev.EmployeeID, got.EmployeeID)
	}
}

func TestDeviceRepo_FindByExternalID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	got, err := repo.FindByExternalID(ctx, "nonexistent-dev")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDeviceRepo_FindByEmployeeID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	empID := "e0022222-0000-0000-0000-000000000001"

	devs := []identity.EmployeeDevice{
		{ID: "de022222-0000-0000-0000-000000000001", EmployeeID: empID, ExternalID: "dev-emp-1", DeviceName: "设备1", Platform: "Windows"},
		{ID: "de022222-0000-0000-0000-000000000002", EmployeeID: empID, ExternalID: "dev-emp-2", DeviceName: "设备2", Platform: "Mac"},
		{ID: "de022222-0000-0000-0000-000000000003", EmployeeID: "e00aaaaa-0000-0000-0000-000000000001", ExternalID: "dev-emp-3", DeviceName: "其他员工设备", Platform: "Linux"},
	}
	for i := range devs {
		if err := repo.Save(ctx, &devs[i]); err != nil {
			t.Fatalf("Save dev[%d]: %v", i, err)
		}
	}

	got, err := repo.FindByEmployeeID(ctx, empID)
	if err != nil {
		t.Fatalf("FindByEmployeeID: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 devices, got %d", len(got))
	}
}

func TestDeviceRepo_SaveBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	devs := []identity.EmployeeDevice{
		{ID: "de033333-0000-0000-0000-000000000001", ExternalID: "dev-batch-1", DeviceName: "批量设备1", Platform: "Windows"},
		{ID: "de033333-0000-0000-0000-000000000002", ExternalID: "dev-batch-2", DeviceName: "批量设备2", Platform: "Mac"},
	}
	if err := repo.SaveBatch(ctx, devs); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	for _, d := range devs {
		got, err := repo.FindByExternalID(ctx, d.ExternalID)
		if err != nil {
			t.Fatalf("FindByExternalID %q: %v", d.ExternalID, err)
		}
		if got == nil {
			t.Errorf("device %q not found after SaveBatch", d.ExternalID)
		}
	}
}

func TestDeviceRepo_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	dev := &identity.EmployeeDevice{
		ID:         "de044444-0000-0000-0000-000000000001",
		ExternalID: "dev-upsert",
		DeviceName: "原设备名",
		Platform:   "Windows",
		Status:     "active",
	}
	if err := repo.Save(ctx, dev); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	dev.DeviceName = "新设备名"
	dev.Status = "inactive"
	if err := repo.Save(ctx, dev); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "dev-upsert")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got.DeviceName != "新设备名" {
		t.Errorf("DeviceName: want %q, got %q", "新设备名", got.DeviceName)
	}
}

func TestDeviceRepo_DeleteByExternalID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeviceRepository(db)
	ctx := context.Background()

	dev := &identity.EmployeeDevice{
		ID:         "de055555-0000-0000-0000-000000000001",
		ExternalID: "dev-delete",
		DeviceName: "待删除设备",
	}
	if err := repo.Save(ctx, dev); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.DeleteByExternalID(ctx, "dev-delete"); err != nil {
		t.Fatalf("DeleteByExternalID: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "dev-delete")
	if err != nil {
		t.Fatalf("FindByExternalID after delete: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}
