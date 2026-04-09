// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestPostgreSQLCheck_Run(t *testing.T) {
	// Create a mock logger
	logger := logrus.WithField("test", "TestPostgreSQLCheck_Run")

	// Test case: Invalid timeout
	t.Run("Invalid timeout", func(t *testing.T) {
		check := PostgreSQLCheck{
			Host:    "localhost",
			Port:    5432,
			Timeout: "invalid",
			Logger:  logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot parse timeout")
	})

	// Test case: Empty host
	t.Run("Empty host", func(t *testing.T) {
		check := PostgreSQLCheck{
			Host:    "",
			Port:    5432,
			Timeout: "5s",
			Logger:  logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrEmptyHost)
	})

	// Note: Add integration tests with a real PostgreSQL instance if available
}

func TestPostgreSQLTimeCheck_Run(t *testing.T) {
	// Create a mock logger
	logger := logrus.WithField("test", "TestPostgreSQLTimeCheck_Run")

	// Test case: Invalid timeout
	t.Run("Invalid timeout", func(t *testing.T) {
		check := PostgreSQLTimeCheck{
			Host:     "localhost",
			Port:     5432,
			Timeout:  "invalid",
			TimeType: "timestamp",
			Config: PostgreSQLTimeQueryConfig{
				Difference: "5m",
			},
			Logger: logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot parse timeout")
	})

	// Test case: Invalid difference
	t.Run("Invalid difference", func(t *testing.T) {
		check := PostgreSQLTimeCheck{
			Host:     "localhost",
			Port:     5432,
			Timeout:  "5s",
			TimeType: "timestamp",
			Config: PostgreSQLTimeQueryConfig{
				Difference: "invalid",
			},
			Logger: logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot parse difference")
	})

	// Test case: Empty host
	t.Run("Empty host", func(t *testing.T) {
		check := PostgreSQLTimeCheck{
			Host:     "",
			Port:     5432,
			Timeout:  "5s",
			TimeType: "timestamp",
			Config: PostgreSQLTimeQueryConfig{
				Difference: "5m",
			},
			Logger: logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrEmptyHost)
	})

	// Note: Add integration tests with a real PostgreSQL instance if available
}

func TestPostgreSQLReplicationCheck_Run(t *testing.T) {
	// Create a mock logger
	logger := logrus.WithField("test", "TestPostgreSQLReplicationCheck_Run")

	// Test case: Invalid timeout
	t.Run("Invalid timeout", func(t *testing.T) {
		check := PostgreSQLReplicationCheck{
			Host:      "localhost",
			Port:      5432,
			Timeout:   "invalid",
			CheckType: "replication",
			Config: PostgreSQLReplicationConfig{
				Lag: "5s",
			},
			Logger: logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot parse timeout")
	})

	// Test case: Empty host
	t.Run("Empty host", func(t *testing.T) {
		check := PostgreSQLReplicationCheck{
			Host:      "",
			Port:      5432,
			Timeout:   "5s",
			CheckType: "replication",
			Config: PostgreSQLReplicationConfig{
				Lag: "5s",
			},
			Logger: logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrEmptyHost)
	})

	// Note: Add integration tests with a real PostgreSQL instance if available
}
