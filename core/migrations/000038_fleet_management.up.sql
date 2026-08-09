-- Device tags
CREATE TABLE device_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    tag_key TEXT NOT NULL,
    tag_value TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, tag_key)
);

CREATE INDEX idx_device_tags_key ON device_tags(tag_key);
CREATE INDEX idx_device_tags_device ON device_tags(device_id);

-- Dynamic groups (query-based)
ALTER TABLE groups ADD COLUMN is_dynamic BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN query JSONB NOT NULL DEFAULT '{}';
ALTER TABLE groups ADD COLUMN last_evaluated_at TIMESTAMPTZ;

-- Device inventory
CREATE TABLE device_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    cpu_model TEXT NOT NULL DEFAULT '',
    cpu_cores INTEGER NOT NULL DEFAULT 0,
    cpu_threads INTEGER NOT NULL DEFAULT 0,
    cpu_freq_mhz NUMERIC NOT NULL DEFAULT 0,
    gpu_model TEXT NOT NULL DEFAULT '',
    gpu_memory_mb INTEGER NOT NULL DEFAULT 0,
    ram_total_mb INTEGER NOT NULL DEFAULT 0,
    ram_used_mb INTEGER NOT NULL DEFAULT 0,
    storage_total_gb NUMERIC NOT NULL DEFAULT 0,
    storage_used_gb NUMERIC NOT NULL DEFAULT 0,
    storage_type TEXT NOT NULL DEFAULT '',
    network_interfaces JSONB NOT NULL DEFAULT '[]',
    kernel_version TEXT NOT NULL DEFAULT '',
    os_family TEXT NOT NULL DEFAULT '',
    os_version TEXT NOT NULL DEFAULT '',
    installed_packages JSONB NOT NULL DEFAULT '[]',
    installed_software JSONB NOT NULL DEFAULT '[]',
    uptime_seconds BIGINT NOT NULL DEFAULT 0,
    last_collected_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id)
);

CREATE INDEX idx_device_inventory_device ON device_inventory(device_id);

-- Compliance checks
CREATE TABLE compliance_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    check_type TEXT NOT NULL DEFAULT 'manual' CHECK (check_type IN ('manual', 'os_version', 'agent_version', 'backup', 'plugin', 'custom')),
    severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('info', 'warning', 'critical')),
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE compliance_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id UUID NOT NULL REFERENCES compliance_checks(id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pass', 'fail', 'skip', 'pending')),
    message TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '{}',
    last_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(check_id, device_id)
);

CREATE INDEX idx_compliance_results_device ON compliance_results(device_id);
CREATE INDEX idx_compliance_results_check ON compliance_results(check_id);
CREATE INDEX idx_compliance_results_status ON compliance_results(status);

-- Bulk operations
CREATE TABLE bulk_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT 'devices',
    target_ids JSONB NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    total_count INTEGER NOT NULL DEFAULT 0,
    completed_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    error_message TEXT NOT NULL DEFAULT '',
    created_by UUID,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bulk_operations_status ON bulk_operations(status);
