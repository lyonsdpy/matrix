CREATE TABLE IF NOT EXISTS online_sessions (
    id                       BIGSERIAL PRIMARY KEY,
    employee_id              UUID,
    employee_name            TEXT,
    device_id                UUID,
    office                   TEXT NOT NULL,
    ip                       TEXT,
    mac                      TEXT,
    login_time               TEXT,
    online_time              TEXT,
    platform                 TEXT,
    system                   TEXT,
    device                   TEXT,
    compliance_check_result  TEXT,
    fetched_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_online_sessions_office ON online_sessions (office);
