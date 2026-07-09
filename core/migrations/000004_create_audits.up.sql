CREATE TABLE audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    signature TEXT,
    prev_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audits_timestamp ON audits(timestamp);
CREATE INDEX idx_audits_actor ON audits(actor_type, actor_id);
CREATE INDEX idx_audits_target ON audits(target_type, target_id);
CREATE INDEX idx_audits_action ON audits(action);
