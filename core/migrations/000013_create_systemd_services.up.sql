CREATE TABLE systemd_services (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    unit_name   TEXT NOT NULL,
    load_state  TEXT NOT NULL DEFAULT '',
    active_state TEXT NOT NULL DEFAULT '',
    sub_state   TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    enabled     TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, unit_name)
);

CREATE INDEX idx_systemd_services_device_id ON systemd_services(device_id);

CREATE OR REPLACE FUNCTION update_systemd_services_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_systemd_services_updated_at ON systemd_services;
CREATE TRIGGER trg_systemd_services_updated_at
    BEFORE UPDATE ON systemd_services
    FOR EACH ROW
    EXECUTE FUNCTION update_systemd_services_updated_at();
