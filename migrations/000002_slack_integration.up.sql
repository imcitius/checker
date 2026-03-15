-- Migration 000002: Slack integration support
-- Adds Slack threading columns to check_definitions and creates alert_silences table.

-- Add Slack threading columns to check_definitions
ALTER TABLE check_definitions ADD COLUMN slack_thread_ts TEXT;
ALTER TABLE check_definitions ADD COLUMN slack_channel_id TEXT;

-- Alert silences table: suppresses alerts by scope (check or project)
CREATE TABLE IF NOT EXISTS alert_silences (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    scope       TEXT NOT NULL,            -- 'check' or 'project'
    target      TEXT NOT NULL DEFAULT '', -- check UUID or project name
    silenced_by TEXT NOT NULL DEFAULT '', -- Slack user ID
    silenced_at DATETIME NOT NULL DEFAULT (datetime('now')),
    expires_at  DATETIME,                -- NULL = never expires
    reason      TEXT NOT NULL DEFAULT '',
    active      INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_alert_silences_active_scope_target
    ON alert_silences (active, scope, target)
    WHERE active = 1;
