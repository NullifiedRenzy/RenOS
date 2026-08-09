-- Update history
CREATE TABLE update_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_from TEXT NOT NULL,
    version_to TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'stable' CHECK (channel IN ('stable', 'beta', 'nightly')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'downloading', 'migrating', 'applying', 'completed', 'failed', 'rolled_back')),
    components JSONB NOT NULL DEFAULT '["core","hub","agent"]',
    backup_id TEXT,
    error_message TEXT,
    started_by TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_update_history_status ON update_history(status);
CREATE INDEX idx_update_history_channel ON update_history(channel);
CREATE INDEX idx_update_history_started_at ON update_history(started_at);

-- Backups
CREATE TABLE update_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version TEXT NOT NULL,
    backup_type TEXT NOT NULL DEFAULT 'pre_update' CHECK (backup_type IN ('pre_update', 'manual', 'scheduled')),
    backup_path TEXT NOT NULL,
    database_dump_path TEXT NOT NULL DEFAULT '',
    config_backup_path TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'creating' CHECK (status IN ('creating', 'ready', 'expired', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_update_backups_version ON update_backups(version);
CREATE INDEX idx_update_backups_status ON update_backups(status);

-- Plugin compatibility matrix
CREATE TABLE plugin_compat_matrix (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_name TEXT NOT NULL,
    min_renos_version TEXT NOT NULL DEFAULT '0.1.0',
    max_renos_version TEXT NOT NULL DEFAULT '999.999.999',
    tested BOOLEAN NOT NULL DEFAULT false,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plugin_compat_name ON plugin_compat_matrix(plugin_name);
