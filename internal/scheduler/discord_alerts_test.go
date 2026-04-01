package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/discord"
	"checker/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure unused imports are referenced.
var _ = time.Now
var _ = config.Config{}

// Compile-time interface check.
var _ AppAlerter = (*DiscordAppAlerter)(nil)

// --- Mock Discord Sender ---

type mockDiscordSender struct {
	sendMessageCalls []sendMessageCall
	editMessageCalls []editMessageCall
	createThreadCalls []createThreadCall
	threadReplyCalls  []threadReplyCall
	sendMessageErr    error
	editMessageErr    error
	createThreadErr   error
	threadReplyErr    error
	sendMessageResult *discord.Message
	createThreadResult *discord.Channel
}

type sendMessageCall struct {
	channelID string
	payload   discord.MessagePayload
}

type editMessageCall struct {
	channelID string
	messageID string
	payload   discord.MessagePayload
}

type createThreadCall struct {
	channelID string
	messageID string
	name      string
}

type threadReplyCall struct {
	threadID string
	payload  discord.MessagePayload
}

func (m *mockDiscordSender) SendMessage(_ context.Context, channelID string, payload discord.MessagePayload) (*discord.Message, error) {
	m.sendMessageCalls = append(m.sendMessageCalls, sendMessageCall{channelID: channelID, payload: payload})
	if m.sendMessageErr != nil {
		return nil, m.sendMessageErr
	}
	msg := m.sendMessageResult
	if msg == nil {
		msg = &discord.Message{ID: fmt.Sprintf("msg-%d", len(m.sendMessageCalls)), ChannelID: channelID}
	}
	return msg, nil
}

func (m *mockDiscordSender) EditMessage(_ context.Context, channelID, messageID string, payload discord.MessagePayload) error {
	m.editMessageCalls = append(m.editMessageCalls, editMessageCall{channelID: channelID, messageID: messageID, payload: payload})
	return m.editMessageErr
}

func (m *mockDiscordSender) CreateThread(_ context.Context, channelID, messageID, name string) (*discord.Channel, error) {
	m.createThreadCalls = append(m.createThreadCalls, createThreadCall{channelID: channelID, messageID: messageID, name: name})
	if m.createThreadErr != nil {
		return nil, m.createThreadErr
	}
	ch := m.createThreadResult
	if ch == nil {
		ch = &discord.Channel{ID: fmt.Sprintf("thread-%d", len(m.createThreadCalls)), Name: name}
	}
	return ch, nil
}

func (m *mockDiscordSender) SendThreadReply(_ context.Context, threadID string, payload discord.MessagePayload) (*discord.Message, error) {
	m.threadReplyCalls = append(m.threadReplyCalls, threadReplyCall{threadID: threadID, payload: payload})
	if m.threadReplyErr != nil {
		return nil, m.threadReplyErr
	}
	return &discord.Message{ID: fmt.Sprintf("reply-%d", len(m.threadReplyCalls)), ChannelID: threadID}, nil
}

// --- Mock Repository for Discord ---

type mockDiscordRepo struct {
	discordThreads   map[string]models.DiscordAlertThread // checkUUID -> thread
	silenced         map[string]bool
	resolvedThreads  []string
	createdThreads   []discordCreatedThread
	resolveErr       error
	createThreadErr  error
	getThreadErr     error
}

type discordCreatedThread struct {
	checkUUID string
	channelID string
	messageID string
	threadID  string
}

func newMockDiscordRepo() *mockDiscordRepo {
	return &mockDiscordRepo{
		discordThreads: make(map[string]models.DiscordAlertThread),
		silenced:       make(map[string]bool),
	}
}

// Discord thread tracking
func (r *mockDiscordRepo) CreateDiscordThread(_ context.Context, checkUUID, channelID, messageID, threadID string) error {
	if r.createThreadErr != nil {
		return r.createThreadErr
	}
	r.createdThreads = append(r.createdThreads, discordCreatedThread{
		checkUUID: checkUUID, channelID: channelID, messageID: messageID, threadID: threadID,
	})
	r.discordThreads[checkUUID] = models.DiscordAlertThread{
		CheckUUID: checkUUID, ChannelID: channelID, MessageID: messageID, ThreadID: threadID,
	}
	return nil
}

