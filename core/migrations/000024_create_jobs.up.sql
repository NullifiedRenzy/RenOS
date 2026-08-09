CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    total_devices INT NOT NULL DEFAULT 0,
    completed_devices INT NOT NULL DEFAULT 0,
    failed_devices INT NOT NULL DEFAULT 0,
    retries INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_action_id UUID REFERENCES group_actions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_group_action ON jobs(group_action_id);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
