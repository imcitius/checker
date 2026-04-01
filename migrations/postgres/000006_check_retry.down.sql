-- Remove retry configuration from check_definitions
ALTER TABLE check_definitions DROP COLUMN retry_count;
ALTER TABLE check_definitions DROP COLUMN retry_interval;
