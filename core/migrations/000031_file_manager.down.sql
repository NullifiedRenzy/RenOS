DROP TABLE IF EXISTS file_permissions;
DROP TABLE IF EXISTS file_transfer_chunks;
DROP TABLE IF EXISTS file_transfers;

ALTER TABLE file_actions DROP CONSTRAINT IF EXISTS file_actions_action_type_check;
ALTER TABLE file_actions ADD CONSTRAINT file_actions_action_type_check
    CHECK (action_type IN ('list_directory', 'read_file', 'search_files'));

ALTER TABLE file_actions DROP COLUMN IF EXISTS target_path;
ALTER TABLE file_actions DROP COLUMN IF EXISTS metadata;
