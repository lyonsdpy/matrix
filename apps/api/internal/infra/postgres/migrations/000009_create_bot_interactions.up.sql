CREATE TABLE IF NOT EXISTS bot_interactions (
    id           BIGSERIAL   PRIMARY KEY,
    event_type   TEXT        NOT NULL,
    user_id      TEXT        NOT NULL,
    payload      JSONB       NOT NULL DEFAULT '{}',
    status       TEXT        NOT NULL DEFAULT 'pending',
    error_msg    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_bot_interactions_status_created
    ON bot_interactions (status, created_at);
