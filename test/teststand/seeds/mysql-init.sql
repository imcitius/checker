-- =============================================================================
-- MySQL seed data for checker test stand
-- =============================================================================

USE checker_test;

-- -----------------------------------------------------------------------
-- heartbeat: used by mysql_query_unixtime PASS scenario (timestamp variant)
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS heartbeat (
    id         INT           AUTO_INCREMENT PRIMARY KEY,
    status     VARCHAR(50)   NOT NULL DEFAULT 'healthy',
    updated_at TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO heartbeat (status, updated_at) VALUES ('healthy', NOW());

-- -----------------------------------------------------------------------
-- heartbeat_unix: used by mysql_query_unixtime PASS scenario
-- Query: SELECT updated_at FROM heartbeat_unix ORDER BY id DESC LIMIT 1
-- Expected: UNIX_TIMESTAMP() - updated_at < 60
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS heartbeat_unix (
    id         INT    AUTO_INCREMENT PRIMARY KEY,
    updated_at BIGINT NOT NULL
);
INSERT INTO heartbeat_unix (updated_at) VALUES (UNIX_TIMESTAMP());

-- -----------------------------------------------------------------------
-- heartbeat_stale: used by mysql_query_unixtime FAIL scenario
-- updated_at is ~2 hours in the past -> staleness check fails
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS heartbeat_stale (
    id         INT    AUTO_INCREMENT PRIMARY KEY,
    updated_at BIGINT NOT NULL
);
INSERT INTO heartbeat_stale (updated_at) VALUES (UNIX_TIMESTAMP() - 7200);

-- -----------------------------------------------------------------------
-- test_health: generic table for mysql_query PASS scenario
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS test_health (
    id         INT          AUTO_INCREMENT PRIMARY KEY,
    status     VARCHAR(50)  NOT NULL,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO test_health (status) VALUES ('healthy');

-- -----------------------------------------------------------------------
-- replication_test: used by mysql_replication PASS scenario
-- The checker writes here and reads from the replica
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS replication_test (
    id         INT AUTO_INCREMENT PRIMARY KEY,
    test_value INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
