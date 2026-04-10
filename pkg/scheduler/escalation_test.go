// SPDX-License-Identifier: BUSL-1.1

package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/imcitius/checker/pkg/config"
	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// mockEscalationRepo implements db.Repository for escalation tests.
// It embeds full interface stubs and overrides escalation-specific methods.
type mockEscalationRepo struct {
	policies      map[string]models.EscalationPolicy
	notifications []models.EscalationNotification
}

func (m *mockEscalationRepo) Close() {}

func newMockEscalationRepo() *mockEscalationRepo {
	return &mockEscalationRepo{
		policies: make(map[string]models.EscalationPolicy),
	}
}

// Escalation-specific methods (actual implementations)

func (m *mockEscalationRepo) GetEscalationPolicyByName(_ context.Context, name string) (models.EscalationPolicy, error) {
	p, ok := m.policies[name]
	if !ok {
		return models.EscalationPolicy{}, fmt.Errorf("escalation policy not found")
	}
	return p, nil
}

func (m *mockEscalationRepo) GetEscalationNotifications(_ context.Context, checkUUID, policyName string) ([]models.EscalationNotification, error) {
	var result []models.EscalationNotification
	for _, n := range m.notifications {
		if n.CheckUUID == checkUUID && n.PolicyName == policyName {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *mockEscalationRepo) CreateEscalationNotification(_ context.Context, n models.EscalationNotification) error {
	m.notifications = append(m.notifications, n)
	return nil
}

func (m *mockEscalationRepo) DeleteEscalationNotifications(_ context.Context, checkUUID string) error {
	var remaining []models.EscalationNotification
	for _, n := range m.notifications {
		if n.CheckUUID != checkUUID {
			remaining = append(remaining, n)
		}
	}
	m.notifications = remaining
	return nil
}

// All other Repository interface methods (stubs)

func (m *mockEscalationRepo) GetAllCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockEscalationRepo) GetEnabledCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockEscalationRepo) GetCheckDefinitionByUUID(_ context.Context, _ string) (models.CheckDefinition, error) {
	return models.CheckDefinition{}, nil
}
func (m *mockEscalationRepo) CreateCheckDefinition(_ context.Context, _ models.CheckDefinition) (string, error) {
	return "", nil
}
func (m *mockEscalationRepo) UpdateCheckDefinition(_ context.Context, _ models.CheckDefinition) error {
	return nil
}
func (m *mockEscalationRepo) DeleteCheckDefinition(_ context.Context, _ string) error { return nil }
func (m *mockEscalationRepo) ToggleCheckDefinition(_ context.Context, _ string, _ bool) error {
	return nil
}
func (m *mockEscalationRepo) UpdateCheckStatus(_ context.Context, _ models.CheckStatus) error {
	return nil
}
func (m *mockEscalationRepo) SetMaintenanceWindow(_ context.Context, _ string, _ *time.Time) error {
	return nil
}
func (m *mockEscalationRepo) BulkToggleCheckDefinitions(_ context.Context, _ []string, _ bool) (int64, error) {
	return 0, nil
}
func (m *mockEscalationRepo) BulkDeleteCheckDefinitions(_ context.Context, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockEscalationRepo) BulkUpdateAlertChannels(_ context.Context, _ []string, _ string, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockEscalationRepo) GetAllProjects(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockEscalationRepo) GetAllCheckTypes(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockEscalationRepo) ConvertConfigToCheckDefinitions(_ context.Context, _ *config.Config) error {
	return nil
}
func (m *mockEscalationRepo) CountCheckDefinitions(_ context.Context) (int, error) { return 0, nil }
func (m *mockEscalationRepo) GetAllDefaultTimeouts() map[string]string              { return nil }
func (m *mockEscalationRepo) CreateSlackThread(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (m *mockEscalationRepo) GetUnresolvedThread(_ context.Context, _ string) (models.SlackAlertThread, error) {
	return models.SlackAlertThread{}, fmt.Errorf("not found")
}
func (m *mockEscalationRepo) ResolveThread(_ context.Context, _ string) error    { return nil }
func (m *mockEscalationRepo) UpdateSlackThread(_ context.Context, _, _, _ string) error { return nil }
func (m *mockEscalationRepo) CreateTelegramThread(_ context.Context, _, _ string, _ int) error {
	return nil
}
func (m *mockEscalationRepo) GetUnresolvedTelegramThread(_ context.Context, _ string) (models.TelegramAlertThread, error) {
	return models.TelegramAlertThread{}, fmt.Errorf("not found")
}
func (m *mockEscalationRepo) GetTelegramThreadByMessage(_ context.Context, _ string, _ int) (models.TelegramAlertThread, error) {
	return models.TelegramAlertThread{}, fmt.Errorf("not found")
}
func (m *mockEscalationRepo) ResolveTelegramThread(_ context.Context, _ string) error { return nil }
func (m *mockEscalationRepo) CreateDiscordThread(_ context.Context, _, _, _, _ string) error { return nil }
func (m *mockEscalationRepo) GetUnresolvedDiscordThread(_ context.Context, _ string) (models.DiscordAlertThread, error) {
	return models.DiscordAlertThread{}, fmt.Errorf("not found")
}
func (m *mockEscalationRepo) ResolveDiscordThread(_ context.Context, _ string) error { return nil }
func (m *mockEscalationRepo) CreateSilence(_ context.Context, _ models.AlertSilence) error {
	return nil
}
func (m *mockEscalationRepo) IsCheckSilenced(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockEscalationRepo) IsChannelSilenced(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockEscalationRepo) DeactivateSilence(_ context.Context, _, _ string) error { return nil }
func (m *mockEscalationRepo) DeactivateSilenceByID(_ context.Context, _ int) error  { return nil }
func (m *mockEscalationRepo) GetActiveSilences(_ context.Context) ([]models.AlertSilence, error) {
	return nil, nil
}
func (m *mockEscalationRepo) GetUnhealthyChecks(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockEscalationRepo) CreateAlertEvent(_ context.Context, _ models.AlertEvent) error {
	return nil
}
func (m *mockEscalationRepo) ResolveAlertEvent(_ context.Context, _ string) error { return nil }
func (m *mockEscalationRepo) GetAlertHistory(_ context.Context, _, _ int, _ models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	return nil, 0, nil
}
func (m *mockEscalationRepo) GetAllEscalationPolicies(_ context.Context) ([]models.EscalationPolicy, error) {
	var policies []models.EscalationPolicy
	for _, p := range m.policies {
		policies = append(policies, p)
	}
	return policies, nil
}
func (m *mockEscalationRepo) CreateEscalationPolicy(_ context.Context, p models.EscalationPolicy) error {
	m.policies[p.Name] = p
	return nil
}
func (m *mockEscalationRepo) UpdateEscalationPolicy(_ context.Context, p models.EscalationPolicy) error {
	m.policies[p.Name] = p
	return nil
}
func (m *mockEscalationRepo) DeleteEscalationPolicy(_ context.Context, name string) error {
	delete(m.policies, name)
	return nil
}

// Alert channel stubs (not used in escalation tests)
func (m *mockEscalationRepo) GetAllAlertChannels(_ context.Context) ([]models.AlertChannel, error) {
	return nil, nil
}
func (m *mockEscalationRepo) GetAlertChannelByName(_ context.Context, _ string) (models.AlertChannel, error) {
	return models.AlertChannel{}, fmt.Errorf("not found")
}
func (m *mockEscalationRepo) CreateAlertChannel(_ context.Context, _ models.AlertChannel) error {
	return nil
}
func (m *mockEscalationRepo) UpdateAlertChannel(_ context.Context, _ models.AlertChannel) error {
	return nil
}
func (m *mockEscalationRepo) DeleteAlertChannel(_ context.Context, _ string) error { return nil }
func (m *mockEscalationRepo) MigrateLegacyAlertFields(_ context.Context) (int, error) {
	return 0, nil
}
func (m *mockEscalationRepo) GetSetting(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not found")
}
func (m *mockEscalationRepo) SetSetting(_ context.Context, _, _ string) error { return nil }
func (m *mockEscalationRepo) GetCheckDefaults(_ context.Context) (models.CheckDefaults, error) {
	return models.CheckDefaults{}, nil
}
func (m *mockEscalationRepo) SaveCheckDefaults(_ context.Context, _ models.CheckDefaults) error {
	return nil
}
func (m *mockEscalationRepo) InsertCheckResult(_ context.Context, _ models.CheckResult) error {
	return nil
}
func (m *mockEscalationRepo) GetLatestRegionResults(_ context.Context, _ string) ([]models.CheckResult, error) {
	return nil, nil
}
func (m *mockEscalationRepo) GetUnevaluatedCycles(_ context.Context, _ int, _ time.Duration) ([]db.UnevaluatedCycle, error) {
	return nil, nil
}
func (m *mockEscalationRepo) ClaimCycleForEvaluation(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}
func (m *mockEscalationRepo) GetCycleResults(_ context.Context, _ string, _ time.Time) ([]models.CheckResult, error) {
	return nil, nil
}
func (m *mockEscalationRepo) PurgeOldCheckResults(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

// --- Tests ---

func TestProcessEscalation_NoPolicyName(t *testing.T) {
	repo := newMockEscalationRepo()

	checkDef := models.CheckDefinition{
		UUID:                 "test-uuid-1",
		EscalationPolicyName: "",
	}
	checkStatus := models.CheckStatus{
		UUID:      "test-uuid-1",
		IsHealthy: false,
		LastRun:   time.Now(),
	}

	processEscalation(repo, checkDef, checkStatus, nil)

	if len(repo.notifications) != 0 {
		t.Errorf("Expected no notifications, got %d", len(repo.notifications))
	}
}

func TestProcessEscalation_ImmediateStep(t *testing.T) {
	repo := newMockEscalationRepo()
	repo.policies["test-policy"] = models.EscalationPolicy{
		Name: "test-policy",
		Steps: []models.EscalationStep{
			{Channel: "telegram", DelayMin: 0},
		},
	}

	checkDef := models.CheckDefinition{
		UUID:                 "test-uuid-2",
		Name:                 "test-check",
		Project:              "test-project",
		EscalationPolicyName: "test-policy",
	}
	checkStatus := models.CheckStatus{
		UUID:      "test-uuid-2",
		IsHealthy: false,
		LastRun:   time.Now(),
		Message:   "connection refused",
	}

	processEscalation(repo, checkDef, checkStatus, nil)

	if len(repo.notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(repo.notifications))
	}
	if repo.notifications[0].StepIndex != 0 {
		t.Errorf("Expected step index 0, got %d", repo.notifications[0].StepIndex)
	}
	if repo.notifications[0].PolicyName != "test-policy" {
		t.Errorf("Expected policy name 'test-policy', got %s", repo.notifications[0].PolicyName)
	}
}

func TestProcessEscalation_DelayedStepNotReady(t *testing.T) {
	repo := newMockEscalationRepo()
	repo.policies["delayed-policy"] = models.EscalationPolicy{
		Name: "delayed-policy",
		Steps: []models.EscalationStep{
			{Channel: "telegram", DelayMin: 0},
			{Channel: "pagerduty", DelayMin: 10},
		},
	}

	repo.notifications = []models.EscalationNotification{
		{
			CheckUUID:  "test-uuid-3",
			PolicyName: "delayed-policy",
			StepIndex:  0,
			NotifiedAt: time.Now(),
		},
	}

	checkDef := models.CheckDefinition{
		UUID:                 "test-uuid-3",
		Name:                 "test-check",
		Project:              "test-project",
		EscalationPolicyName: "delayed-policy",
	}
	checkStatus := models.CheckStatus{
		UUID:      "test-uuid-3",
		IsHealthy: false,
		LastRun:   time.Now(),
		Message:   "timeout",
	}

	processEscalation(repo, checkDef, checkStatus, nil)

	if len(repo.notifications) != 1 {
		t.Errorf("Expected 1 notification (delayed step not ready), got %d", len(repo.notifications))
	}
}

func TestProcessEscalation_DelayedStepReady(t *testing.T) {
	repo := newMockEscalationRepo()
	repo.policies["delayed-policy"] = models.EscalationPolicy{
		Name: "delayed-policy",
		Steps: []models.EscalationStep{
			{Channel: "telegram", DelayMin: 0},
			{Channel: "pagerduty", DelayMin: 10},
		},
	}

	repo.notifications = []models.EscalationNotification{
		{
			CheckUUID:  "test-uuid-4",
			PolicyName: "delayed-policy",
			StepIndex:  0,
			NotifiedAt: time.Now().Add(-15 * time.Minute),
		},
	}

	checkDef := models.CheckDefinition{
		UUID:                 "test-uuid-4",
		Name:                 "test-check",
		Project:              "test-project",
		EscalationPolicyName: "delayed-policy",
	}
	checkStatus := models.CheckStatus{
		UUID:      "test-uuid-4",
		IsHealthy: false,
		LastRun:   time.Now().Add(-15 * time.Minute),
		Message:   "timeout",
	}

	processEscalation(repo, checkDef, checkStatus, nil)

	if len(repo.notifications) != 2 {
		t.Fatalf("Expected 2 notifications, got %d", len(repo.notifications))
	}
	if repo.notifications[1].StepIndex != 1 {
		t.Errorf("Expected step index 1, got %d", repo.notifications[1].StepIndex)
	}
}

func TestProcessEscalation_AlreadyNotified(t *testing.T) {
	repo := newMockEscalationRepo()
	repo.policies["test-policy"] = models.EscalationPolicy{
		Name: "test-policy",
		Steps: []models.EscalationStep{
			{Channel: "telegram", DelayMin: 0},
		},
	}

	repo.notifications = []models.EscalationNotification{
		{
			CheckUUID:  "test-uuid-5",
			PolicyName: "test-policy",
			StepIndex:  0,
			NotifiedAt: time.Now().Add(-5 * time.Minute),
		},
	}

	checkDef := models.CheckDefinition{
		UUID:                 "test-uuid-5",
		Name:                 "test-check",
		Project:              "test-project",
		EscalationPolicyName: "test-policy",
	}
	checkStatus := models.CheckStatus{
		UUID:      "test-uuid-5",
		IsHealthy: false,
		LastRun:   time.Now(),
		Message:   "timeout",
	}

	processEscalation(repo, checkDef, checkStatus, nil)

	if len(repo.notifications) != 1 {
		t.Errorf("Expected 1 notification (no duplicate), got %d", len(repo.notifications))
	}
}

func TestClearEscalationNotifications(t *testing.T) {
	repo := newMockEscalationRepo()
	repo.notifications = []models.EscalationNotification{
		{CheckUUID: "uuid-to-clear", PolicyName: "p1", StepIndex: 0, NotifiedAt: time.Now()},
		{CheckUUID: "uuid-to-clear", PolicyName: "p1", StepIndex: 1, NotifiedAt: time.Now()},
		{CheckUUID: "other-uuid", PolicyName: "p1", StepIndex: 0, NotifiedAt: time.Now()},
	}

	clearEscalationNotifications(repo, "uuid-to-clear")

	if len(repo.notifications) != 1 {
		t.Fatalf("Expected 1 remaining notification, got %d", len(repo.notifications))
	}
	if repo.notifications[0].CheckUUID != "other-uuid" {
		t.Errorf("Expected remaining notification to be for 'other-uuid', got %s", repo.notifications[0].CheckUUID)
	}
}

func TestProcessEscalation_PolicyNotFound(t *testing.T) {
	repo := newMockEscalationRepo()
	// No policies added

	checkDef := models.CheckDefinition{
		UUID:                 "test-uuid-6",
		Name:                 "test-check",
		EscalationPolicyName: "nonexistent-policy",
	}
	checkStatus := models.CheckStatus{
		UUID:      "test-uuid-6",
		IsHealthy: false,
		LastRun:   time.Now(),
	}

	// Should not panic, just log warning
	processEscalation(repo, checkDef, checkStatus, nil)

	if len(repo.notifications) != 0 {
		t.Errorf("Expected no notifications for missing policy, got %d", len(repo.notifications))
	}
}

func TestProcessEscalation_MultipleStepsAllReady(t *testing.T) {
	repo := newMockEscalationRepo()
	repo.policies["multi-step"] = models.EscalationPolicy{
		Name: "multi-step",
		Steps: []models.EscalationStep{
			{Channel: "telegram", DelayMin: 0},
			{Channel: "slack", DelayMin: 0},
			{Channel: "pagerduty", DelayMin: 0},
		},
	}

	checkDef := models.CheckDefinition{
		UUID:                 "test-uuid-7",
		Name:                 "test-check",
		Project:              "test-project",
		EscalationPolicyName: "multi-step",
	}
	checkStatus := models.CheckStatus{
		UUID:      "test-uuid-7",
		IsHealthy: false,
		LastRun:   time.Now(),
		Message:   "error",
	}

	processEscalation(repo, checkDef, checkStatus, nil)

	if len(repo.notifications) != 3 {
		t.Fatalf("Expected 3 notifications, got %d", len(repo.notifications))
	}
}
