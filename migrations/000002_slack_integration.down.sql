-- Migration 000002 (down): Remove Slack integration support

BEGIN;

DROP TABLE IF EXISTS alert_silences;

ALTER TABLE check_definitions
    DROP COLUMN IF EXISTS slack_thread_ts,
    DROP COLUMN IF EXISTS slack_channel_id;

COMMIT;
