ALTER TABLE container_actions DROP CONSTRAINT IF EXISTS container_actions_action_check;

ALTER TABLE container_actions ADD CONSTRAINT container_actions_action_check
    CHECK (action IN ('start', 'stop', 'restart'));
