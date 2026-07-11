CREATE TABLE network_interfaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    mac_address TEXT NOT NULL DEFAULT '',
    ipv4 TEXT[] DEFAULT '{}',
    ipv6 TEXT[] DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'down',
    speed BIGINT NOT NULL DEFAULT 0,
    interface_type TEXT NOT NULL DEFAULT '',
    is_default_gateway BOOLEAN NOT NULL DEFAULT FALSE,
    dns_servers TEXT[] DEFAULT '{}',
    mtu INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_network_interfaces_device ON network_interfaces(device_id);
