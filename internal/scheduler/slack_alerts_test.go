package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"checker/internal/config"
	"checker/internal/models"
	"checker/internal/slack"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Slack Sender ---

type mockSlackSender struct {
	postAlertCalls     []postAlertCall
	sendReplyCalls     []sendReplyCall
	sendResolveCalls   []sendResolveCall
	postAlertErr       error
	sendReplyErr       error
	sendResolveErr     error
	postAlertTs        string
	sendReplyTs        string
}

type postAlertCall struct {
	channelID string
	info      slack.CheckAlertInfo
}

type sendReplyCall struct {
	channelID string
	threadTS  string
	info      slack.CheckAlertInfo
}

type sendResolveCall struct {
	info             slack.CheckAlertInfo
	originalThreadTS string
	channelID        string
}

func (m *mockSlackSender) PostAlert(_ context.Context, channelID string, info slack.CheckAlertInfo) (string, error) {
	m.postAlertCalls = append(m.postAlertCalls, postAlertCall{channelID: channelID, info: info})
	ts := m.postAlertTs
	if ts == "" {
		ts = fmt.Sprintf("ts-%d", len(m.postAlertCalls))
	}
	return ts, m.postAlertErr
}

func (m *mockSlackSender) SendAlertReply(_ context.Context, channelID, threadTS string, info slack.CheckAlertInfo) (string, error) {
	m.sendReplyCalls = append(m.sendReplyCalls, sendReplyCall{channelID: channelID, threadTS: threadTS, info: info})
	ts := m.sendReplyTs
	if ts == "" {
		ts = fmt.Sprintf("reply-ts-%d", len(m.sendReplyCalls))
	}
	return ts, m.sendReplyErr
}

func (m *mockSlackSender) SendResolve(_ context.Context, info slack.CheckAlertInfo, originalThreadTS, channelID string) error {
	m.sendResolveCalls = append(m.sendResolveCalls, sendResolveCall{info: info, originalThreadTS: originalThreadTS, channelID: channelID})
	return m.sendResolveErr
}

// --- Mock Repository (only Slack-related methods) ---

type mockRepo struct {
	threads          map[string]models.SlackAlertThread // checkUUID -> thread
	silenced         map[string]bool                    // checkUUID -> silenced
	resolvedThreads  []string                           // checkUUIDs that had ResolveThread called
	createdThreads   []createdThread
	checkDefs        map[string]models.CheckDefinition // uuid -> def
	resolveErr       error
	createThreadErr  error
	getThreadErr     error
}

type createdThread struct {
	checkUUID string
	channelID string
	threadTs  string
	parentTs  string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		threads:   make(map[string]models.SlackAlertThread),
		silenced:  make(map[string]bool),
		checkDefs: make(map[string]models.CheckDefinition),
	}
}

func (r *mockRepo) IsCheckSilenced(_ context.Context, checkUUID, _ string) (bool, error) {
	return r.silenced[checkUUID], nil
}

func (r *mockRepo) GetUnresolvedThread(_ context.Context, checkUUID string) (models.SlackAlertThread, error) {
	if r.getThreadErr != nil {
		return models.SlackAlertThread{}, r.getThreadErr
	}
	t, ok := r.threads[checkUUID]
	if !ok {
		return models.SlackAlertThread{}, fmt.Errorf("no unresolved thread")
	}
	return t, nil
}

func (r *mockRepo) CreateSlackThread(_ context.Context, checkUUID, channelID, threadTs, parentTs string) error {
	if r.createThreadErr != nil {
		return r.createThreadErr
	}
	r.createdThreads = append(r.createdThreads, createdThread{
		checkUUID: checkUUID, channelID: channelID, threadTs: threadTs, parentTs: parentTs,
	})
	r.threads[checkUUID] = models.SlackAlertThread{
		CheckUUID: checkUUID, ChannelID: channelID, ThreadTs: threadTs, ParentTs: parentTs,
	}
	return nil
}

func (r *mockRepo) ResolveThread(_ context.Context, checkUUID string) error {
	if r.resolveErr != nil {
		return r.resolveErr
	}
	r.resolvedThreads = append(r.resolvedThreads, checkUUID)
	delete(r.threads, checkUUID)
	return nil
}

func (r *mockRepo) UpdateSlackThread(_ context.Context, _, _, _ string) error {
	return nil
}

func (r *mockRepo) DeactivateSilence(_ context.Context, _, _ string) error {
	return nil
}

