CREATE TABLE IF NOT EXISTS software_blacklist (
    id         UUID        PRIMARY KEY,
    name       TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS blacklist_hits (
    id               UUID        PRIMARY KEY,
    employee_id      TEXT        NOT NULL DEFAULT '',
    employee_name    TEXT        NOT NULL DEFAULT '',
    device_id        TEXT        NOT NULL DEFAULT '',
    device_name      TEXT        NOT NULL DEFAULT '',
    software_name    TEXT        NOT NULL DEFAULT '',
    software_version TEXT        NOT NULL DEFAULT '',
    detected_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_blacklist_hits_employee_id ON blacklist_hits (employee_id);
CREATE INDEX IF NOT EXISTS idx_blacklist_hits_detected_at ON blacklist_hits (detected_at);
