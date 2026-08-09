DROP TABLE IF EXISTS bulk_operations;
DROP TABLE IF EXISTS compliance_results;
DROP TABLE IF EXISTS compliance_checks;
DROP TABLE IF EXISTS device_inventory;
DROP TABLE IF EXISTS device_tags;
ALTER TABLE groups DROP COLUMN IF EXISTS is_dynamic;
ALTER TABLE groups DROP COLUMN IF EXISTS query;
ALTER TABLE groups DROP COLUMN IF EXISTS last_evaluated_at;
