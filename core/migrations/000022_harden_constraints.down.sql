-- Revert constraint changes
ALTER TABLE ai_conversations DROP CONSTRAINT IF EXISTS fk_ai_conversations_user;

-- We do NOT revert FK changes here as they are structural improvements
-- that should not be rolled back in production. The down migration is
-- provided for development rollback only.

ALTER TABLE commands
    ALTER COLUMN device_id DROP NOT NULL,
    ALTER COLUMN user_id DROP NOT NULL;
-- Note: This is intentionally minimal. Full reversal would require
-- recreating dropped indexes and unique constraints.
