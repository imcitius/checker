package db_test

import (
	"context"
	"os"
	"testing"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresDB_Integration(t *testing.T) {
	connStr := os.Getenv("CHECKER_PG_CONN")
	if connStr == "" {
		t.Skip("Skipping integration test: CHECKER_PG_CONN not set")
	}

	// Setup config
	// Setup config
	cfg := &config.Config{}
	cfg.DB.Protocol = "postgres"
	cfg.DB.Host = "localhost:5432"
	cfg.DB.Username = "postgres"
	cfg.DB.Password = "password"
	cfg.DB.Database = "checker_test"

	// Allow overriding via env vars for the test
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

	ctx := context.Background()
	repo, err := db.NewPostgresDB(cfg)
	require.NoError(t, err)
	defer repo.Close()

	// Test Create
	checkDef := models.CheckDefinition{
		Name:      "Integration Test Check",
		Project:   "Test Project",
		GroupName: "Test Group",
		Type:      "http",
		Enabled:   true,
		Config: &models.HTTPCheckConfig{
			URL:     "http://example.com",
			Timeout: "5s",
		},
	}

	id, err := repo.CreateCheckDefinition(ctx, checkDef)
	require.NoError(t, err)
	require.NotEmpty(t, id)
	checkDef.UUID = id // Update UUID with returned one if it was generated

	// Test Get
	fetchedDef, err := repo.GetCheckDefinitionByUUID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, checkDef.Name, fetchedDef.Name)
	assert.Equal(t, checkDef.Project, fetchedDef.Project)

	// Test Update
	fetchedDef.Name = "Updated Test Check"
	err = repo.UpdateCheckDefinition(ctx, fetchedDef)
	require.NoError(t, err)

	updatedDef, err := repo.GetCheckDefinitionByUUID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Test Check", updatedDef.Name)

	// Test Toggle
	err = repo.ToggleCheckDefinition(ctx, id, false)
	require.NoError(t, err)

	toggledDef, err := repo.GetCheckDefinitionByUUID(ctx, id)
	require.NoError(t, err)
	assert.False(t, toggledDef.Enabled)

	// Test Delete
	err = repo.DeleteCheckDefinition(ctx, id)
	require.NoError(t, err)

	_, err = repo.GetCheckDefinitionByUUID(ctx, id)
	assert.Error(t, err) // Should error as not found
}
