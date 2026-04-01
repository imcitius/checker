-- Add maintenance window support to check_definitions
ALTER TABLE check_definitions ADD COLUMN maintenance_until TIMESTAMPTZ;
