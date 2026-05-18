package postgres

import (
	"context"
	"testing"

	"matrix/api/domain/identity"
)

func TestSessionRepo_ReplaceByOffice(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOnlineSessionRepository(db)
	ctx := context.Background()

	office := "上海-虹桥"

	initial := []identity.OnlineSession{
		{Office: office, EmployeeName: "张三", IP: "192.168.1.1", MAC: "AA:BB:CC:DD:EE:01"},
		{Office: office, EmployeeName: "李四", IP: "192.168.1.2", MAC: "AA:BB:CC:DD:EE:02"},
	}
	if err := repo.ReplaceByOffice(ctx, office, initial); err != nil {
		t.Fatalf("first ReplaceByOffice: %v", err)
	}

	// 替换为新数据（旧数据应被清除）
	updated := []identity.OnlineSession{
		{Office: office, EmployeeName: "王五", IP: "192.168.1.3", MAC: "AA:BB:CC:DD:EE:03"},
	}
	if err := repo.ReplaceByOffice(ctx, office, updated); err != nil {
		t.Fatalf("second ReplaceByOffice: %v", err)
	}

	got, err := repo.FindByOffice(ctx, office)
	if err != nil {
		t.Fatalf("FindByOffice: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 session after replace, got %d", len(got))
	}
	if got[0].EmployeeName != "王五" {
		t.Errorf("EmployeeName: want %q, got %q", "王五", got[0].EmployeeName)
	}
}

func TestSessionRepo_ReplaceByOffice_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOnlineSessionRepository(db)
	ctx := context.Background()

	office := "北京-朝阳"

	sessions := []identity.OnlineSession{
		{Office: office, EmployeeName: "张三", IP: "10.0.0.1"},
	}
	if err := repo.ReplaceByOffice(ctx, office, sessions); err != nil {
		t.Fatalf("ReplaceByOffice: %v", err)
	}

	// 用空列表替换 → 清空该 office 数据
	if err := repo.ReplaceByOffice(ctx, office, []identity.OnlineSession{}); err != nil {
		t.Fatalf("ReplaceByOffice with empty: %v", err)
	}

	got, err := repo.FindByOffice(ctx, office)
	if err != nil {
		t.Fatalf("FindByOffice: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 sessions after empty replace, got %d", len(got))
	}
}

func TestSessionRepo_FindByOffice(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOnlineSessionRepository(db)
	ctx := context.Background()

	officeA := "广州-天河"
	officeB := "深圳-南山"

	if err := repo.ReplaceByOffice(ctx, officeA, []identity.OnlineSession{
		{Office: officeA, EmployeeName: "广州员工1", IP: "172.16.0.1"},
		{Office: officeA, EmployeeName: "广州员工2", IP: "172.16.0.2"},
	}); err != nil {
		t.Fatalf("ReplaceByOffice officeA: %v", err)
	}
	if err := repo.ReplaceByOffice(ctx, officeB, []identity.OnlineSession{
		{Office: officeB, EmployeeName: "深圳员工1", IP: "172.16.1.1"},
	}); err != nil {
		t.Fatalf("ReplaceByOffice officeB: %v", err)
	}

	got, err := repo.FindByOffice(ctx, officeA)
	if err != nil {
		t.Fatalf("FindByOffice: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 sessions for officeA, got %d", len(got))
	}
	for _, s := range got {
		if s.Office != officeA {
			t.Errorf("unexpected office %q, want %q", s.Office, officeA)
		}
	}
}

func TestSessionRepo_FindByEmployeeID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOnlineSessionRepository(db)
	ctx := context.Background()

	empID := "e0000001-0000-0000-0000-000000000001"
	otherEmpID := "e0000002-0000-0000-0000-000000000001"
	office1 := "上海-陆家嘴"
	office2 := "北京-望京"

	// 插入目标员工在两个 office 的会话，以及另一员工的会话
	if err := repo.ReplaceByOffice(ctx, office1, []identity.OnlineSession{
		{Office: office1, EmployeeID: empID, EmployeeName: "张三", IP: "10.1.0.1"},
		{Office: office1, EmployeeID: otherEmpID, EmployeeName: "李四", IP: "10.1.0.2"},
	}); err != nil {
		t.Fatalf("ReplaceByOffice office1: %v", err)
	}
	if err := repo.ReplaceByOffice(ctx, office2, []identity.OnlineSession{
		{Office: office2, EmployeeID: empID, EmployeeName: "张三", IP: "10.2.0.1"},
	}); err != nil {
		t.Fatalf("ReplaceByOffice office2: %v", err)
	}

	got, err := repo.FindByEmployeeID(ctx, empID)
	if err != nil {
		t.Fatalf("FindByEmployeeID: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 sessions for empID, got %d", len(got))
	}
	for _, s := range got {
		if s.EmployeeID != empID {
			t.Errorf("unexpected employee_id %q, want %q", s.EmployeeID, empID)
		}
	}
	// 验证按 office 排序
	if len(got) == 2 && got[0].Office > got[1].Office {
		t.Errorf("expected sessions ordered by office, got %q > %q", got[0].Office, got[1].Office)
	}
}

func TestSessionRepo_FindByEmployeeID_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOnlineSessionRepository(db)
	ctx := context.Background()

	got, err := repo.FindByEmployeeID(ctx, "ffffffff-ffff-ffff-ffff-ffffffffffff")
	if err != nil {
		t.Fatalf("FindByEmployeeID: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d sessions", len(got))
	}
}

func TestSessionRepo_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOnlineSessionRepository(db)
	ctx := context.Background()

	offices := []string{"成都-高新", "杭州-西湖"}
	for _, o := range offices {
		if err := repo.ReplaceByOffice(ctx, o, []identity.OnlineSession{
			{Office: o, EmployeeName: "员工", IP: "10.0.0.1"},
		}); err != nil {
			t.Fatalf("ReplaceByOffice %q: %v", o, err)
		}
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(all))
	}
}