func (r *mockRepo) GetCheckDefinitionByUUID(_ context.Context, uuid string) (models.CheckDefinition, error) {
	def, ok := r.checkDefs[uuid]
	if !ok {
		return models.CheckDefinition{}, fmt.Errorf("not found")
	}
	return def, nil
}

// Unused Repository interface methods (required for compilation if used as db.Repository)
func (r *mockRepo) GetAllCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (r *mockRepo) GetEnabledCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (r *mockRepo) CreateCheckDefinition(_ context.Context, _ models.CheckDefinition) (string, error) {
	return "", nil
}
func (r *mockRepo) UpdateCheckDefinition(_ context.Context, _ models.CheckDefinition) error {
	return nil
}
func (r *mockRepo) DeleteCheckDefinition(_ context.Context, _ string) error { return nil }
func (r *mockRepo) ToggleCheckDefinition(_ context.Context, _ string, _ bool) error {
	return nil
}
func (r *mockRepo) UpdateCheckStatus(_ context.Context, _ models.CheckStatus) error { return nil }
func (r *mockRepo) GetAllProjects(_ context.Context) ([]string, error)     { return nil, nil }
func (r *mockRepo) GetAllCheckTypes(_ context.Context) ([]string, error)   { return nil, nil }
func (r *mockRepo) GetAllDefaultTimeouts() map[string]string               { return nil }
func (r *mockRepo) CreateSilence(_ context.Context, _ models.AlertSilence) error {
	return nil
}
func (r *mockRepo) GetActiveSilences(_ context.Context) ([]models.AlertSilence, error) {
	return nil, nil
}
func (r *mockRepo) GetUnhealthyChecks(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (r *mockRepo) CreateAlertEvent(_ context.Context, _ models.AlertEvent) error { return nil }
func (r *mockRepo) ResolveAlertEvent(_ context.Context, _ string) error           { return nil }
func (r *mockRepo) GetAlertHistory(_ context.Context, _, _ int, _ models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	return nil, 0, nil
}
func (r *mockRepo) DeactivateSilenceByID(_ context.Context, _ int) error { return nil }
func (r *mockRepo) ConvertConfigToCheckDefinitions(_ context.Context, _ *config.Config) error {
	return nil
}
func (r *mockRepo) SetMaintenanceWindow(_ context.Context, _ string, _ *time.Time) error {
	return nil
}

// --- Tests ---

func TestSendAlert_NewFailure_CreatesNewThread(t *testing.T) {
	repo := newMockRepo()
	sender := &mockSlackSender{}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "connection refused"}

	alerter.SendAlert(context.Background(), checkDef, status, false)

	require.Len(t, sender.postAlertCalls, 1, "should post a new alert")
	assert.Equal(t, "C123", sender.postAlertCalls[0].channelID)
	assert.Len(t, sender.sendReplyCalls, 0, "should not send a reply")
	assert.Len(t, repo.createdThreads, 1, "should track the new thread")
	assert.Equal(t, "check-1", repo.createdThreads[0].checkUUID)
}

func TestSendAlert_OngoingFailure_RepliesToExistingThread(t *testing.T) {
	repo := newMockRepo()
	repo.threads["check-1"] = models.SlackAlertThread{
		CheckUUID: "check-1", ChannelID: "C123", ThreadTs: "1234.5678", ParentTs: "1234.5678",
	}
	sender := &mockSlackSender{}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "still failing"}

	// isNewIncident=false: ongoing failure, should reply to thread
	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.postAlertCalls, 0, "should NOT post a new alert")
	require.Len(t, sender.sendReplyCalls, 1, "should reply to existing thread")
	assert.Equal(t, "1234.5678", sender.sendReplyCalls[0].threadTS)
}

func TestSendAlert_ReFailure_CreatesNewThreadAfterResolvingStale(t *testing.T) {
	// This is the core bug scenario:
	// 1. Check was failing → thread created
	// 2. Check recovered but HandleRecovery was missed (race condition)
	// 3. Check fails again → should create NEW thread, not reply to old one

	repo := newMockRepo()
	// Simulate a stale unresolved thread from the previous incident
	repo.threads["check-1"] = models.SlackAlertThread{
		CheckUUID: "check-1", ChannelID: "C123", ThreadTs: "old-thread-ts",
		ParentTs: "old-thread-ts", CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	sender := &mockSlackSender{}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "new failure"}

	// isNewIncident=true: check was healthy before this failure
	alerter.SendAlert(context.Background(), checkDef, status, true)

	// Should have resolved the stale thread
	require.Len(t, repo.resolvedThreads, 1, "should resolve stale thread")
	assert.Equal(t, "check-1", repo.resolvedThreads[0])

	// Should have created a new top-level alert (NOT replied to old thread)
	require.Len(t, sender.postAlertCalls, 1, "should post a NEW alert")
	assert.Len(t, sender.sendReplyCalls, 0, "should NOT reply to old thread")

	// Should have tracked the new thread
	require.Len(t, repo.createdThreads, 1, "should track the new thread")
}

