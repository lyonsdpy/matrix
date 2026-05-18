CREATE TABLE IF NOT EXISTS violation_records (
    id            UUID PRIMARY KEY,
    employee_id   UUID,
    employee_name TEXT,
    office        TEXT NOT NULL,
    ip            TEXT,
    mac           TEXT,
    status        TEXT NOT NULL,
    detected_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_violation_records_office ON violation_records (office);
CREATE INDEX IF NOT EXISTS idx_violation_records_detected_at ON violation_records (detected_at);
