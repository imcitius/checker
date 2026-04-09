// SPDX-License-Identifier: BUSL-1.1

package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/imcitius/checker/pkg/config"
	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a PostgresDB for integration tests.
// Skips the test if CHECKER_PG_CONN env var is not set.
func setupTestDB(t *testing.T) *db.PostgresDB {
	t.Helper()
	connStr := os.Getenv("CHECKER_PG_CONN")
	if connStr == "" {
		t.Skip("Skipping integration test: CHECKER_PG_CONN not set")
	}

	cfg := &config.Config{}
	cfg.DB.Driver = "postgres"
	cfg.DB.Host = "localhost:5432"
	cfg.DB.Username = "postgres"
	cfg.DB.Password = "password"
	cfg.DB.Database = "checker_test"

	if h := os.Getenv("PG_HOST"); h != "" {
		cfg.DB.Host = h
	}
	if u := os.Getenv("PG_USER"); u != "" {
		cfg.DB.Username = u
	}
	if p := os.Getenv("PG_PASSWORD"); p != "" {
		cfg.DB.Password = p
	}
	if d := os.Getenv("PG_DB"); d != "" {
		cfg.DB.Database = d
	}

	repo, err := db.NewPostgresDB(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestSlackThreadLifecycle(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	checkUUID := "test-check-" + time.Now().Format("20060102150405")

	// Create a slack thread
	err := repo.CreateSlackThread(ctx, checkUUID, "C12345", "1234567890.123456", "1234567890.000000")
	require.NoError(t, err)

	// Get unresolved thread
	thread, err := repo.GetUnresolvedThread(ctx, checkUUID)
	require.NoError(t, err)
	assert.Equal(t, checkUUID, thread.CheckUUID)
	assert.Equal(t, "C12345", thread.ChannelID)
	assert.Equal(t, "1234567890.123456", thread.ThreadTs)
	assert.Equal(t, "1234567890.000000", thread.ParentTs)
	assert.False(t, thread.IsResolved)
	assert.Nil(t, thread.ResolvedAt)

	// Resolve the thread
	err = repo.ResolveThread(ctx, checkUUID)
	require.NoError(t, err)

	// Should no longer find an unresolved thread
	_, err = repo.GetUnresolvedThread(ctx, checkUUID)
	assert.Error(t, err, "expected error when no unresolved thread exists")
}

func TestIsCheckSilenced(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	checkUUID := "silenced-check-" + time.Now().Format("20060102150405")
	project := "silenced-project-" + time.Now().Format("20060102150405")

	// Initially not silenced
	silenced, err := repo.IsCheckSilenced(ctx, checkUUID, project)
	require.NoError(t, err)
	assert.False(t, silenced, "check should not be silenced initially")

	// Silence by check UUID
	expiresAt := time.Now().Add(1 * time.Hour)
	err = repo.CreateSilence(ctx, models.AlertSilence{
		Scope:      "check",
		Target:     checkUUID,
		SilencedBy: "U12345",
		ExpiresAt:  &expiresAt,
		Reason:     "test silence",
		Active:     true,
	})
	require.NoError(t, err)

	// Now should be silenced
	silenced, err = repo.IsCheckSilenced(ctx, checkUUID, project)
	require.NoError(t, err)
	assert.True(t, silenced, "check should be silenced after creating check-scope silence")

	// Different check, same project — not silenced by check scope
	silenced, err = repo.IsCheckSilenced(ctx, "other-check-uuid", project)
	require.NoError(t, err)
	assert.False(t, silenced, "different check should not be silenced by check-scope silence")

	// Silence by project
	err = repo.CreateSilence(ctx, models.AlertSilence{
		Scope:      "project",
		Target:     project,
		SilencedBy: "U12345",
		ExpiresAt:  nil, // no expiry
		Reason:     "project-wide silence",
		Active:     true,
	})
	require.NoError(t, err)

	// Now even a different check in the same project should be silenced
	silenced, err = repo.IsCheckSilenced(ctx, "other-check-uuid", project)
	require.NoError(t, err)
	assert.True(t, silenced, "check in silenced project should be silenced")
}

func TestIsCheckSilenced_ExpiredSilence(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	checkUUID := "expired-check-" + time.Now().Format("20060102150405")
	project := "expired-project-" + time.Now().Format("20060102150405")

	// Create an expired silence
	expiredAt := time.Now().Add(-1 * time.Hour) // expired 1 hour ago
	err := repo.CreateSilence(ctx, models.AlertSilence{
		Scope:      "check",
		Target:     checkUUID,
		SilencedBy: "U12345",
		ExpiresAt:  &expiredAt,
		Reason:     "already expired",
		Active:     true,
	})
	require.NoError(t, err)

	// Should NOT be silenced since the silence has expired
	silenced, err := repo.IsCheckSilenced(ctx, checkUUID, project)
	require.NoError(t, err)
	assert.False(t, silenced, "check should not be silenced when silence has expired")
}

func TestIsCheckSilenced_InactiveSilence(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	checkUUID := "inactive-check-" + time.Now().Format("20060102150405")
	project := "inactive-project-" + time.Now().Format("20060102150405")

	// Create an inactive silence (active=false)
	err := repo.CreateSilence(ctx, models.AlertSilence{
		Scope:      "check",
		Target:     checkUUID,
		SilencedBy: "U12345",
		ExpiresAt:  nil,
		Reason:     "deactivated silence",
		Active:     false,
	})
	require.NoError(t, err)

	// Should NOT be silenced since the silence is inactive
	silenced, err := repo.IsCheckSilenced(ctx, checkUUID, project)
	require.NoError(t, err)
	assert.False(t, silenced, "check should not be silenced when silence is inactive")
}
