package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"matrix/api/internal/infra/queue"
)

func TestBotInteractionRepo_Enqueue(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBotInteractionRepository(db)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"user_id": "u001", "chat_id": "c001"})
	item := queue.BotInteraction{
		EventType: queue.BotEventChatEntered,
		UserID:    "u001",
		Payload:   payload,
	}

	if err := repo.Enqueue(ctx, item); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// 写入后应存在一条 pending 记录
	rows, err := db.QueryContext(ctx, `SELECT id, event_type, user_id, status FROM bot_interactions`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var count int
	for rows.Next() {
		var id int64
		var eventType, userID, status string
		if err := rows.Scan(&id, &eventType, &userID, &status); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if eventType != queue.BotEventChatEntered {
			t.Errorf("event_type: want %q, got %q", queue.BotEventChatEntered, eventType)
		}
		if userID != "u001" {
			t.Errorf("user_id: want %q, got %q", "u001", userID)
		}
		if status != queue.StatusPending {
			t.Errorf("status: want %q, got %q", queue.StatusPending, status)
		}
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestBotInteractionRepo_DequeueBatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBotInteractionRepository(db)
	ctx := context.Background()

	// 写入 3 条 pending
	for i, eventType := range []string{queue.BotEventChatEntered, queue.BotEventMessage, queue.BotEventCardAction} {
		payload, _ := json.Marshal(map[string]string{"user_id": "u00" + string(rune('1'+i))})
		if err := repo.Enqueue(ctx, queue.BotInteraction{
			EventType: eventType,
			UserID:    "u00" + string(rune('1'+i)),
			Payload:   payload,
		}); err != nil {
			t.Fatalf("Enqueue %d: %v", i, err)
		}
	}

	// DequeueBatch(ctx, 2) 应返回 2 条，状态改为 processing
	items, err := repo.DequeueBatch(ctx, 2)
	if err != nil {
		t.Fatalf("DequeueBatch: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for _, item := range items {
		if item.Status != queue.StatusProcessing {
			t.Errorf("status after dequeue: want %q, got %q", queue.StatusProcessing, item.Status)
		}
		if item.ID == 0 {
			t.Error("ID should not be 0")
		}
	}

	// 再次取，应只剩 1 条（SKIP LOCKED 不返回已 processing 的记录）
	remaining, err := repo.DequeueBatch(ctx, 10)
	if err != nil {
		t.Fatalf("second DequeueBatch: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining, got %d", len(remaining))
	}
}

func TestBotInteractionRepo_DequeueBatch_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBotInteractionRepository(db)
	ctx := context.Background()

	items, err := repo.DequeueBatch(ctx, 10)
	if err != nil {
		t.Fatalf("DequeueBatch empty: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestBotInteractionRepo_MarkDone(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBotInteractionRepository(db)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"user_id": "u001"})
	if err := repo.Enqueue(ctx, queue.BotInteraction{
		EventType: queue.BotEventMessage,
		UserID:    "u001",
		Payload:   payload,
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	items, err := repo.DequeueBatch(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("DequeueBatch: %v, items=%d", err, len(items))
	}

	if err := repo.MarkDone(ctx, items[0].ID); err != nil {
		t.Fatalf("MarkDone: %v", err)
	}

	var status string
	var processedAt *time.Time
	err = db.QueryRowContext(ctx,
		`SELECT status, processed_at FROM bot_interactions WHERE id = $1`, items[0].ID,
	).Scan(&status, &processedAt)
	if err != nil {
		t.Fatalf("query after MarkDone: %v", err)
	}
	if status != queue.StatusDone {
		t.Errorf("status: want %q, got %q", queue.StatusDone, status)
	}
	if processedAt == nil {
		t.Error("processed_at should be set after MarkDone")
	}
}

func TestBotInteractionRepo_MarkFailed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBotInteractionRepository(db)
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"user_id": "u001"})
	if err := repo.Enqueue(ctx, queue.BotInteraction{
		EventType: queue.BotEventCardAction,
		UserID:    "u001",
		Payload:   payload,
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	items, err := repo.DequeueBatch(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("DequeueBatch: %v, items=%d", err, len(items))
	}

	if err := repo.MarkFailed(ctx, items[0].ID, "feishu api timeout"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	var status, errMsg string
	err = db.QueryRowContext(ctx,
		`SELECT status, COALESCE(error_msg,'') FROM bot_interactions WHERE id = $1`, items[0].ID,
	).Scan(&status, &errMsg)
	if err != nil {
		t.Fatalf("query after MarkFailed: %v", err)
	}
	if status != queue.StatusFailed {
		t.Errorf("status: want %q, got %q", queue.StatusFailed, status)
	}
	if errMsg != "feishu api timeout" {
		t.Errorf("error_msg: want %q, got %q", "feishu api timeout", errMsg)
	}
}
