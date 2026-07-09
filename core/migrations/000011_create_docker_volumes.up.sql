CREATE TABLE docker_volumes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id  UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    driver     TEXT NOT NULL DEFAULT 'local',
    mountpoint TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, name)
);

CREATE INDEX idx_docker_volumes_device_id ON docker_volumes(device_id);

CREATE OR REPLACE FUNCTION update_docker_volumes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_docker_volumes_updated_at ON docker_volumes;
CREATE TRIGGER trg_docker_volumes_updated_at
    BEFORE UPDATE ON docker_volumes
    FOR EACH ROW
    EXECUTE FUNCTION update_docker_volumes_updated_at();
