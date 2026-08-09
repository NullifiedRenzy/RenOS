-- Process snapshots: agent streams full process list periodically via NATS
CREATE TABLE process_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    pid INTEGER NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    executable TEXT NOT NULL DEFAULT '',
    command_line TEXT NOT NULL DEFAULT '',
    cpu_usage REAL NOT NULL DEFAULT 0,
    memory_bytes BIGINT NOT NULL DEFAULT 0,
    memory_usage REAL NOT NULL DEFAULT 0,
    threads INTEGER NOT NULL DEFAULT 0,
    open_handles INTEGER NOT NULL DEFAULT 0,
    parent_pid INTEGER NOT NULL DEFAULT 0,
    owner TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    start_time BIGINT NOT NULL DEFAULT 0,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_process_snapshots_device_pid ON process_snapshots(device_id, pid);
CREATE INDEX idx_process_snapshots_device_collected ON process_snapshots(device_id, collected_at DESC);
CREATE INDEX idx_process_snapshots_device_name ON process_snapshots(device_id, name);

-- Process actions: kill, terminate, suspend, resume, restart
CREATE TABLE process_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    pid INTEGER NOT NULL,
    action_type TEXT NOT NULL CHECK (action_type IN ('kill', 'terminate', 'suspend', 'resume', 'restart')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    exit_code INTEGER,
    output TEXT DEFAULT '',
    error TEXT DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_process_actions_device_status ON process_actions(device_id, status);
CREATE INDEX idx_process_actions_device_pid ON process_actions(device_id, pid);

-- Process history: CPU/memory time series for graphs (keep 1 hour, 5s resolution = 720 rows max)
CREATE TABLE process_history (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    total_cpu REAL NOT NULL DEFAULT 0,
    total_memory_bytes BIGINT NOT NULL DEFAULT 0,
    total_memory_usage REAL NOT NULL DEFAULT 0,
    process_count INTEGER NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_process_history_device_time ON process_history(device_id, recorded_at DESC);
