CREATE TABLE update_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    package_name TEXT NOT NULL,
    current_version TEXT NOT NULL DEFAULT '',
    available_version TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_update_status_device ON update_status(device_id);
