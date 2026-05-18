package postgres

import (
	"context"
	"testing"

	"matrix/api/domain/identity"
)

func TestDepartmentRepo_Save_FindByExternalID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDepartmentRepository(db)
	ctx := context.Background()

	dept := &identity.Department{
		ID:          "de001111-0000-0000-0000-000000000001",
		ExternalID:  "dept-ext-001",
		Name:        "技术部",
		MemberCount: 10,
	}

	if err := repo.Save(ctx, dept); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "dept-ext-001")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got == nil {
		t.Fatal("expected department, got nil")
		return
	}
	if got.Name != dept.Name {
		t.Errorf("Name: want %q, got %q", dept.Name, got.Name)
	}
	if got.ParentID != "" {
		t.Errorf("ParentID: want empty, got %q", got.ParentID)
	}
}

func TestDepartmentRepo_Save_WithParent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDepartmentRepository(db)
	ctx := context.Background()

	parent := &identity.Department{
		ID:         "de002222-0000-0000-0000-000000000001",
		ExternalID: "dept-parent",
		Name:       "母公司",
	}
	if err := repo.Save(ctx, parent); err != nil {
		t.Fatalf("Save parent: %v", err)
	}

	child := &identity.Department{
		ID:         "de002222-0000-0000-0000-000000000002",
		ExternalID: "dept-child",
		Name:       "子部门",
		ParentID:   parent.ID,
		LeaderID:   "eeeeeeee-0000-0000-0000-000000000001",
	}
	if err := repo.Save(ctx, child); err != nil {
		t.Fatalf("Save child: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "dept-child")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got.ParentID != parent.ID {
		t.Errorf("ParentID: want %q, got %q", parent.ID, got.ParentID)
	}
}

func TestDepartmentRepo_FindByExternalID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDepartmentRepository(db)
	ctx := context.Background()

	got, err := repo.FindByExternalID(ctx, "nonexistent-dept")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDepartmentRepo_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDepartmentRepository(db)
	ctx := context.Background()

	for i, name := range []string{"部门A", "部门B", "部门C"} {
		d := &identity.Department{
			ID:         "de003333-0000-0000-0000-00000000000" + string(rune('1'+i)),
			ExternalID: "dept-all-" + string(rune('1'+i)),
			Name:       name,
		}
		if err := repo.Save(ctx, d); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) < 3 {
		t.Errorf("expected at least 3, got %d", len(all))
	}
}

func TestDepartmentRepo_SaveBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDepartmentRepository(db)
	ctx := context.Background()

	depts := []identity.Department{
		{ID: "de004444-0000-0000-0000-000000000001", ExternalID: "dept-batch-1", Name: "批量部门1"},
		{ID: "de004444-0000-0000-0000-000000000002", ExternalID: "dept-batch-2", Name: "批量部门2"},
	}
	if err := repo.SaveBatch(ctx, depts); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2, got %d", len(all))
	}
}

func TestDepartmentRepo_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDepartmentRepository(db)
	ctx := context.Background()

	dept := &identity.Department{
		ID:          "de005555-0000-0000-0000-000000000001",
		ExternalID:  "dept-upsert",
		Name:        "原部门",
		MemberCount: 5,
	}
	if err := repo.Save(ctx, dept); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	dept.Name = "新部门"
	dept.MemberCount = 15
	if err := repo.Save(ctx, dept); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "dept-upsert")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got.Name != "新部门" {
		t.Errorf("Name: want %q, got %q", "新部门", got.Name)
	}
	if got.MemberCount != 15 {
		t.Errorf("MemberCount: want 15, got %d", got.MemberCount)
	}
}

func TestDepartmentRepo_DeleteByExternalID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDepartmentRepository(db)
	ctx := context.Background()

	dept := &identity.Department{
		ID:         "de006666-0000-0000-0000-000000000001",
		ExternalID: "dept-delete",
		Name:       "待删除部门",
	}
	if err := repo.Save(ctx, dept); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.DeleteByExternalID(ctx, "dept-delete"); err != nil {
		t.Fatalf("DeleteByExternalID: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "dept-delete")
	if err != nil {
		t.Fatalf("FindByExternalID after delete: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}
