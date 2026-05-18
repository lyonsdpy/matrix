package postgres

import (
	"context"
	"testing"
	"time"

	"matrix/api/domain/violation"
)

func TestViolationRepo_Save_FindByOffice(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViolationRepository(db)
	ctx := context.Background()

	office := "上海-静安"
	rec := &violation.ViolationRecord{
		ID:           "a1011111-0000-0000-0000-000000000001",
		EmployeeID:   "e0011111-0000-0000-0000-000000000001",
		EmployeeName: "违规员工",
		Office:       office,
		IP:           "192.168.10.1",
		MAC:          "00:11:22:33:44:55",
		Status:       violation.ComplianceNonCompliant,
		DetectedAt:   time.Now().UTC().Truncate(time.Second),
	}

	if err := repo.Save(ctx, rec); err != nil {
		t.Fatalf("Save: %v", err)
	}

	records, err := repo.FindByOffice(ctx, office)
	if err != nil {
		t.Fatalf("FindByOffice: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	got := records[0]
	if got.ID != rec.ID {
		t.Errorf("ID: want %q, got %q", rec.ID, got.ID)
	}
	if got.Status != rec.Status {
		t.Errorf("Status: want %q, got %q", rec.Status, got.Status)
	}
}

func TestViolationRepo_FindByOffice_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViolationRepository(db)
	ctx := context.Background()

	records, err := repo.FindByOffice(ctx, "不存在的办公室")
	if err != nil {
		t.Fatalf("FindByOffice: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0, got %d", len(records))
	}
}

func TestViolationRepo_SaveBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViolationRepository(db)
	ctx := context.Background()

	office := "北京-海淀"
	records := []violation.ViolationRecord{
		{
			ID:           "a2022222-0000-0000-0000-000000000001",
			EmployeeName: "员工甲",
			Office:       office,
			IP:           "10.0.1.1",
			Status:       violation.ComplianceNonCompliant,
			DetectedAt:   time.Now().UTC().Truncate(time.Second),
		},
		{
			ID:           "a2022222-0000-0000-0000-000000000002",
			EmployeeName: "员工乙",
			Office:       office,
			IP:           "10.0.1.2",
			Status:       violation.ComplianceUnchecked,
			DetectedAt:   time.Now().UTC().Truncate(time.Second),
		},
	}

	if err := repo.SaveBatch(ctx, records); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	got, err := repo.FindByOffice(ctx, office)
	if err != nil {
		t.Fatalf("FindByOffice: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 records, got %d", len(got))
	}
}

func TestViolationRepo_MultiOffice_Isolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViolationRepository(db)
	ctx := context.Background()

	officeA := "南京-鼓楼"
	officeB := "武汉-光谷"

	if err := repo.Save(ctx, &violation.ViolationRecord{
		ID: "a3033333-0000-0000-0000-000000000001", Office: officeA,
		Status: violation.ComplianceNonCompliant, DetectedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Save officeA: %v", err)
	}
	if err := repo.Save(ctx, &violation.ViolationRecord{
		ID: "a3033333-0000-0000-0000-000000000002", Office: officeB,
		Status: violation.ComplianceNonCompliant, DetectedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Save officeB: %v", err)
	}

	gotA, _ := repo.FindByOffice(ctx, officeA)
	if len(gotA) != 1 {
		t.Errorf("officeA: expected 1, got %d", len(gotA))
	}
	gotB, _ := repo.FindByOffice(ctx, officeB)
	if len(gotB) != 1 {
		t.Errorf("officeB: expected 1, got %d", len(gotB))
	}
}
