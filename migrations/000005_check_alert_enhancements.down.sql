-- Remove alert severity levels and multi-channel support from check_definitions
ALTER TABLE check_definitions DROP COLUMN IF EXISTS severity;
ALTER TABLE check_definitions DROP COLUMN IF EXISTS alert_channels;
ALTER TABLE check_definitions DROP COLUMN IF EXISTS re_alert_interval;
