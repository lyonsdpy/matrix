package pg_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"matrix/api/internal/infra/queue"
)

// compile-time interface check
var _ queue.BotInteractionRepository = (*botInteractionRepo)(nil)

// botInteractionRepo 实现 queue.BotInteractionRepository，使用 PostgreSQL 存储。
type botInteractionRepo struct {
	db *sqlx.DB
}

// NewBotInteractionRepository 创建 BotInteractionRepository 的 PostgreSQL 实现。
func NewBotInteractionRepository(db *sqlx.DB) queue.BotInteractionRepository {
	return &botInteractionRepo{db: db}
}

// Enqueue 写入一条 pending 状态的交互记录。
func (r *botInteractionRepo) Enqueue(ctx context.Context, item queue.BotInteraction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO bot_interactions (event_type, user_id, payload, status)
		 VALUES ($1, $2, $3, $4)`,
		item.EventType, item.UserID, item.Payload, queue.StatusPending,
	)
	if err != nil {
		return fmt.Errorf("bot_interaction_repo: enqueue: %w", err)
	}
	return nil
}

// DequeueBatch 在单个事务内原子获取最多 limit 条 pending 记录并更新为 processing。
// 使用 SELECT FOR UPDATE SKIP LOCKED 保证并发消费安全，不重复处理同一条记录。
func (r *botInteractionRepo) DequeueBatch(ctx context.Context, limit int) ([]queue.BotInteraction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("bot_interaction_repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryxContext(ctx,
		`SELECT id, event_type, user_id, payload, status, COALESCE(error_msg,'') AS error_msg, created_at, processed_at
		 FROM bot_interactions
		 WHERE status = $1
		 ORDER BY created_at
		 LIMIT $2
		 FOR UPDATE SKIP LOCKED`,
		queue.StatusPending, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("bot_interaction_repo: select for update: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []queue.BotInteraction
	for rows.Next() {
		var item queue.BotInteraction
		if err := rows.Scan(
			&item.ID, &item.EventType, &item.UserID, &item.Payload,
			&item.Status, &item.ErrorMsg, &item.CreatedAt, &item.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("bot_interaction_repo: scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bot_interaction_repo: rows: %w", err)
	}

	if len(items) == 0 {
		_ = tx.Commit()
		return nil, nil
	}

	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	// pgx 原生支持 []int64 参数与 ANY($N) 语法，避免 sqlx.In 扩展 IN 子句的兼容性问题。
	if _, err := tx.ExecContext(ctx,
		`UPDATE bot_interactions SET status = $1 WHERE id = ANY($2)`,
		queue.StatusProcessing, ids,
	); err != nil {
		return nil, fmt.Errorf("bot_interaction_repo: update to processing: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("bot_interaction_repo: commit: %w", err)
	}

	for i := range items {
		items[i].Status = queue.StatusProcessing
	}
	return items, nil
}

// MarkDone 将指定记录更新为 done，设置 processed_at 为当前时间。
func (r *botInteractionRepo) MarkDone(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE bot_interactions SET status = $1, processed_at = $2 WHERE id = $3`,
		queue.StatusDone, now, id,
	)
	if err != nil {
		return fmt.Errorf("bot_interaction_repo: mark done id=%d: %w", id, err)
	}
	return nil
}

// MarkFailed 将指定记录更新为 failed，记录错误信息。
func (r *botInteractionRepo) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE bot_interactions SET status = $1, error_msg = $2, processed_at = $3 WHERE id = $4`,
		queue.StatusFailed, errMsg, now, id,
	)
	if err != nil {
		return fmt.Errorf("bot_interaction_repo: mark failed id=%d: %w", id, err)
	}
	return nil
}
