CREATE TABLE service_actions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id    UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id),
    unit_name    TEXT NOT NULL,
    action       TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    exit_code    INTEGER,
    output       TEXT NOT NULL DEFAULT '',
    error        TEXT NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_service_actions_device_id ON service_actions(device_id);
CREATE INDEX idx_service_actions_status ON service_actions(status);

CREATE OR REPLACE FUNCTION update_service_actions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_service_actions_updated_at ON service_actions;
CREATE TRIGGER trg_service_actions_updated_at
    BEFORE UPDATE ON service_actions
    FOR EACH ROW
    EXECUTE FUNCTION update_service_actions_updated_at();
