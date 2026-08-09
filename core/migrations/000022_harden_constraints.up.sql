-- Add missing FK on ai_conversations
ALTER TABLE ai_conversations
    ADD CONSTRAINT fk_ai_conversations_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add ON DELETE CASCADE to existing FKs by dropping and recreating
ALTER TABLE commands DROP CONSTRAINT IF EXISTS commands_device_id_fkey;
ALTER TABLE commands
    ADD CONSTRAINT commands_device_id_fkey
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE;

ALTER TABLE commands DROP CONSTRAINT IF EXISTS commands_user_id_fkey;
ALTER TABLE commands
    ADD CONSTRAINT commands_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE container_actions DROP CONSTRAINT IF EXISTS container_actions_user_id_fkey;
ALTER TABLE container_actions
    ADD CONSTRAINT container_actions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE service_actions DROP CONSTRAINT IF EXISTS service_actions_user_id_fkey;
ALTER TABLE service_actions
    ADD CONSTRAINT service_actions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE file_actions DROP CONSTRAINT IF EXISTS file_actions_user_id_fkey;
ALTER TABLE file_actions
    ADD CONSTRAINT file_actions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add unique constraints to prevent duplicates
ALTER TABLE update_status
    ADD CONSTRAINT uq_update_status_device_package
    UNIQUE (device_id, package_name);

ALTER TABLE plugins
    ADD CONSTRAINT uq_plugins_device_name
    UNIQUE (device_id, name);

-- Drop redundant indexes (covered by unique/composite indexes)
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_sessions_token_hash;
DROP INDEX IF EXISTS idx_containers_device_id;
DROP INDEX IF EXISTS idx_docker_images_device_id;
DROP INDEX IF EXISTS idx_docker_volumes_device_id;
DROP INDEX IF EXISTS idx_docker_networks_device_id;
DROP INDEX IF EXISTS idx_systemd_services_device_id;

-- Add missing composite indexes for efficient query patterns
CREATE INDEX IF NOT EXISTS idx_service_actions_device_status
    ON service_actions(device_id, status);
