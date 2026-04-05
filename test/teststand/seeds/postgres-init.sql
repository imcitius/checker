-- =============================================================================
-- PostgreSQL seed data for checker test stand
-- =============================================================================

-- -----------------------------------------------------------------------
-- heartbeat: used by pgsql_query_timestamp PASS scenario
-- Query: SELECT updated_at FROM heartbeat ORDER BY id DESC LIMIT 1
-- Expected: diff from now() < 60s
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS heartbeat (
    id         SERIAL PRIMARY KEY,
    status     TEXT        NOT NULL DEFAULT 'healthy',
    updated_at TIMESTAMP   NOT NULL DEFAULT NOW()
);
INSERT INTO heartbeat (status, updated_at) VALUES ('healthy', NOW());

-- -----------------------------------------------------------------------
-- heartbeat_unix: used by pgsql_query_unixtime PASS scenario
-- Query: SELECT EXTRACT(EPOCH FROM updated_at)::BIGINT FROM heartbeat_unix ORDER BY id DESC LIMIT 1
-- Expected: diff from now() < 60s
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS heartbeat_unix (
    id         SERIAL PRIMARY KEY,
    updated_at TIMESTAMP   NOT NULL DEFAULT NOW()
);
INSERT INTO heartbeat_unix (updated_at) VALUES (NOW());

-- -----------------------------------------------------------------------
-- heartbeat_stale: used by pgsql_query_timestamp FAIL and pgsql_query_unixtime FAIL scenarios
-- Query returns a timestamp > 1 hour ago -> check fails due to staleness
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS heartbeat_stale (
    id         SERIAL PRIMARY KEY,
    updated_at TIMESTAMP   NOT NULL DEFAULT NOW()
);
INSERT INTO heartbeat_stale (updated_at) VALUES (NOW() - INTERVAL '2 hours');

-- -----------------------------------------------------------------------
-- test_health: generic table for pgsql_query PASS scenario (SELECT 1 equivalent)
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS test_health (
    id         SERIAL PRIMARY KEY,
    status     TEXT        NOT NULL,
    updated_at TIMESTAMP   NOT NULL DEFAULT NOW()
);
INSERT INTO test_health (status) VALUES ('healthy');

-- -----------------------------------------------------------------------
-- repl_test: used by pgsql_replication PASS scenario
-- The checker writes here and reads from the replica
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS repl_test (
    id         SERIAL PRIMARY KEY,
    test_value INT         NOT NULL,
    created_at TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- -----------------------------------------------------------------------
-- Mock helper: simulate replication lag of 0 for pgsql_replication_status
-- -----------------------------------------------------------------------
CREATE OR REPLACE FUNCTION pg_mock_replication_lag()
RETURNS INTERVAL AS $$
BEGIN
    RETURN INTERVAL '0 seconds';
END;
$$ LANGUAGE plpgsql;
