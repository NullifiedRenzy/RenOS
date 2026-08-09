-- Extend container_actions with all Docker management actions
ALTER TABLE container_actions DROP CONSTRAINT IF EXISTS container_actions_action_check;

ALTER TABLE container_actions ADD CONSTRAINT container_actions_action_check
    CHECK (action IN (
        'start', 'stop', 'restart', 'logs', 'stats',
        'kill', 'remove', 'exec', 'inspect', 'pause', 'unpause', 'rename',
        'image_pull', 'image_remove', 'image_prune',
        'volume_remove', 'volume_prune',
        'network_disconnect', 'network_connect',
        'system_prune'
    ));

-- Compose stacks: tracks docker compose projects on each device
CREATE TABLE compose_stacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT '',
    config_files TEXT NOT NULL DEFAULT '',
    services TEXT NOT NULL DEFAULT '[]',
    working_dir TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, name)
);

CREATE INDEX idx_compose_stacks_device ON compose_stacks(device_id);

-- Container metrics: time-series CPU/memory/network for live dashboard
CREATE TABLE container_metrics (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    container_name TEXT NOT NULL DEFAULT '',
    cpu_percent REAL NOT NULL DEFAULT 0,
    memory_usage_bytes BIGINT NOT NULL DEFAULT 0,
    memory_limit_bytes BIGINT NOT NULL DEFAULT 0,
    memory_percent REAL NOT NULL DEFAULT 0,
    network_rx_bytes BIGINT NOT NULL DEFAULT 0,
    network_tx_bytes BIGINT NOT NULL DEFAULT 0,
    block_read_bytes BIGINT NOT NULL DEFAULT 0,
    block_write_bytes BIGINT NOT NULL DEFAULT 0,
    pids INTEGER NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_container_metrics_device_time ON container_metrics(device_id, recorded_at DESC);
CREATE INDEX idx_container_metrics_device_container ON container_metrics(device_id, container_id, recorded_at DESC);