func (r *mockDiscordRepo) GetUnresolvedDiscordThread(_ context.Context, checkUUID string) (models.DiscordAlertThread, error) {
	if r.getThreadErr != nil {
		return models.DiscordAlertThread{}, r.getThreadErr
	}
	t, ok := r.discordThreads[checkUUID]
	if !ok {
		return models.DiscordAlertThread{}, fmt.Errorf("no unresolved discord thread")
	}
	return t, nil
}

func (r *mockDiscordRepo) ResolveDiscordThread(_ context.Context, checkUUID string) error {
	if r.resolveErr != nil {
		return r.resolveErr
	}
	r.resolvedThreads = append(r.resolvedThreads, checkUUID)
	delete(r.discordThreads, checkUUID)
	return nil
}

// Required Repository interface methods (stubs)
func (r *mockDiscordRepo) Close()                                                                                          {}
func (r *mockDiscordRepo) IsCheckSilenced(_ context.Context, _ string, _ string) (bool, error) { return false, nil }
func (r *mockDiscordRepo) IsChannelSilenced(_ context.Context, checkUUID, _, _ string) (bool, error) {
	return r.silenced[checkUUID], nil
}
func (r *mockDiscordRepo) GetUnresolvedThread(_ context.Context, _ string) (models.SlackAlertThread, error) {
	return models.SlackAlertThread{}, fmt.Errorf("not found")
}
func (r *mockDiscordRepo) CreateSlackThread(_ context.Context, _, _, _, _ string) error { return nil }
func (r *mockDiscordRepo) ResolveThread(_ context.Context, _ string) error              { return nil }
func (r *mockDiscordRepo) UpdateSlackThread(_ context.Context, _, _, _ string) error    { return nil }
func (r *mockDiscordRepo) DeactivateSilence(_ context.Context, _, _ string) error       { return nil }
func (r *mockDiscordRepo) GetCheckDefinitionByUUID(_ context.Context, _ string) (models.CheckDefinition, error) {
	return models.CheckDefinition{}, fmt.Errorf("not found")
}
func (r *mockDiscordRepo) GetAllCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error)     { return nil, nil }
func (r *mockDiscordRepo) GetEnabledCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) { return nil, nil }
func (r *mockDiscordRepo) CreateCheckDefinition(_ context.Context, _ models.CheckDefinition) (string, error) { return "", nil }
func (r *mockDiscordRepo) UpdateCheckDefinition(_ context.Context, _ models.CheckDefinition) error        { return nil }
func (r *mockDiscordRepo) DeleteCheckDefinition(_ context.Context, _ string) error                        { return nil }
func (r *mockDiscordRepo) ToggleCheckDefinition(_ context.Context, _ string, _ bool) error                { return nil }
func (r *mockDiscordRepo) UpdateCheckStatus(_ context.Context, _ models.CheckStatus) error                { return nil }
func (r *mockDiscordRepo) GetAllProjects(_ context.Context) ([]string, error)                             { return nil, nil }
func (r *mockDiscordRepo) GetAllCheckTypes(_ context.Context) ([]string, error)                           { return nil, nil }
func (r *mockDiscordRepo) GetAllDefaultTimeouts() map[string]string                                       { return nil }
func (r *mockDiscordRepo) CreateSilence(_ context.Context, _ models.AlertSilence) error                   { return nil }
func (r *mockDiscordRepo) GetActiveSilences(_ context.Context) ([]models.AlertSilence, error)             { return nil, nil }
func (r *mockDiscordRepo) GetUnhealthyChecks(_ context.Context) ([]models.CheckDefinition, error)        { return nil, nil }
func (r *mockDiscordRepo) CreateAlertEvent(_ context.Context, _ models.AlertEvent) error                  { return nil }
func (r *mockDiscordRepo) ResolveAlertEvent(_ context.Context, _ string) error                            { return nil }
func (r *mockDiscordRepo) GetAlertHistory(_ context.Context, _, _ int, _ models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	return nil, 0, nil
}
func (r *mockDiscordRepo) DeactivateSilenceByID(_ context.Context, _ int) error { return nil }
func (r *mockDiscordRepo) ConvertConfigToCheckDefinitions(_ context.Context, _ *config.Config) error { return nil }
func (r *mockDiscordRepo) CountCheckDefinitions(_ context.Context) (int, error)               { return 0, nil }
func (r *mockDiscordRepo) SetMaintenanceWindow(_ context.Context, _ string, _ *time.Time) error { return nil }
func (r *mockDiscordRepo) BulkToggleCheckDefinitions(_ context.Context, _ []string, _ bool) (int64, error) { return 0, nil }
func (r *mockDiscordRepo) BulkDeleteCheckDefinitions(_ context.Context, _ []string) (int64, error) { return 0, nil }
func (r *mockDiscordRepo) GetAllEscalationPolicies(_ context.Context) ([]models.EscalationPolicy, error) { return nil, nil }
func (r *mockDiscordRepo) GetEscalationPolicyByName(_ context.Context, _ string) (models.EscalationPolicy, error) { return models.EscalationPolicy{}, nil }
func (r *mockDiscordRepo) CreateEscalationPolicy(_ context.Context, _ models.EscalationPolicy) error { return nil }
func (r *mockDiscordRepo) UpdateEscalationPolicy(_ context.Context, _ models.EscalationPolicy) error { return nil }
func (r *mockDiscordRepo) DeleteEscalationPolicy(_ context.Context, _ string) error                   { return nil }
func (r *mockDiscordRepo) GetEscalationNotifications(_ context.Context, _, _ string) ([]models.EscalationNotification, error) { return nil, nil }
func (r *mockDiscordRepo) CreateEscalationNotification(_ context.Context, _ models.EscalationNotification) error { return nil }
func (r *mockDiscordRepo) DeleteEscalationNotifications(_ context.Context, _ string) error   { return nil }
func (r *mockDiscordRepo) GetAllAlertChannels(_ context.Context) ([]models.AlertChannel, error) { return nil, nil }
func (r *mockDiscordRepo) GetAlertChannelByName(_ context.Context, _ string) (models.AlertChannel, error) { return models.AlertChannel{}, fmt.Errorf("not found") }
func (r *mockDiscordRepo) CreateAlertChannel(_ context.Context, _ models.AlertChannel) error { return nil }
func (r *mockDiscordRepo) UpdateAlertChannel(_ context.Context, _ models.AlertChannel) error { return nil }
func (r *mockDiscordRepo) DeleteAlertChannel(_ context.Context, _ string) error              { return nil }
func (r *mockDiscordRepo) CreateTelegramThread(_ context.Context, _, _ string, _ int) error  { return nil }
func (r *mockDiscordRepo) GetUnresolvedTelegramThread(_ context.Context, _ string) (models.TelegramAlertThread, error) { return models.TelegramAlertThread{}, fmt.Errorf("not found") }
func (r *mockDiscordRepo) GetTelegramThreadByMessage(_ context.Context, _ string, _ int) (models.TelegramAlertThread, error) { return models.TelegramAlertThread{}, fmt.Errorf("not found") }
func (r *mockDiscordRepo) ResolveTelegramThread(_ context.Context, _ string) error { return nil }
func (r *mockDiscordRepo) MigrateLegacyAlertFields(_ context.Context) (int, error) { return 0, nil }
func (r *mockDiscordRepo) GetSetting(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not found")
}
func (r *mockDiscordRepo) SetSetting(_ context.Context, _, _ string) error { return nil }
func (r *mockDiscordRepo) GetCheckDefaults(_ context.Context) (models.CheckDefaults, error) {
	return models.CheckDefaults{}, nil
}
func (r *mockDiscordRepo) SaveCheckDefaults(_ context.Context, _ models.CheckDefaults) error {
	return nil
}
func (r *mockDiscordRepo) InsertCheckResult(_ context.Context, _ models.CheckResult) error {
	return nil
}
func (r *mockDiscordRepo) GetUnevaluatedCycles(_ context.Context, _ int, _ time.Duration) ([]db.UnevaluatedCycle, error) {
	return nil, nil
}
func (r *mockDiscordRepo) ClaimCycleForEvaluation(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}
func (r *mockDiscordRepo) GetCycleResults(_ context.Context, _ string, _ time.Time) ([]models.CheckResult, error) {
	return nil, nil
}
func (r *mockDiscordRepo) PurgeOldCheckResults(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

// --- Tests ---

func TestDiscordSendAlert_NewFailure_CreatesNewThread(t *testing.T) {
	repo := newMockDiscordRepo()
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "connection refused"}

	alerter.SendAlert(context.Background(), checkDef, status, false)

	require.Len(t, sender.sendMessageCalls, 1, "should post a new alert message")
	assert.Equal(t, "discord-channel-1", sender.sendMessageCalls[0].channelID)
	require.Len(t, sender.createThreadCalls, 1, "should create a thread")
	assert.Len(t, sender.threadReplyCalls, 1, "should post error snapshot in thread")
	assert.Len(t, repo.createdThreads, 1, "should track the new thread in DB")
	assert.Equal(t, "check-1", repo.createdThreads[0].checkUUID)
}

