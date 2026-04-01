-- Add alert severity levels and multi-channel support to check_definitions
ALTER TABLE check_definitions ADD COLUMN severity TEXT DEFAULT 'critical';
ALTER TABLE check_definitions ADD COLUMN alert_channels TEXT;
ALTER TABLE check_definitions ADD COLUMN re_alert_interval TEXT;
