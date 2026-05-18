CREATE TABLE IF NOT EXISTS employees (
    id              UUID PRIMARY KEY,
    external_id     TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    email           TEXT,
    mobile          TEXT,
    status          INT NOT NULL DEFAULT 1,
    department_ids  JSONB NOT NULL DEFAULT '[]',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_employees_external_id ON employees (external_id);
CREATE INDEX IF NOT EXISTS idx_employees_name ON employees (name);
