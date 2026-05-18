CREATE TABLE IF NOT EXISTS departments (
    id            UUID PRIMARY KEY,
    external_id   TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    parent_id     UUID,
    leader_id     UUID,
    member_count  INT NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_departments_external_id ON departments (external_id);
