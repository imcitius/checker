-- Migration 000002 (down): Remove Slack integration support

DROP TABLE IF EXISTS alert_silences;

ALTER TABLE check_definitions DROP COLUMN slack_thread_ts;
ALTER TABLE check_definitions DROP COLUMN slack_channel_id;