func TestDiscordSendAlert_OngoingFailure_RepliesToExistingThread(t *testing.T) {
	repo := newMockDiscordRepo()
	repo.discordThreads["check-1"] = models.DiscordAlertThread{
		CheckUUID: "check-1", ChannelID: "discord-channel-1", MessageID: "msg-1", ThreadID: "thread-1",
	}
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "still failing"}

	// isNewIncident=false: ongoing failure, should reply to thread
	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.sendMessageCalls, 0, "should NOT post a new alert")
	require.Len(t, sender.threadReplyCalls, 1, "should reply to existing thread")
	assert.Equal(t, "thread-1", sender.threadReplyCalls[0].threadID)
	// Thread reply should NOT have buttons
	assert.Empty(t, sender.threadReplyCalls[0].payload.Components, "thread reply should not have buttons")
}

func TestDiscordSendAlert_ReFailure_CreatesNewThreadAfterResolvingStale(t *testing.T) {
	repo := newMockDiscordRepo()
	repo.discordThreads["check-1"] = models.DiscordAlertThread{
		CheckUUID: "check-1", ChannelID: "discord-channel-1", MessageID: "old-msg", ThreadID: "old-thread",
	}
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "new failure"}

	// isNewIncident=true: check was healthy before this failure
	alerter.SendAlert(context.Background(), checkDef, status, true)

	// Should have resolved the stale thread
	require.Len(t, repo.resolvedThreads, 1, "should resolve stale thread")
	assert.Equal(t, "check-1", repo.resolvedThreads[0])

	// Should have created a new top-level alert
	require.Len(t, sender.sendMessageCalls, 1, "should post a NEW alert")
	require.Len(t, sender.createThreadCalls, 1, "should create a new thread")
}

