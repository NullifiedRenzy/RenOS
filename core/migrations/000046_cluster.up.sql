-- Cluster nodes
CREATE TABLE cluster_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id TEXT NOT NULL UNIQUE,
    hostname TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    port INTEGER NOT NULL DEFAULT 8443,
    role TEXT NOT NULL DEFAULT 'follower' CHECK (role IN ('leader', 'follower', 'candidate')),
    status TEXT NOT NULL DEFAULT 'joining' CHECK (status IN ('joining', 'active', 'draining', 'offline', 'maintenance')),
    metadata JSONB NOT NULL DEFAULT '{}',
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_cluster_nodes_node_id ON cluster_nodes(node_id);
CREATE INDEX idx_cluster_nodes_role ON cluster_nodes(role);
CREATE INDEX idx_cluster_nodes_status ON cluster_nodes(status);

-- Leader election log
CREATE TABLE cluster_elections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id TEXT NOT NULL,
    term INTEGER NOT NULL DEFAULT 1,
    votes_for INTEGER NOT NULL DEFAULT 0,
    votes_against INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'won', 'lost', 'expired')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cluster_elections_term ON cluster_elections(term);

-- Cluster task distribution
CREATE TABLE cluster_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type TEXT NOT NULL,
    task_data JSONB NOT NULL DEFAULT '{}',
    assigned_to TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'assigned', 'running', 'completed', 'failed')),
    priority INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    retry_count INTEGER NOT NULL DEFAULT 0,
    result JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cluster_tasks_status ON cluster_tasks(status);
CREATE INDEX idx_cluster_tasks_assigned ON cluster_tasks(assigned_to);
CREATE INDEX idx_cluster_tasks_type ON cluster_tasks(task_type);

-- Cluster state (shared key-value store)
CREATE TABLE cluster_state (
    key TEXT NOT NULL,
    value JSONB NOT NULL DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    owner TEXT NOT NULL DEFAULT '',
    ttl_seconds INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (key)
);

CREATE INDEX idx_cluster_state_owner ON cluster_state(owner);
CREATE INDEX idx_cluster_state_expires ON cluster_state(expires_at);

-- Cluster metrics (per-node)
CREATE TABLE cluster_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id TEXT NOT NULL,
    cpu_usage NUMERIC NOT NULL DEFAULT 0,
    memory_usage NUMERIC NOT NULL DEFAULT 0,
    disk_usage NUMERIC NOT NULL DEFAULT 0,
    active_connections INTEGER NOT NULL DEFAULT 0,
    task_count INTEGER NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cluster_metrics_node ON cluster_metrics(node_id);
CREATE INDEX idx_cluster_metrics_time ON cluster_metrics(recorded_at);
