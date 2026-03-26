CREATE TABLE IF NOT EXISTS discord_alert_threads (
    id          SERIAL PRIMARY KEY,
    check_uuid  TEXT NOT NULL,
    channel_id  TEXT NOT NULL,
    message_id  TEXT NOT NULL,
    thread_id   TEXT NOT NULL,
    is_resolved INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_discord_threads_unresolved
    ON discord_alert_threads (check_uuid, is_resolved) WHERE is_resolved = 0;
