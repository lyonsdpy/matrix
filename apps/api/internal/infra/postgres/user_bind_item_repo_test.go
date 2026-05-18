package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"matrix/api/domain/identity"
)

func TestUserBindItemRepo_ReplaceByDevice(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "TRUNCATE user_bind_items"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	repo := NewUserBindItemRepository(db)
	device := "acg-device-001"

	initial := []identity.UserBindItem{
		{
			ID:        uuid.MustParse("11111111-0000-0000-0000-000000000001"),
			ACGDevice: device,
			UserPath:  "org/team/user1",
			BindType:  "bind_include",
			Address:   "192.168.1.1",
			IsExclude: false,
		},
		{
			ID:        uuid.MustParse("11111111-0000-0000-0000-000000000002"),
			ACGDevice: device,
			UserPath:  "org/team/user2",
			BindType:  "bind_include",
			Address:   "192.168.1.2",
			IsExclude: false,
		},
	}
	if err := repo.ReplaceByDevice(ctx, device, initial); err != nil {
		t.Fatalf("ReplaceByDevice initial: %v", err)
	}

	// 替换为只有 1 条
	replacement := []identity.UserBindItem{
		{
			ID:        uuid.MustParse("22222222-0000-0000-0000-000000000001"),
			ACGDevice: device,
			UserPath:  "org/team/user3",
			BindType:  "bind_exclude",
			Address:   "10.0.0.1",
			IsExclude: true,
		},
	}
	if err := repo.ReplaceByDevice(ctx, device, replacement); err != nil {
		t.Fatalf("ReplaceByDevice replacement: %v", err)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 item after replace, got %d", len(all))
	}
	if all[0].UserPath != "org/team/user3" {
		t.Errorf("UserPath: want org/team/user3, got %q", all[0].UserPath)
	}
	if !all[0].IsExclude {
		t.Errorf("IsExclude: want true, got false")
	}
}
