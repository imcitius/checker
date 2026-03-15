-- Remove alert severity levels and multi-channel support from check_definitions
ALTER TABLE check_definitions DROP COLUMN severity;
ALTER TABLE check_definitions DROP COLUMN alert_channels;
ALTER TABLE check_definitions DROP COLUMN re_alert_interval;
