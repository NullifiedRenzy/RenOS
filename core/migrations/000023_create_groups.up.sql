CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_groups_created_by ON groups(created_by);

CREATE TABLE device_groups (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, device_id)
);

CREATE INDEX idx_device_groups_device ON device_groups(device_id);

CREATE TABLE group_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID REFERENCES groups(id) ON DELETE SET NULL,
    action_type TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_value TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_devices INT NOT NULL DEFAULT 0,
    completed_devices INT NOT NULL DEFAULT 0,
    failed_devices INT NOT NULL DEFAULT 0,
    result_summary JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_group_actions_group ON group_actions(group_id);
CREATE INDEX idx_group_actions_status ON group_actions(status);

CREATE TABLE group_action_commands (
    group_action_id UUID NOT NULL REFERENCES group_actions(id) ON DELETE CASCADE,
    command_id UUID NOT NULL REFERENCES commands(id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    PRIMARY KEY (group_action_id, command_id)
);

CREATE INDEX idx_group_action_commands_action ON group_action_commands(group_action_id);
CREATE INDEX idx_group_action_commands_command ON group_action_commands(command_id);