func TestDiscordHandleRecovery_ResolvesThread(t *testing.T) {
	repo := newMockDiscordRepo()
	repo.discordThreads["check-1"] = models.DiscordAlertThread{
		CheckUUID: "check-1", ChannelID: "discord-channel-1", MessageID: "msg-1", ThreadID: "thread-1",
	}
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	require.Len(t, sender.editMessageCalls, 1, "should edit original message to resolved")
	assert.Equal(t, "msg-1", sender.editMessageCalls[0].messageID)
	assert.Equal(t, "discord-channel-1", sender.editMessageCalls[0].channelID)
	require.Len(t, sender.threadReplyCalls, 1, "should post recovery reply in thread")
	assert.Equal(t, "thread-1", sender.threadReplyCalls[0].threadID)
	require.Len(t, repo.resolvedThreads, 1, "should resolve thread in DB")
}

func TestDiscordHandleRecovery_ResolvesThread_EvenIfDiscordFails(t *testing.T) {
	repo := newMockDiscordRepo()
	repo.discordThreads["check-1"] = models.DiscordAlertThread{
		CheckUUID: "check-1", ChannelID: "discord-channel-1", MessageID: "msg-1", ThreadID: "thread-1",
	}
	sender := &mockDiscordSender{editMessageErr: fmt.Errorf("discord API error"), threadReplyErr: fmt.Errorf("discord API error")}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	// Discord calls failed, but thread should still be resolved in DB
	require.Len(t, sender.editMessageCalls, 1, "should attempt edit")
	require.Len(t, repo.resolvedThreads, 1, "should still resolve thread in DB despite Discord failure")
}

