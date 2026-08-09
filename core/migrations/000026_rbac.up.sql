-- Migrate old 'user' role to 'operator'
UPDATE users SET role = 'operator' WHERE role = 'user';

-- Ensure all remaining roles are valid, default to 'viewer' for anything else
UPDATE users SET role = 'viewer' WHERE role NOT IN ('admin', 'operator', 'viewer');

ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'operator', 'viewer'));

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role TEXT NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role, resource, action)
);

INSERT INTO permissions (role, resource, action) VALUES
    ('admin', '*', '*'),
    ('operator', 'device', 'read'),
    ('operator', 'device', 'command'),
    ('operator', 'device', 'container_action'),
    ('operator', 'device', 'service_action'),
    ('operator', 'device', 'file_action'),
    ('operator', 'device', 'update'),
    ('operator', 'device', 'plugin_action'),
    ('operator', 'device', 'vm_action'),
    ('operator', 'group', 'read'),
    ('operator', 'group', 'write'),
    ('operator', 'group', 'delete'),
    ('operator', 'group', 'command'),
    ('operator', 'job', 'read'),
    ('operator', 'job', 'cancel'),
    ('operator', 'job', 'retry'),
    ('operator', 'schedule', 'read'),
    ('operator', 'schedule', 'write'),
    ('operator', 'schedule', 'delete'),
    ('operator', 'schedule', 'enable'),
    ('operator', 'schedule', 'disable'),
    ('operator', 'ai', 'read'),
    ('operator', 'ai', 'chat'),
    ('viewer', 'device', 'read'),
    ('viewer', 'group', 'read'),
    ('viewer', 'job', 'read'),
    ('viewer', 'schedule', 'read'),
    ('viewer', 'ai', 'read');
