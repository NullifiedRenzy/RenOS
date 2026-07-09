CREATE TABLE IF NOT EXISTS containers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    image       TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT '',
    state       TEXT NOT NULL DEFAULT '',
    ports       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, container_id)
);

CREATE INDEX idx_containers_device_id ON containers(device_id);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_containers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_containers_updated_at ON containers;
CREATE TRIGGER trg_containers_updated_at
    BEFORE UPDATE ON containers
    FOR EACH ROW
    EXECUTE FUNCTION update_containers_updated_at();