func TestDiscordSendAlert_FullIncidentCycle(t *testing.T) {
	repo := newMockDiscordRepo()
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "error"}

	// Step 1: First failure — new incident
	alerter.SendAlert(context.Background(), checkDef, status, true)
	require.Len(t, sender.sendMessageCalls, 1, "step 1: should create new message")
	require.Len(t, sender.createThreadCalls, 1, "step 1: should create thread")

	// Step 2: Ongoing failure — reply to existing thread
	alerter.SendAlert(context.Background(), checkDef, status, false)
	assert.Len(t, sender.sendMessageCalls, 1, "step 2: should NOT create new message")
	require.Len(t, sender.threadReplyCalls, 2, "step 2: should reply to thread (snapshot + ongoing)")

	// Step 3: Recovery
	alerter.HandleRecovery(context.Background(), checkDef)
	require.Len(t, repo.resolvedThreads, 1, "step 3: should resolve thread")

	// Step 4: Re-failure — should create NEW thread
	alerter.SendAlert(context.Background(), checkDef, status, true)
	require.Len(t, sender.sendMessageCalls, 2, "step 4: should create a NEW message")
	require.Len(t, sender.createThreadCalls, 2, "step 4: should create a NEW thread")
}

func TestDiscordSendAlert_Silenced_SkipsAlert(t *testing.T) {
	repo := newMockDiscordRepo()
	repo.silenced["check-1"] = true
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "error"}

	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.sendMessageCalls, 0, "should not post alert when silenced")
	assert.Len(t, sender.threadReplyCalls, 0, "should not reply when silenced")
}

func TestDiscordSendAlert_NoChannel_SkipsAlert(t *testing.T) {
	repo := newMockDiscordRepo()
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "") // no channel configured

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "error"}

	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.sendMessageCalls, 0, "should not post alert when no channel")
}

func TestDiscordHandleRecovery_NoThread_Noop(t *testing.T) {
	repo := newMockDiscordRepo()
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	assert.Len(t, sender.editMessageCalls, 0, "should not edit anything if no thread")
	assert.Len(t, sender.threadReplyCalls, 0, "should not reply if no thread")
}

func TestDiscordOwnedTypes(t *testing.T) {
	repo := newMockDiscordRepo()
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	assert.Equal(t, []string{"discord"}, alerter.OwnedTypes())
}

func TestDiscordSendAlert_AlertMessageHasButtons(t *testing.T) {
	repo := newMockDiscordRepo()
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "connection refused"}

	alerter.SendAlert(context.Background(), checkDef, status, true)

	require.Len(t, sender.sendMessageCalls, 1)
	payload := sender.sendMessageCalls[0].payload
	require.Len(t, payload.Components, 1, "alert should have action row")
	assert.Len(t, payload.Components[0].Components, 3, "should have 3 buttons (ack, silence 1h, silence 24h)")
}

func TestDiscordHandleRecovery_EditedMessageHasNoButtons(t *testing.T) {
	repo := newMockDiscordRepo()
	repo.discordThreads["check-1"] = models.DiscordAlertThread{
		CheckUUID: "check-1", ChannelID: "discord-channel-1", MessageID: "msg-1", ThreadID: "thread-1",
	}
	sender := &mockDiscordSender{}
	alerter := newDiscordAppAlerterWithSender(sender, repo, "discord-channel-1")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	require.Len(t, sender.editMessageCalls, 1)
	payload := sender.editMessageCalls[0].payload
	assert.Empty(t, payload.Components, "resolved message should not have buttons")
}
