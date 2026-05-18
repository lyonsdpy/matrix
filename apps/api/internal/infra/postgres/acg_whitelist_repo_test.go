package postgres

import (
	"context"
	"testing"

	"matrix/api/domain/aisacg"
)

func TestACGWhitelistRepo_ReplaceByDevice(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "TRUNCATE acg_whitelist_entries"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	repo := NewACGWhitelistRepository(db)
	device := "acg-device-001"

	initial := []aisacg.WhitelistEntry{
		{Enable: true, Name: "白名单A", Desc: "描述A", Addrs: []string{"192.168.1.0/24"}},
		{Enable: false, Name: "白名单B", Desc: "描述B", Addrs: []string{"10.0.0.1"}},
	}
	if err := repo.ReplaceByDevice(ctx, device, initial); err != nil {
		t.Fatalf("ReplaceByDevice initial: %v", err)
	}

	// 替换为只有 1 条
	replacement := []aisacg.WhitelistEntry{
		{Enable: true, Name: "白名单C", Desc: "描述C", Addrs: []string{"172.16.0.0/12", "10.10.0.1"}},
	}
	if err := repo.ReplaceByDevice(ctx, device, replacement); err != nil {
		t.Fatalf("ReplaceByDevice replacement: %v", err)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 entry after replace, got %d", len(all))
	}
	if all[0].Name != "白名单C" {
		t.Errorf("Name: want 白名单C, got %q", all[0].Name)
	}
	if len(all[0].Addrs) != 2 {
		t.Errorf("Addrs: want 2, got %d", len(all[0].Addrs))
	}
}
