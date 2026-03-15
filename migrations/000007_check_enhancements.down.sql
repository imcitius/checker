-- Remove maintenance window support from check_definitions
ALTER TABLE check_definitions DROP COLUMN IF EXISTS maintenance_until;
