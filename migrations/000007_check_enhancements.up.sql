-- Add maintenance window support to check_definitions
ALTER TABLE check_definitions ADD COLUMN IF NOT EXISTS maintenance_until TIMESTAMPTZ;
