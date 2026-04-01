CREATE TABLE alert_history (
    id SERIAL PRIMARY KEY,
    check_uuid TEXT NOT NULL,
    check_name TEXT NOT NULL,
    project TEXT NOT NULL,
    group_name TEXT NOT NULL DEFAULT '',
    check_type TEXT NOT NULL,
    message TEXT NOT NULL,
    alert_type TEXT NOT NULL DEFAULT 'slack',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_alert_history_created ON alert_history (created_at DESC);
CREATE INDEX idx_alert_history_check ON alert_history (check_uuid, created_at DESC);
CREATE INDEX idx_alert_history_project ON alert_history (project, created_at DESC);
CREATE INDEX idx_alert_history_unresolved ON alert_history (is_resolved) WHERE is_resolved = FALSE;
