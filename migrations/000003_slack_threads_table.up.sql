CREATE TABLE IF NOT EXISTS slack_alert_threads (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    check_uuid  TEXT NOT NULL,
    channel_id  TEXT NOT NULL,
    thread_ts   TEXT NOT NULL,
    parent_ts   TEXT NOT NULL,
    is_resolved INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    resolved_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_slack_threads_unresolved
    ON slack_alert_threads (check_uuid, is_resolved) WHERE is_resolved = 0;
