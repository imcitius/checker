CREATE TABLE IF NOT EXISTS check_results (
    id           BIGSERIAL PRIMARY KEY,
    check_uuid   TEXT NOT NULL,
    region       TEXT NOT NULL,
    is_healthy   BOOLEAN NOT NULL,
    message      TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cycle_key    TIMESTAMPTZ NOT NULL,
    evaluated_at TIMESTAMPTZ
);

CREATE INDEX idx_check_results_unevaluated
    ON check_results (check_uuid, cycle_key) WHERE evaluated_at IS NULL;

CREATE INDEX idx_check_results_cleanup
    ON check_results (created_at);
