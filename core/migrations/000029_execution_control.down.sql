ALTER TABLE devices DROP COLUMN IF EXISTS device_state;
ALTER TABLE devices DROP COLUMN IF EXISTS failure_count;
ALTER TABLE devices DROP COLUMN IF EXISTS last_failure_at;

ALTER TABLE jobs DROP COLUMN IF EXISTS max_job_duration_ms;
ALTER TABLE jobs DROP COLUMN IF EXISTS max_device_execution_ms;
ALTER TABLE jobs DROP COLUMN IF EXISTS stop_on_failure;
ALTER TABLE jobs DROP COLUMN IF EXISTS batch_size;
ALTER TABLE jobs DROP COLUMN IF EXISTS batch_delay_ms;
ALTER TABLE jobs DROP COLUMN IF EXISTS stopped_reason;
