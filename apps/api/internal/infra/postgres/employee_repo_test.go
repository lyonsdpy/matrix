package postgres

import (
	"context"
	"testing"
	"time"

	"matrix/api/domain/identity"
)

func TestEmployeeRepo_Save_FindByExternalID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	emp := &identity.Employee{
		ID:            "11111111-0000-0000-0000-000000000001",
		ExternalID:    "ext-001",
		Name:          "张三",
		Email:         "zhangsan@example.com",
		Mobile:        "13800000001",
		Status:        1,
		DepartmentIDs: []string{"d1111111-0000-0000-0000-000000000001"},
	}

	if err := repo.Save(ctx, emp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "ext-001")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got == nil {
		t.Fatal("expected employee, got nil")
		return
	}
	if got.Name != emp.Name {
		t.Errorf("Name: want %q, got %q", emp.Name, got.Name)
	}
	if len(got.DepartmentIDs) != 1 || got.DepartmentIDs[0] != emp.DepartmentIDs[0] {
		t.Errorf("DepartmentIDs: want %v, got %v", emp.DepartmentIDs, got.DepartmentIDs)
	}
}

func TestEmployeeRepo_Save_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	emp := &identity.Employee{
		ID:            "22222222-0000-0000-0000-000000000001",
		ExternalID:    "ext-002",
		Name:          "李四",
		Status:        1,
		DepartmentIDs: []string{},
	}

	if err := repo.Save(ctx, emp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, emp.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected employee, got nil")
		return
	}
	if got.ID != emp.ID {
		t.Errorf("ID: want %q, got %q", emp.ID, got.ID)
	}
}

func TestEmployeeRepo_FindByExternalID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	got, err := repo.FindByExternalID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindByExternalID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestEmployeeRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	got, err := repo.FindByID(ctx, "33333333-0000-0000-0000-000000000099")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestEmployeeRepo_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	for i, name := range []string{"王五", "赵六"} {
		emp := &identity.Employee{
			ID:            "44444444-0000-0000-0000-00000000000" + string(rune('1'+i)),
			ExternalID:    "ext-find-all-" + string(rune('1'+i)),
			Name:          name,
			Status:        1,
			DepartmentIDs: []string{},
		}
		if err := repo.Save(ctx, emp); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 employees, got %d", len(all))
	}
}

func TestEmployeeRepo_SaveBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	emps := []identity.Employee{
		{ID: "55555555-0000-0000-0000-000000000001", ExternalID: "ext-batch-1", Name: "批量一", Status: 1, DepartmentIDs: []string{}},
		{ID: "55555555-0000-0000-0000-000000000002", ExternalID: "ext-batch-2", Name: "批量二", Status: 1, DepartmentIDs: []string{}},
	}

	if err := repo.SaveBatch(ctx, emps); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 employees, got %d", len(all))
	}
}

func TestEmployeeRepo_Save_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	emp := &identity.Employee{
		ID:            "66666666-0000-0000-0000-000000000001",
		ExternalID:    "ext-upsert",
		Name:          "原名",
		Status:        1,
		DepartmentIDs: []string{},
	}
	if err := repo.Save(ctx, emp); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	emp.Name = "新名"
	emp.Email = "new@example.com"
	if err := repo.Save(ctx, emp); err != nil {
		t.Fatalf("second Save (upsert): %v", err)
	}

	got, err := repo.FindByID(ctx, emp.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Name != "新名" {
		t.Errorf("Name after upsert: want %q, got %q", "新名", got.Name)
	}
	if got.Email != "new@example.com" {
		t.Errorf("Email after upsert: want %q, got %q", "new@example.com", got.Email)
	}
}

func TestEmployeeRepo_DeleteByExternalID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	emp := &identity.Employee{
		ID:            "77777777-0000-0000-0000-000000000001",
		ExternalID:    "ext-delete",
		Name:          "待删除",
		Status:        1,
		DepartmentIDs: []string{},
	}
	if err := repo.Save(ctx, emp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.DeleteByExternalID(ctx, "ext-delete"); err != nil {
		t.Fatalf("DeleteByExternalID: %v", err)
	}

	got, err := repo.FindByExternalID(ctx, "ext-delete")
	if err != nil {
		t.Fatalf("FindByExternalID after delete: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

func TestEmployeeRepo_DepartmentIDs_JSONB(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmployeeRepository(db)
	ctx := context.Background()

	deptIDs := []string{
		"aaaaaaaa-0000-0000-0000-000000000001",
		"aaaaaaaa-0000-0000-0000-000000000002",
		"aaaaaaaa-0000-0000-0000-000000000003",
	}

	emp := &identity.Employee{
		ID:            "88888888-0000-0000-0000-000000000001",
		ExternalID:    "ext-jsonb",
		Name:          "多部门员工",
		Status:        1,
		DepartmentIDs: deptIDs,
		UpdatedAt:     time.Now(),
	}
	if err := repo.Save(ctx, emp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, emp.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected employee, got nil")
		return
	}
	if len(got.DepartmentIDs) != len(deptIDs) {
		t.Fatalf("DepartmentIDs len: want %d, got %d", len(deptIDs), len(got.DepartmentIDs))
	}
	for i, id := range deptIDs {
		if got.DepartmentIDs[i] != id {
			t.Errorf("DepartmentIDs[%d]: want %q, got %q", i, id, got.DepartmentIDs[i])
		}
	}
}
