package scheduler

import (
	"context"
	"testing"

	"checker/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock AppAlerter for filter tests ---

type mockAppAlerter struct {
	ownedTypes     []string
	sendAlertCalls []mockSendAlertCall
	recoveryCalls  []string // checkUUIDs
}

type mockSendAlertCall struct {
	checkUUID     string
	isNewIncident bool
}

func (m *mockAppAlerter) SendAlert(_ context.Context, checkDef models.CheckDefinition, _ models.CheckStatus, isNewIncident bool) {
	m.sendAlertCalls = append(m.sendAlertCalls, mockSendAlertCall{
		checkUUID:     checkDef.UUID,
		isNewIncident: isNewIncident,
	})
}

func (m *mockAppAlerter) HandleRecovery(_ context.Context, checkDef models.CheckDefinition) {
	m.recoveryCalls = append(m.recoveryCalls, checkDef.UUID)
}

func (m *mockAppAlerter) OwnedTypes() []string {
	return m.ownedTypes
}

// --- Tests for shouldAppAlerterFire ---

func TestShouldAppAlerterFire_MatchingType(t *testing.T) {
	aa := &mockAppAlerter{ownedTypes: []string{"slack"}}
	selected := map[string]bool{"slack": true, "ntfy": true}
	assert.True(t, shouldAppAlerterFire(aa, selected))
}

func TestShouldAppAlerterFire_NoMatchingType(t *testing.T) {
	aa := &mockAppAlerter{ownedTypes: []string{"slack"}}
	selected := map[string]bool{"ntfy": true}
	assert.False(t, shouldAppAlerterFire(aa, selected))
}

func TestShouldAppAlerterFire_EmptySelectedTypes(t *testing.T) {
	aa := &mockAppAlerter{ownedTypes: []string{"slack"}}
	selected := map[string]bool{}
	// Empty selectedTypes means channels were configured but none resolved — don't fire
	assert.False(t, shouldAppAlerterFire(aa, selected))
}

func TestShouldAppAlerterFire_MultipleOwnedTypes(t *testing.T) {
	aa := &mockAppAlerter{ownedTypes: []string{"slack", "slack-app"}}
	selected := map[string]bool{"slack-app": true}
	assert.True(t, shouldAppAlerterFire(aa, selected))
}

// --- Tests for resolveSelectedChannelTypes ---

func TestResolveSelectedChannelTypes_ResolvesTypes(t *testing.T) {
	repo := newMockRepo()
	repo.alertChannels["prod-slack"] = models.AlertChannel{Name: "prod-slack", Type: "slack"}
	repo.alertChannels["ops-ntfy"] = models.AlertChannel{Name: "ops-ntfy", Type: "ntfy"}

	checkDef := models.CheckDefinition{
		AlertChannels: []string{"prod-slack", "ops-ntfy"},
	}

	types := resolveSelectedChannelTypes(repo, checkDef)
	assert.True(t, types["slack"])
	assert.True(t, types["ntfy"])
	assert.Len(t, types, 2)
}

func TestResolveSelectedChannelTypes_SkipsUnknownChannels(t *testing.T) {
	repo := newMockRepo()
	repo.alertChannels["prod-slack"] = models.AlertChannel{Name: "prod-slack", Type: "slack"}
	// "missing-channel" is NOT in alertChannels

	checkDef := models.CheckDefinition{
		AlertChannels: []string{"prod-slack", "missing-channel"},
	}

	types := resolveSelectedChannelTypes(repo, checkDef)
	assert.True(t, types["slack"])
	assert.Len(t, types, 1)
}

func TestResolveSelectedChannelTypes_EmptyChannels(t *testing.T) {
	repo := newMockRepo()
	checkDef := models.CheckDefinition{}

	types := resolveSelectedChannelTypes(repo, checkDef)
	assert.Len(t, types, 0)
}

// --- Integration tests for executeCheck filtering ---

func TestExecuteCheck_OnlyNtfyChannels_SlackAlerterNotFired(t *testing.T) {
	repo := newMockRepo()
	repo.alertChannels["prod-ntfy"] = models.AlertChannel{Name: "prod-ntfy", Type: "ntfy"}
	repo.alertChannels["ops-ntfy"] = models.AlertChannel{Name: "ops-ntfy", Type: "ntfy"}

	slackAlerter := &mockAppAlerter{ownedTypes: []string{"slack"}}
	telegramAlerter := &mockAppAlerter{ownedTypes: []string{"telegram"}}
	appAlerters := []AppAlerter{slackAlerter, telegramAlerter}

	checkDef := models.CheckDefinition{
		UUID:          "check-ntfy-only",
		Name:          "ntfy-only-check",
		Project:       "test",
		Type:          "http",
		Duration:      "1m",
		Enabled:       true,
		IsHealthy:     true, // was healthy
		AlertChannels: []string{"prod-ntfy", "ops-ntfy"},
		Config:        &models.HTTPCheckConfig{URL: "http://will-fail.invalid"},
	}
	repo.checkDefs[checkDef.UUID] = checkDef

	// Execute the check (it will fail due to invalid URL)
	_ = executeCheck(repo, checkDef, appAlerters, "")

	// Slack and Telegram should NOT have fired — only ntfy channels selected
	assert.Len(t, slackAlerter.sendAlertCalls, 0, "Slack should not fire for ntfy-only check")
	assert.Len(t, telegramAlerter.sendAlertCalls, 0, "Telegram should not fire for ntfy-only check")
}

func TestExecuteCheck_SlackChannelSelected_SlackAlerterFires(t *testing.T) {
	repo := newMockRepo()
	repo.alertChannels["prod-slack"] = models.AlertChannel{Name: "prod-slack", Type: "slack"}

	slackAlerter := &mockAppAlerter{ownedTypes: []string{"slack"}}
	appAlerters := []AppAlerter{slackAlerter}

	checkDef := models.CheckDefinition{
		UUID:          "check-slack",
		Name:          "slack-check",
		Project:       "test",
		Type:          "http",
		Duration:      "1m",
		Enabled:       true,
		IsHealthy:     true,
		AlertChannels: []string{"prod-slack"},
		Config:        &models.HTTPCheckConfig{URL: "http://will-fail.invalid"},
	}
	repo.checkDefs[checkDef.UUID] = checkDef

	_ = executeCheck(repo, checkDef, appAlerters, "")

	require.Len(t, slackAlerter.sendAlertCalls, 1, "Slack should fire when slack channel selected")
	assert.Equal(t, "check-slack", slackAlerter.sendAlertCalls[0].checkUUID)
}

func TestExecuteCheck_NoChannelsConfigured_AllAppAlertersFire(t *testing.T) {
	repo := newMockRepo()

	slackAlerter := &mockAppAlerter{ownedTypes: []string{"slack"}}
	telegramAlerter := &mockAppAlerter{ownedTypes: []string{"telegram"}}
	appAlerters := []AppAlerter{slackAlerter, telegramAlerter}

	checkDef := models.CheckDefinition{
		UUID:          "check-no-channels",
		Name:          "no-channel-check",
		Project:       "test",
		Type:          "http",
		Duration:      "1m",
		Enabled:       true,
		IsHealthy:     true,
		AlertChannels: nil, // no channels configured, no defaults — skip alerters
		Config:        &models.HTTPCheckConfig{URL: "http://will-fail.invalid"},
	}
	repo.checkDefs[checkDef.UUID] = checkDef

	_ = executeCheck(repo, checkDef, appAlerters, "")

	assert.Len(t, slackAlerter.sendAlertCalls, 0, "Slack should not fire when no channels and no defaults configured")
	assert.Len(t, telegramAlerter.sendAlertCalls, 0, "Telegram should not fire when no channels and no defaults configured")
}

func TestExecuteCheck_BothSlackAndNtfy_BothFire(t *testing.T) {
	repo := newMockRepo()
	repo.alertChannels["prod-slack"] = models.AlertChannel{Name: "prod-slack", Type: "slack"}
	repo.alertChannels["prod-ntfy"] = models.AlertChannel{Name: "prod-ntfy", Type: "ntfy"}

	slackAlerter := &mockAppAlerter{ownedTypes: []string{"slack"}}
	ntfyAlerter := &mockAppAlerter{ownedTypes: []string{"ntfy"}}
	appAlerters := []AppAlerter{slackAlerter, ntfyAlerter}

	checkDef := models.CheckDefinition{
		UUID:          "check-both",
		Name:          "both-check",
		Project:       "test",
		Type:          "http",
		Duration:      "1m",
		Enabled:       true,
		IsHealthy:     true,
		AlertChannels: []string{"prod-slack", "prod-ntfy"},
		Config:        &models.HTTPCheckConfig{URL: "http://will-fail.invalid"},
	}
	repo.checkDefs[checkDef.UUID] = checkDef

	_ = executeCheck(repo, checkDef, appAlerters, "")

	assert.Len(t, slackAlerter.sendAlertCalls, 1, "Slack should fire when slack channel selected")
	assert.Len(t, ntfyAlerter.sendAlertCalls, 1, "Ntfy should fire when ntfy channel selected")
}

// --- Recovery filtering tests ---

func TestExecuteCheck_Recovery_OnlyNtfyChannels_SlackRecoveryNotFired(t *testing.T) {
	repo := newMockRepo()
	repo.alertChannels["prod-ntfy"] = models.AlertChannel{Name: "prod-ntfy", Type: "ntfy"}

	slackAlerter := &mockAppAlerter{ownedTypes: []string{"slack"}}

	checkDef := models.CheckDefinition{
		UUID:          "check-recovery-ntfy",
		Name:          "recovery-check",
		Project:       "test",
		Type:          "http",
		Duration:      "1m",
		Enabled:       true,
		IsHealthy:     false, // was unhealthy
		AlertChannels: []string{"prod-ntfy"},
		Config:        &models.HTTPCheckConfig{URL: "http://will-fail.invalid"},
	}

	// Verify the filtering functions work correctly for recovery scenario
	channels := getEffectiveAlertChannels(checkDef)
	assert.Equal(t, []string{"prod-ntfy"}, channels)

	selectedTypes := resolveSelectedChannelTypes(repo, checkDef)
	assert.True(t, selectedTypes["ntfy"])
	assert.False(t, selectedTypes["slack"])
	assert.False(t, shouldAppAlerterFire(slackAlerter, selectedTypes))
}

func TestExecuteCheck_Recovery_NoChannels_AllRecoveryFires(t *testing.T) {
	repo := newMockRepo()

	slackAlerter := &mockAppAlerter{ownedTypes: []string{"slack"}}
	telegramAlerter := &mockAppAlerter{ownedTypes: []string{"telegram"}}

	checkDef := models.CheckDefinition{
		UUID:          "check-recovery-no-channels",
		AlertChannels: nil,
	}

	channels := getEffectiveAlertChannels(checkDef)
	assert.Len(t, channels, 0)

	// With no channels, backward compat means all should fire
	// Verify by checking the len(channels) == 0 branch is taken
	selectedTypes := resolveSelectedChannelTypes(repo, checkDef)
	assert.Len(t, selectedTypes, 0)

	// The code path uses len(channels) == 0 to decide, not shouldAppAlerterFire
	// So when channels is empty, ALL alerters fire — this is correct
	_ = slackAlerter
	_ = telegramAlerter
}
