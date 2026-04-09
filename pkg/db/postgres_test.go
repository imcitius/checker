// SPDX-License-Identifier: BUSL-1.1

package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/imcitius/checker/pkg/config"
	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresDB_Integration(t *testing.T) {
	connStr := os.Getenv("CHECKER_PG_CONN")
	if connStr == "" {
		t.Skip("Skipping integration test: CHECKER_PG_CONN not set")
	}

	// Setup config
	cfg := &config.Config{}
	cfg.DB.Driver = "postgres"
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

// setupPostgresTestDB creates a PostgresDB for integration tests.
// Requires CHECKER_PG_CONN env var to be set.
func setupPostgresTestDB(t *testing.T) (*db.PostgresDB, context.Context) {
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

	ctx := context.Background()
	repo, err := db.NewPostgresDB(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { repo.Close() })

	return repo, ctx
}

func TestPostgresDB_NewCheckTypes_Roundtrip(t *testing.T) {
	repo, ctx := setupPostgresTestDB(t)

	// Define test cases for all new check types
	testCases := []struct {
		name       string
		checkType  string
		config     models.CheckConfig
		assertType func(t *testing.T, cfg models.CheckConfig)
	}{
		{
			name:      "DNS check roundtrip",
			checkType: "dns",
			config: &models.DNSCheckConfig{
				Host:       "8.8.8.8",
				Domain:     "example.com",
				RecordType: "A",
				Timeout:    "5s",
				Expected:   "93.184.216.34",
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				dc, ok := cfg.(*models.DNSCheckConfig)
				require.True(t, ok, "expected *DNSCheckConfig, got %T", cfg)
				assert.Equal(t, "example.com", dc.Domain)
				assert.Equal(t, "A", dc.RecordType)
				assert.Equal(t, "8.8.8.8", dc.Host)
				assert.Equal(t, "93.184.216.34", dc.Expected)
			},
		},
		{
			name:      "SSL cert check roundtrip",
			checkType: "ssl_cert",
			config: &models.SSLCertCheckConfig{
				Host:              "example.com",
				Port:              443,
				Timeout:           "10s",
				ExpiryWarningDays: 30,
				ValidateChain:     true,
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				sc, ok := cfg.(*models.SSLCertCheckConfig)
				require.True(t, ok, "expected *SSLCertCheckConfig, got %T", cfg)
				assert.Equal(t, "example.com", sc.Host)
				assert.Equal(t, 443, sc.Port)
				assert.Equal(t, 30, sc.ExpiryWarningDays)
				assert.True(t, sc.ValidateChain)
			},
		},
		{
			name:      "SSH check roundtrip",
			checkType: "ssh",
			config: &models.SSHCheckConfig{
				Host:         "server.example.com",
				Port:         22,
				Timeout:      "5s",
				ExpectBanner: "SSH-2.0",
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				sc, ok := cfg.(*models.SSHCheckConfig)
				require.True(t, ok, "expected *SSHCheckConfig, got %T", cfg)
				assert.Equal(t, "server.example.com", sc.Host)
				assert.Equal(t, 22, sc.Port)
				assert.Equal(t, "SSH-2.0", sc.ExpectBanner)
			},
		},
		{
			name:      "Redis check roundtrip",
			checkType: "redis",
			config: &models.RedisCheckConfig{
				Host:     "redis.example.com",
				Port:     6379,
				Timeout:  "5s",
				Password: "secret",
				DB:       1,
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				rc, ok := cfg.(*models.RedisCheckConfig)
				require.True(t, ok, "expected *RedisCheckConfig, got %T", cfg)
				assert.Equal(t, "redis.example.com", rc.Host)
				assert.Equal(t, 6379, rc.Port)
				assert.Equal(t, "secret", rc.Password)
				assert.Equal(t, 1, rc.DB)
			},
		},
		{
			name:      "MongoDB check roundtrip",
			checkType: "mongodb",
			config: &models.MongoDBCheckConfig{
				URI:     "mongodb://localhost:27017/test",
				Timeout: "10s",
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				mc, ok := cfg.(*models.MongoDBCheckConfig)
				require.True(t, ok, "expected *MongoDBCheckConfig, got %T", cfg)
				assert.Equal(t, "mongodb://localhost:27017/test", mc.URI)
			},
		},
		{
			name:      "Domain expiry check roundtrip",
			checkType: "domain_expiry",
			config: &models.DomainExpiryCheckConfig{
				Domain:            "example.com",
				Timeout:           "10s",
				ExpiryWarningDays: 30,
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				dc, ok := cfg.(*models.DomainExpiryCheckConfig)
				require.True(t, ok, "expected *DomainExpiryCheckConfig, got %T", cfg)
				assert.Equal(t, "example.com", dc.Domain)
				assert.Equal(t, 30, dc.ExpiryWarningDays)
			},
		},
		{
			name:      "SMTP check roundtrip",
			checkType: "smtp",
			config: &models.SMTPCheckConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Timeout:  "10s",
				StartTLS: true,
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				sc, ok := cfg.(*models.SMTPCheckConfig)
				require.True(t, ok, "expected *SMTPCheckConfig, got %T", cfg)
				assert.Equal(t, "smtp.example.com", sc.Host)
				assert.Equal(t, 587, sc.Port)
				assert.True(t, sc.StartTLS)
			},
		},
		{
			name:      "gRPC health check roundtrip",
			checkType: "grpc_health",
			config: &models.GRPCHealthCheckConfig{
				Host:    "grpc.example.com:50051",
				Timeout: "5s",
				UseTLS:  true,
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				gc, ok := cfg.(*models.GRPCHealthCheckConfig)
				require.True(t, ok, "expected *GRPCHealthCheckConfig, got %T", cfg)
				assert.Equal(t, "grpc.example.com:50051", gc.Host)
				assert.True(t, gc.UseTLS)
			},
		},
		{
			name:      "WebSocket check roundtrip",
			checkType: "websocket",
			config: &models.WebSocketCheckConfig{
				URL:     "wss://ws.example.com/socket",
				Timeout: "5s",
			},
			assertType: func(t *testing.T, cfg models.CheckConfig) {
				wc, ok := cfg.(*models.WebSocketCheckConfig)
				require.True(t, ok, "expected *WebSocketCheckConfig, got %T", cfg)
				assert.Equal(t, "wss://ws.example.com/socket", wc.URL)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			def := models.CheckDefinition{
				Name:      "test-" + tc.checkType,
				Project:   "integration-test",
				GroupName: "new-types",
				Type:      tc.checkType,
				Enabled:   true,
				Config:    tc.config,
			}

			// Create
			id, err := repo.CreateCheckDefinition(ctx, def)
			require.NoError(t, err)
			require.NotEmpty(t, id)

			// Read back
			fetched, err := repo.GetCheckDefinitionByUUID(ctx, id)
			require.NoError(t, err)
			require.NotNil(t, fetched.Config, "Config should not be nil for type %s", tc.checkType)

			// Assert correct type and values
			tc.assertType(t, fetched.Config)

			// Cleanup
			err = repo.DeleteCheckDefinition(ctx, id)
			require.NoError(t, err)
		})
	}
}
