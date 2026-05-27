ALTER TABLE users ADD COLUMN IF NOT EXISTS lark_open_id TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_users_lark_open_id ON users (lark_open_id) WHERE lark_open_id IS NOT NULL;
