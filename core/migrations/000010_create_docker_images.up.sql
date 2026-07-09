CREATE TABLE docker_images (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id  UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    image_id   TEXT NOT NULL,
    repository TEXT NOT NULL DEFAULT '',
    tag        TEXT NOT NULL DEFAULT '',
    size       TEXT NOT NULL DEFAULT '',
    created    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, image_id)
);

CREATE INDEX idx_docker_images_device_id ON docker_images(device_id);

CREATE OR REPLACE FUNCTION update_docker_images_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_docker_images_updated_at ON docker_images;
CREATE TRIGGER trg_docker_images_updated_at
    BEFORE UPDATE ON docker_images
    FOR EACH ROW
    EXECUTE FUNCTION update_docker_images_updated_at();
