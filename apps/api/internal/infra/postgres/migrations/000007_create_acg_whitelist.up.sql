CREATE TABLE IF NOT EXISTS acg_whitelist_entries (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    acg_device TEXT        NOT NULL,
    enable     BOOLEAN     NOT NULL DEFAULT true,
    name       TEXT        NOT NULL,
    "desc"     TEXT        NOT NULL DEFAULT '',
    addrs      TEXT[]      NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (acg_device, name)
);
