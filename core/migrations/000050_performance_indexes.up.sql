-- Performance indexes (M64) — derived from query analysis:
-- 1. Commands are listed per device ordered by requested_at; existing
--    (device_id, status) index cannot serve the ORDER BY.
-- 2. Jobs are filtered by created_by (list-jobs-by-creator path).
-- 3. System logs are commonly filtered by (level, source) together; the
--    existing single-column indexes require a bitmap combine.

CREATE INDEX IF NOT EXISTS idx_commands_device_requested ON commands(device_id, requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_jobs_created_by ON jobs(created_by);

CREATE INDEX IF NOT EXISTS idx_system_logs_level_source ON system_logs(level, source, created_at DESC);
