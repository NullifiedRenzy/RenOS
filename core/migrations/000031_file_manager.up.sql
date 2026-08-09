-- Extend file_actions with new operation types for M33
ALTER TABLE file_actions DROP CONSTRAINT IF EXISTS file_actions_action_type_check;
ALTER TABLE file_actions ADD CONSTRAINT file_actions_action_type_check
    CHECK (action_type IN (
        'list_directory', 'read_file', 'search_files',
        'create_directory', 'delete_path', 'rename_path', 'move_path', 'copy_path',
        'set_permissions', 'upload_file', 'download_file', 'preview_file'
    ));

-- Add target_path for rename/move/copy operations
ALTER TABLE file_actions ADD COLUMN target_path TEXT DEFAULT '';

-- Add metadata for additional context (permissions string, etc.)
ALTER TABLE file_actions ADD COLUMN metadata JSONB DEFAULT '{}';

-- Track chunked file transfers (uploads/downloads via Core)
CREATE TABLE file_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    action_id UUID REFERENCES file_actions(id) ON DELETE SET NULL,
    transfer_type TEXT NOT NULL CHECK (transfer_type IN ('upload', 'download')),
    filename TEXT NOT NULL,
    remote_path TEXT NOT NULL,
    total_size BIGINT NOT NULL DEFAULT 0,
    chunk_size INTEGER NOT NULL DEFAULT 1048576,
    total_chunks INTEGER NOT NULL DEFAULT 0,
    uploaded_chunks INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'uploading', 'downloading', 'assembling', 'completed', 'failed', 'cancelled')),
    mime_type TEXT DEFAULT '',
    error TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_file_transfers_device_status ON file_transfers(device_id, status);
CREATE INDEX idx_file_transfers_user ON file_transfers(user_id);

-- Store uploaded chunks on Core (agent downloads from here, hub uploads to here)
CREATE TABLE file_transfer_chunks (
    id SERIAL PRIMARY KEY,
    transfer_id UUID NOT NULL REFERENCES file_transfers(id) ON DELETE CASCADE,
    chunk_number INTEGER NOT NULL,
    chunk_data BYTEA NOT NULL,
    checksum TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (transfer_id, chunk_number)
);

CREATE INDEX idx_file_transfer_chunks_transfer ON file_transfer_chunks(transfer_id);

-- Per-device file access RBAC (optional; if empty, all authenticated users can access)
CREATE TABLE file_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    can_browse BOOLEAN NOT NULL DEFAULT true,
    can_modify BOOLEAN NOT NULL DEFAULT false,
    can_upload BOOLEAN NOT NULL DEFAULT false,
    can_download BOOLEAN NOT NULL DEFAULT false,
    can_delete BOOLEAN NOT NULL DEFAULT false,
    allowed_roots JSONB DEFAULT '["/"]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, user_id)
);

CREATE INDEX idx_file_permissions_device ON file_permissions(device_id);