func TestSendAlert_ReFailure_NoStaleThread_CreatesNewThread(t *testing.T) {
	// Re-failure but HandleRecovery worked correctly (no stale thread)
	repo := newMockRepo()
	sender := &mockSlackSender{}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "new failure after recovery"}

	alerter.SendAlert(context.Background(), checkDef, status, true)

	require.Len(t, sender.postAlertCalls, 1, "should post a new alert")
	assert.Len(t, repo.resolvedThreads, 0, "no thread to resolve")
}

func TestHandleRecovery_ResolvesThread(t *testing.T) {
	repo := newMockRepo()
	repo.threads["check-1"] = models.SlackAlertThread{
		CheckUUID: "check-1", ChannelID: "C123", ThreadTs: "1234.5678", ParentTs: "1234.5678",
	}
	sender := &mockSlackSender{}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	require.Len(t, sender.sendResolveCalls, 1, "should send resolve to Slack")
	assert.Equal(t, "1234.5678", sender.sendResolveCalls[0].originalThreadTS)
	require.Len(t, repo.resolvedThreads, 1, "should resolve thread in DB")
}

func TestHandleRecovery_ResolvesThread_EvenIfSlackFails(t *testing.T) {
	// Verify that DB thread is resolved even if the Slack API call fails.
	// This is critical for preventing stale threads.
	repo := newMockRepo()
	repo.threads["check-1"] = models.SlackAlertThread{
		CheckUUID: "check-1", ChannelID: "C123", ThreadTs: "1234.5678", ParentTs: "1234.5678",
	}
	sender := &mockSlackSender{sendResolveErr: fmt.Errorf("slack API error")}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	// Slack call failed, but thread should still be resolved in DB
	require.Len(t, sender.sendResolveCalls, 1, "should attempt Slack resolve")
	require.Len(t, repo.resolvedThreads, 1, "should still resolve thread in DB despite Slack failure")
}

func TestSendAlert_FullIncidentCycle(t *testing.T) {
	// Simulate a complete incident lifecycle:
	// 1. Check fails → new thread
	// 2. Check still fails → reply to thread
	// 3. Check recovers → thread resolved
	// 4. Check fails again → NEW thread (not reply to old)

	repo := newMockRepo()
	sender := &mockSlackSender{}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "error"}

	// Step 1: First failure — new incident
	alerter.SendAlert(context.Background(), checkDef, status, true)
	require.Len(t, sender.postAlertCalls, 1, "step 1: should create new thread")
	assert.Len(t, sender.sendReplyCalls, 0)

	// Step 2: Ongoing failure — reply to existing thread
	alerter.SendAlert(context.Background(), checkDef, status, false)
	assert.Len(t, sender.postAlertCalls, 1, "step 2: should NOT create new thread")
	require.Len(t, sender.sendReplyCalls, 1, "step 2: should reply to thread")

	// Step 3: Recovery
	alerter.HandleRecovery(context.Background(), checkDef)
	require.Len(t, repo.resolvedThreads, 1, "step 3: should resolve thread")

	// Step 4: Re-failure — should create NEW thread
	alerter.SendAlert(context.Background(), checkDef, status, true)
	require.Len(t, sender.postAlertCalls, 2, "step 4: should create a NEW thread")
	assert.Len(t, sender.sendReplyCalls, 1, "step 4: should NOT add more replies")
}

func TestSendAlert_Silenced_SkipsAlert(t *testing.T) {
	repo := newMockRepo()
	repo.silenced["check-1"] = true
	sender := &mockSlackSender{}
	alerter := newSlackAlerterWithSender(sender, repo, "C123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "error"}

	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.postAlertCalls, 0, "should not post alert when silenced")
	assert.Len(t, sender.sendReplyCalls, 0, "should not reply when silenced")
}
