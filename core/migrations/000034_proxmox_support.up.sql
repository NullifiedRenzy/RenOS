-- Proxmox connections: stores credentials for Proxmox VE API access
CREATE TABLE proxmox_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 8006,
    auth_type TEXT NOT NULL CHECK (auth_type IN ('token', 'password')),
    token_id TEXT NOT NULL DEFAULT '',
    token_secret TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    password TEXT NOT NULL DEFAULT '',
    verify_ssl BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'disconnected' CHECK (status IN ('connected', 'disconnected', 'error')),
    error_message TEXT NOT NULL DEFAULT '',
    last_connected_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_proxmox_connections_device ON proxmox_connections(device_id);

-- Proxmox nodes: physical hosts in the cluster
CREATE TABLE proxmox_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES proxmox_connections(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline')),
    cpu REAL NOT NULL DEFAULT 0,
    max_cpu INTEGER NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    max_memory BIGINT NOT NULL DEFAULT 0,
    disk BIGINT NOT NULL DEFAULT 0,
    max_disk BIGINT NOT NULL DEFAULT 0,
    uptime BIGINT NOT NULL DEFAULT 0,
    ssl_fingerprint TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, node_name)
);

CREATE INDEX idx_proxmox_nodes_connection ON proxmox_nodes(connection_id);

-- Proxmox VMs (QEMU)
CREATE TABLE proxmox_vms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES proxmox_connections(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    vm_id INTEGER NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('running', 'stopped', 'paused', 'suspended')),
    cpu INTEGER NOT NULL DEFAULT 0,
    cores INTEGER NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    disk BIGINT NOT NULL DEFAULT 0,
    ip_address TEXT NOT NULL DEFAULT '',
    os_type TEXT NOT NULL DEFAULT '',
    template BOOLEAN NOT NULL DEFAULT false,
    ha_state TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, node_name, vm_id)
);

CREATE INDEX idx_proxmox_vms_connection ON proxmox_vms(connection_id);

-- Proxmox LXCs (OpenVZ containers)
CREATE TABLE proxmox_lxcs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES proxmox_connections(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    vm_id INTEGER NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('running', 'stopped')),
    cpu INTEGER NOT NULL DEFAULT 0,
    cores INTEGER NOT NULL DEFAULT 0,
    memory BIGINT NOT NULL DEFAULT 0,
    disk BIGINT NOT NULL DEFAULT 0,
    ip_address TEXT NOT NULL DEFAULT '',
    os_type TEXT NOT NULL DEFAULT '',
    template BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, node_name, vm_id)
);

CREATE INDEX idx_proxmox_lxcs_connection ON proxmox_lxcs(connection_id);

-- Proxmox storage
CREATE TABLE proxmox_storage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES proxmox_connections(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    storage_id TEXT NOT NULL,
    storage_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    total BIGINT NOT NULL DEFAULT 0,
    used BIGINT NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, node_name, storage_id)
);

CREATE INDEX idx_proxmox_storage_connection ON proxmox_storage(connection_id);

-- Proxmox networks
CREATE TABLE proxmox_networks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES proxmox_connections(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    interface TEXT NOT NULL,
    iface_type TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    netmask TEXT NOT NULL DEFAULT '',
    gateway TEXT NOT NULL DEFAULT '',
    bridge TEXT NOT NULL DEFAULT '',
    vlan INTEGER,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, node_name, interface)
);

CREATE INDEX idx_proxmox_networks_connection ON proxmox_networks(connection_id);

-- Proxmox snapshots
CREATE TABLE proxmox_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES proxmox_connections(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    vm_id INTEGER NOT NULL,
    snapshot_id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    parent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, node_name, vm_id, snapshot_id)
);

CREATE INDEX idx_proxmox_snapshots_connection ON proxmox_snapshots(connection_id);

-- Proxmox tasks
CREATE TABLE proxmox_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES proxmox_connections(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    upid TEXT NOT NULL,
    task_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    user_name TEXT NOT NULL DEFAULT '',
    start_time BIGINT NOT NULL DEFAULT 0,
    end_time BIGINT NOT NULL DEFAULT 0,
    exit_status TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, upid)
);

CREATE INDEX idx_proxmox_tasks_connection ON proxmox_tasks(connection_id);
CREATE INDEX idx_proxmox_tasks_status ON proxmox_tasks(connection_id, status);
