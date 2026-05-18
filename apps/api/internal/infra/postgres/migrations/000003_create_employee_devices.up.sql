CREATE TABLE IF NOT EXISTS employee_devices (
    id               UUID PRIMARY KEY,
    employee_id      UUID,
    external_id      TEXT UNIQUE NOT NULL,
    device_name      TEXT,
    platform         TEXT,
    serial_number    TEXT,
    os_version       TEXT,
    status           TEXT,
    trust_level      TEXT,
    last_online_time BIGINT,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_employee_devices_external_id ON employee_devices (external_id);
CREATE INDEX IF NOT EXISTS idx_employee_devices_employee_id ON employee_devices (employee_id);
