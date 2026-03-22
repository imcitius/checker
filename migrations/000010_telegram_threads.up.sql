CREATE TABLE IF NOT EXISTS telegram_alert_threads (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    check_uuid  TEXT NOT NULL,
    chat_id     TEXT NOT NULL,
    message_id  INTEGER NOT NULL,
    is_resolved INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    resolved_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_tg_threads_unresolved
    ON telegram_alert_threads (check_uuid, is_resolved) WHERE is_resolved = 0;
CREATE INDEX IF NOT EXISTS idx_tg_threads_message
    ON telegram_alert_threads (chat_id, message_id) WHERE is_resolved = 0;
