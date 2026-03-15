-- Add alert severity levels and multi-channel support to check_definitions
ALTER TABLE check_definitions ADD COLUMN IF NOT EXISTS severity VARCHAR(20) DEFAULT 'critical';
ALTER TABLE check_definitions ADD COLUMN IF NOT EXISTS alert_channels TEXT;
ALTER TABLE check_definitions ADD COLUMN IF NOT EXISTS re_alert_interval VARCHAR(20);
