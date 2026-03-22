package db

import (
	"context"
	"os"
	"testing"
	"time"

	"checker/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSQLiteDB(t *testing.T) *SQLiteDB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "checker-test-*.db")
	require.NoError(t, err)
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := NewSQLiteDB(tmpFile.Name())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteDB_CheckDefinitionCRUD(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	// Create
	def := models.CheckDefinition{
		UUID:        "test-uuid-1",
		Name:        "Test Check",
		Project:     "test-project",
		GroupName:   "test-group",
		Type:        "http",
		Description: "A test check",
		Enabled:     true,
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
		UpdatedAt:   time.Now().UTC().Truncate(time.Second),
		Duration:    "30s",
		Config: &models.HTTPCheckConfig{
			URL:     "https://example.com",
			Timeout: "5s",
			Code:    []int{200},
		},
		Severity: "warning",
	}

	uuid, err := db.CreateCheckDefinition(ctx, def)
	require.NoError(t, err)
	assert.Equal(t, "test-uuid-1", uuid)

	// Read
	got, err := db.GetCheckDefinitionByUUID(ctx, "test-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, def.Name, got.Name)
	assert.Equal(t, def.Project, got.Project)
	assert.Equal(t, def.Type, got.Type)
	assert.True(t, got.Enabled)
	assert.Equal(t, "warning", got.Severity)

	httpConfig, ok := got.Config.(*models.HTTPCheckConfig)
	require.True(t, ok)
	assert.Equal(t, "https://example.com", httpConfig.URL)

	// GetAll
	all, err := db.GetAllCheckDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// GetEnabled
	enabled, err := db.GetEnabledCheckDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, enabled, 1)

	// Toggle off
	err = db.ToggleCheckDefinition(ctx, "test-uuid-1", false)
	require.NoError(t, err)

	enabled, err = db.GetEnabledCheckDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, enabled, 0)

	// Update
	def.Name = "Updated Check"
	def.Severity = "critical"
	err = db.UpdateCheckDefinition(ctx, def)
	require.NoError(t, err)

	got, err = db.GetCheckDefinitionByUUID(ctx, "test-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated Check", got.Name)
	assert.Equal(t, "critical", got.Severity)

	// Delete
	err = db.DeleteCheckDefinition(ctx, "test-uuid-1")
	require.NoError(t, err)

	all, err = db.GetAllCheckDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 0)

	// Delete not-found
	err = db.DeleteCheckDefinition(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestSQLiteDB_BulkOperations(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	// Create 3 checks
	for i, uuid := range []string{"uuid-1", "uuid-2", "uuid-3"} {
		_, err := db.CreateCheckDefinition(ctx, models.CheckDefinition{
			UUID:      uuid,
			Name:      "Check " + uuid,
			Project:   "proj",
			Type:      "http",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Duration:  "30s",
			Config:    &models.HTTPCheckConfig{URL: "https://example.com"},
			Severity:  "critical",
		})
		require.NoError(t, err, "create check %d", i)
	}

	// Bulk toggle
	n, err := db.BulkToggleCheckDefinitions(ctx, []string{"uuid-1", "uuid-2"}, false)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	enabled, err := db.GetEnabledCheckDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, enabled, 1)
	assert.Equal(t, "uuid-3", enabled[0].UUID)

	// Bulk delete
	n, err = db.BulkDeleteCheckDefinitions(ctx, []string{"uuid-1", "uuid-3"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	all, err := db.GetAllCheckDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestSQLiteDB_CheckStatus(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	_, err := db.CreateCheckDefinition(ctx, models.CheckDefinition{
		UUID: "status-check", Name: "Status", Project: "proj", Type: "http",
		Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Duration: "30s", IsHealthy: true,
		Config: &models.HTTPCheckConfig{URL: "https://example.com"},
	})
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	err = db.UpdateCheckStatus(ctx, models.CheckStatus{
		UUID: "status-check", LastRun: now, IsHealthy: false,
		Message: "Connection timeout", LastAlertSent: now,
	})
	require.NoError(t, err)

	got, err := db.GetCheckDefinitionByUUID(ctx, "status-check")
	require.NoError(t, err)
	assert.False(t, got.IsHealthy)
	assert.Equal(t, "Connection timeout", got.LastMessage)

	// GetUnhealthyChecks
	unhealthy, err := db.GetUnhealthyChecks(ctx)
	require.NoError(t, err)
	assert.Len(t, unhealthy, 1)
	assert.Equal(t, "status-check", unhealthy[0].UUID)
}

func TestSQLiteDB_MaintenanceWindow(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	_, err := db.CreateCheckDefinition(ctx, models.CheckDefinition{
		UUID: "maint-check", Name: "Maint", Project: "proj", Type: "http",
		Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Duration: "30s", Config: &models.HTTPCheckConfig{URL: "https://example.com"},
	})
	require.NoError(t, err)

	until := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second)
	err = db.SetMaintenanceWindow(ctx, "maint-check", &until)
	require.NoError(t, err)

	got, err := db.GetCheckDefinitionByUUID(ctx, "maint-check")
	require.NoError(t, err)
	require.NotNil(t, got.MaintenanceUntil)

	// Clear
	err = db.SetMaintenanceWindow(ctx, "maint-check", nil)
	require.NoError(t, err)

	got, err = db.GetCheckDefinitionByUUID(ctx, "maint-check")
	require.NoError(t, err)
	assert.Nil(t, got.MaintenanceUntil)
}

func TestSQLiteDB_Projects(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	for _, p := range []string{"alpha", "beta", "alpha"} {
		_, err := db.CreateCheckDefinition(ctx, models.CheckDefinition{
			UUID: "proj-" + p + "-" + time.Now().String(), Name: "C", Project: p, Type: "http",
			Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Duration: "30s",
			Config: &models.HTTPCheckConfig{URL: "https://example.com"},
		})
		require.NoError(t, err)
	}

	projects, err := db.GetAllProjects(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta"}, projects)

	types, err := db.GetAllCheckTypes(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"http"}, types)
}

func TestSQLiteDB_SlackThreads(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	err := db.CreateSlackThread(ctx, "check-1", "C123", "1234.5678", "1234.0000")
	require.NoError(t, err)

	thread, err := db.GetUnresolvedThread(ctx, "check-1")
	require.NoError(t, err)
	assert.Equal(t, "C123", thread.ChannelID)
	assert.Equal(t, "1234.5678", thread.ThreadTs)
	assert.False(t, thread.IsResolved)

	err = db.ResolveThread(ctx, "check-1")
	require.NoError(t, err)

	_, err = db.GetUnresolvedThread(ctx, "check-1")
	assert.Error(t, err) // sql.ErrNoRows
}

func TestSQLiteDB_AlertSilences(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	// Create silence
	err := db.CreateSilence(ctx, models.AlertSilence{
		Scope:      "check",
		Target:     "check-uuid-1",
		SilencedBy: "user1",
		Reason:     "Maintenance",
		Active:     true,
	})
	require.NoError(t, err)

	// Check silenced
	silenced, err := db.IsCheckSilenced(ctx, "check-uuid-1", "some-project")
	require.NoError(t, err)
	assert.True(t, silenced)

	// Not silenced for different check
	silenced, err = db.IsCheckSilenced(ctx, "check-uuid-2", "some-project")
	require.NoError(t, err)
	assert.False(t, silenced)

	// Get active silences
	silences, err := db.GetActiveSilences(ctx)
	require.NoError(t, err)
	assert.Len(t, silences, 1)

	// Deactivate
	err = db.DeactivateSilence(ctx, "check", "check-uuid-1")
	require.NoError(t, err)

	silenced, err = db.IsCheckSilenced(ctx, "check-uuid-1", "some-project")
	require.NoError(t, err)
	assert.False(t, silenced)
}

func TestSQLiteDB_AlertHistory(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	// Create events
	err := db.CreateAlertEvent(ctx, models.AlertEvent{
		CheckUUID: "check-1", CheckName: "Test Check", Project: "proj",
		GroupName: "group", CheckType: "http", Message: "Down", AlertType: "slack",
	})
	require.NoError(t, err)

	err = db.CreateAlertEvent(ctx, models.AlertEvent{
		CheckUUID: "check-2", CheckName: "Other", Project: "proj2",
		GroupName: "group", CheckType: "tcp", Message: "Timeout", AlertType: "email",
	})
	require.NoError(t, err)

	// Query all
	events, total, err := db.GetAlertHistory(ctx, 10, 0, models.AlertHistoryFilters{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, events, 2)

	// Filter by project
	events, total, err = db.GetAlertHistory(ctx, 10, 0, models.AlertHistoryFilters{Project: "proj"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, events, 1)

	// Resolve
	err = db.ResolveAlertEvent(ctx, "check-1")
	require.NoError(t, err)

	resolved := true
	events, _, err = db.GetAlertHistory(ctx, 10, 0, models.AlertHistoryFilters{IsResolved: &resolved})
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "check-1", events[0].CheckUUID)
	assert.True(t, events[0].IsResolved)
}

func TestSQLiteDB_EscalationPolicies(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	policy := models.EscalationPolicy{
		Name: "critical-escalation",
		Steps: []models.EscalationStep{
			{Channel: "slack-alerts", DelayMin: 0},
			{Channel: "pagerduty", DelayMin: 15},
		},
	}

	err := db.CreateEscalationPolicy(ctx, policy)
	require.NoError(t, err)

	got, err := db.GetEscalationPolicyByName(ctx, "critical-escalation")
	require.NoError(t, err)
	assert.Equal(t, "critical-escalation", got.Name)
	assert.Len(t, got.Steps, 2)
	assert.Equal(t, "pagerduty", got.Steps[1].Channel)

	// Update
	policy.Steps = append(policy.Steps, models.EscalationStep{Channel: "email", DelayMin: 30})
	err = db.UpdateEscalationPolicy(ctx, policy)
	require.NoError(t, err)

	got, err = db.GetEscalationPolicyByName(ctx, "critical-escalation")
	require.NoError(t, err)
	assert.Len(t, got.Steps, 3)

	// GetAll
	all, err := db.GetAllEscalationPolicies(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// Delete
	err = db.DeleteEscalationPolicy(ctx, "critical-escalation")
	require.NoError(t, err)

	all, err = db.GetAllEscalationPolicies(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 0)
}

func TestSQLiteDB_EscalationNotifications(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	notif := models.EscalationNotification{
		CheckUUID:  "check-1",
		PolicyName: "policy-1",
		StepIndex:  0,
		NotifiedAt: time.Now().UTC().Truncate(time.Second),
	}

	err := db.CreateEscalationNotification(ctx, notif)
	require.NoError(t, err)

	// Duplicate should be ignored (INSERT OR IGNORE)
	err = db.CreateEscalationNotification(ctx, notif)
	require.NoError(t, err)

	notifications, err := db.GetEscalationNotifications(ctx, "check-1", "policy-1")
	require.NoError(t, err)
	assert.Len(t, notifications, 1)

	err = db.DeleteEscalationNotifications(ctx, "check-1")
	require.NoError(t, err)

	notifications, err = db.GetEscalationNotifications(ctx, "check-1", "policy-1")
	require.NoError(t, err)
	assert.Len(t, notifications, 0)
}

func TestSQLiteDB_AlertChannels(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	ch := models.AlertChannel{
		Name:   "slack-main",
		Type:   "slack",
		Config: []byte(`{"webhook_url":"https://hooks.slack.com/test"}`),
	}

	err := db.CreateAlertChannel(ctx, ch)
	require.NoError(t, err)

	got, err := db.GetAlertChannelByName(ctx, "slack-main")
	require.NoError(t, err)
	assert.Equal(t, "slack", got.Type)
	assert.Contains(t, string(got.Config), "hooks.slack.com")

	// Update
	ch.Config = []byte(`{"webhook_url":"https://hooks.slack.com/updated"}`)
	err = db.UpdateAlertChannel(ctx, ch)
	require.NoError(t, err)

	got, err = db.GetAlertChannelByName(ctx, "slack-main")
	require.NoError(t, err)
	assert.Contains(t, string(got.Config), "updated")

	// GetAll
	all, err := db.GetAllAlertChannels(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// Delete
	err = db.DeleteAlertChannel(ctx, "slack-main")
	require.NoError(t, err)

	_, err = db.GetAlertChannelByName(ctx, "slack-main")
	assert.Error(t, err)
}

func TestSQLiteDB_MigrateLegacyAlertFields(t *testing.T) {
	db := newTestSQLiteDB(t)
	ctx := context.Background()

	// MigrateLegacyAlertFields is now a no-op since alert_type and alert_destination
	// columns have been dropped. Verify it returns 0, nil.
	count, err := db.MigrateLegacyAlertFields(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should be a no-op returning 0")
}

// TestSQLiteDB_ImplementsRepository verifies the interface is satisfied at compile time.
func TestSQLiteDB_ImplementsRepository(t *testing.T) {
	var _ Repository = (*SQLiteDB)(nil)
}
