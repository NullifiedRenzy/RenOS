CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    command TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    exit_code INT,
    shell TEXT NOT NULL DEFAULT 'auto',
    timeout_ms BIGINT NOT NULL DEFAULT 300000,
    max_output_bytes BIGINT NOT NULL DEFAULT 1048576,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_executions_device_id ON executions(device_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_requested_at ON executions(requested_at DESC);
CREATE INDEX idx_executions_device_requested ON executions(device_id, requested_at DESC);

CREATE TABLE execution_output (
    id BIGSERIAL PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    stream TEXT NOT NULL DEFAULT 'stdout',
    content TEXT NOT NULL,
    seq INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_execution_output_exec ON execution_output(execution_id, seq);
