CREATE TABLE IF NOT EXISTS user_bind_items (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    acg_device TEXT        NOT NULL,
    user_path  TEXT        NOT NULL,
    bind_type  TEXT        NOT NULL,
    address    TEXT        NOT NULL,
    is_exclude BOOLEAN     NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (acg_device, user_path, bind_type, address, is_exclude)
);

CREATE INDEX IF NOT EXISTS idx_user_bind_items_user_path ON user_bind_items (user_path);

CREATE TABLE IF NOT EXISTS observed_mac_bindings (
    id           UUID        NOT NULL DEFAULT gen_random_uuid(),
    office       TEXT        NOT NULL,
    ip           TEXT        NOT NULL,
    mac          TEXT        NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (office, ip, mac)
);

CREATE INDEX IF NOT EXISTS idx_observed_mac_bindings_ip  ON observed_mac_bindings (ip);
CREATE INDEX IF NOT EXISTS idx_observed_mac_bindings_mac ON observed_mac_bindings (mac);
