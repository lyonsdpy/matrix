package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"matrix/api/domain/identity"
)

func TestObservedMACBindingRepo_Upsert(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "TRUNCATE observed_mac_bindings"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	repo := NewObservedMACBindingRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	binding := &identity.ObservedMACBinding{
		ID:         uuid.MustParse("ffffffff-0000-0000-0000-000000000001"),
		Office:     "上海-静安",
		IP:         "192.168.10.1",
		MAC:        "aa:bb:cc:dd:ee:ff",
		LastSeenAt: now,
	}

	// 第一次 Upsert
	if err := repo.Upsert(ctx, binding); err != nil {
		t.Fatalf("Upsert first: %v", err)
	}
	// 第二次 Upsert（相同 office/ip/mac，更新 last_seen_at）
	if err := repo.Upsert(ctx, binding); err != nil {
		t.Fatalf("Upsert second: %v", err)
	}

	got, err := repo.FindByIP(ctx, "192.168.10.1")
	if err != nil {
		t.Fatalf("FindByIP: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 binding after two upserts, got %d", len(got))
	}
	if got[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC: want aa:bb:cc:dd:ee:ff, got %q", got[0].MAC)
	}
	if got[0].Office != "上海-静安" {
		t.Errorf("Office: want 上海-静安, got %q", got[0].Office)
	}
}
