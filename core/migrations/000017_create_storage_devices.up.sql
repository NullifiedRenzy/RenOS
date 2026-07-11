CREATE TABLE storage_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    device_type TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    mount_point TEXT NOT NULL DEFAULT '',
    filesystem TEXT NOT NULL DEFAULT '',
    uuid TEXT NOT NULL DEFAULT '',
    is_removable BOOLEAN NOT NULL DEFAULT FALSE,
    is_readonly BOOLEAN NOT NULL DEFAULT FALSE,
    used_bytes BIGINT NOT NULL DEFAULT 0,
    available_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_storage_devices_device ON storage_devices(device_id);
