CREATE TABLE virtual_machines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    domain_name TEXT NOT NULL,
    uuid TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT '',
    vcpu_max INTEGER NOT NULL DEFAULT 0,
    vcpu_current INTEGER NOT NULL DEFAULT 0,
    memory_max_bytes BIGINT NOT NULL DEFAULT 0,
    memory_current_bytes BIGINT NOT NULL DEFAULT 0,
    os_type TEXT NOT NULL DEFAULT '',
    autostart BOOLEAN NOT NULL DEFAULT FALSE,
    persistent BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_virtual_machines_device ON virtual_machines(device_id);
