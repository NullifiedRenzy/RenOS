CREATE TABLE docker_networks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id  UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    network_id TEXT NOT NULL,
    name       TEXT NOT NULL,
    driver     TEXT NOT NULL DEFAULT '',
    scope      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, network_id)
);

CREATE INDEX idx_docker_networks_device_id ON docker_networks(device_id);

CREATE OR REPLACE FUNCTION update_docker_networks_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_docker_networks_updated_at ON docker_networks;
CREATE TRIGGER trg_docker_networks_updated_at
    BEFORE UPDATE ON docker_networks
    FOR EACH ROW
    EXECUTE FUNCTION update_docker_networks_updated_at();
