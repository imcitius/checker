CREATE TABLE IF NOT EXISTS slack_alert_threads (
    id          SERIAL PRIMARY KEY,
    check_uuid  TEXT NOT NULL,
    channel_id  TEXT NOT NULL,
    thread_ts   TEXT NOT NULL,
    parent_ts   TEXT NOT NULL,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_slack_threads_unresolved
    ON slack_alert_threads (check_uuid, is_resolved) WHERE is_resolved = FALSE;
