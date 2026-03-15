-- Add retry configuration to check_definitions
ALTER TABLE check_definitions ADD COLUMN retry_count INTEGER DEFAULT 0;
ALTER TABLE check_definitions ADD COLUMN retry_interval TEXT DEFAULT '';
