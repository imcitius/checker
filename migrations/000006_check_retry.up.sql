-- Add retry configuration to check_definitions
ALTER TABLE check_definitions ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;
ALTER TABLE check_definitions ADD COLUMN IF NOT EXISTS retry_interval VARCHAR(20) DEFAULT '';
