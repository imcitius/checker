-- Remove retry configuration from check_definitions
ALTER TABLE check_definitions DROP COLUMN IF EXISTS retry_count;
ALTER TABLE check_definitions DROP COLUMN IF EXISTS retry_interval;
