CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL CHECK (type IN ('command', 'container_action', 'service_action', 'update', 'schedule', 'job', '*')),
    conditions JSONB NOT NULL DEFAULT '{}',
    action TEXT NOT NULL CHECK (action IN ('allow', 'deny', 'warn')),
    priority INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policies_type ON policies(type);
CREATE INDEX idx_policies_enabled ON policies(enabled);

INSERT INTO policies (name, description, type, conditions, action, priority) VALUES
    ('Max devices per action', 'Limit any action to at most 20 devices', '*',
     '{"max_devices": 20}', 'deny', 100),
    ('Min cron interval 5 min', 'Prevent schedules from running more often than every 5 minutes', 'schedule',
     '{"restrict_cron_frequency": 5}', 'deny', 100),
    ('Restrict dangerous commands', 'Block dangerous system commands', 'command',
     '{"allowed_commands": ["apt update", "apt upgrade", "apt-get update", "apt-get upgrade", "systemctl restart", "systemctl start", "systemctl stop", "systemctl status", "docker ps", "docker stats", "journalctl", "df -h", "free -m", "uptime", "uname -a"]}', 'deny', 90),
    ('Warn on large batch', 'Warn when targeting more than 10 devices', '*',
     '{"max_devices": 10, "requires_confirmation": true}', 'warn', 80);
