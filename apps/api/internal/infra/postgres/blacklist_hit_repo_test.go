package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"matrix/api/domain/violation"
)

func TestBlacklistHitRepo_SaveBatch_FindByEmployeeID(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "TRUNCATE software_blacklist, blacklist_hits"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	repo := NewBlacklistHitRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	hits := []violation.BlacklistHit{
		{
			ID:           uuid.MustParse("aabbccdd-0000-0000-0000-000000000001"),
			EmployeeID:   "emp-001",
			EmployeeName: "张三",
			DeviceID:     "dev-001",
			DeviceName:   "工作机A",
			SoftwareName: "恶意软件X",
			DetectedAt:   now,
		},
		{
			ID:           uuid.MustParse("aabbccdd-0000-0000-0000-000000000002"),
			EmployeeID:   "emp-002",
			EmployeeName: "李四",
			DeviceID:     "dev-002",
			DeviceName:   "工作机B",
			SoftwareName: "恶意软件Y",
			DetectedAt:   now,
		},
	}

	if err := repo.SaveBatch(ctx, hits); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	got, err := repo.FindByEmployeeID(ctx, "emp-001")
	if err != nil {
		t.Fatalf("FindByEmployeeID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(got))
	}
	if got[0].EmployeeID != "emp-001" {
		t.Errorf("EmployeeID: want emp-001, got %q", got[0].EmployeeID)
	}
	if got[0].SoftwareName != "恶意软件X" {
		t.Errorf("SoftwareName: want 恶意软件X, got %q", got[0].SoftwareName)
	}
}

func TestBlacklistHitRepo_FindAll(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "TRUNCATE software_blacklist, blacklist_hits"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	repo := NewBlacklistHitRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	hits := []violation.BlacklistHit{
		{
			ID:         uuid.MustParse("ccddee00-0000-0000-0000-000000000001"),
			EmployeeID: "emp-003",
			DetectedAt: now,
		},
		{
			ID:         uuid.MustParse("ccddee00-0000-0000-0000-000000000002"),
			EmployeeID: "emp-004",
			DetectedAt: now,
		},
	}

	if err := repo.SaveBatch(ctx, hits); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 hits, got %d", len(all))
	}
}
