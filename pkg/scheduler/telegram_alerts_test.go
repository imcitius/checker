package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"checker/pkg/models"
	"checker/internal/telegram"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ AppAlerter = (*TelegramAppAlerter)(nil)

// --- Mock Telegram Sender ---

type mockTelegramSender struct {
	postAlertCalls     []tgPostAlertCall
	postReplyCalls     []tgPostReplyCall
	sendResolveCalls   []tgSendResolveCall
	errorSnapshotCalls []tgErrorSnapshotCall
	postAlertErr       error
	postReplyErr       error
	sendResolveErr     error
	errorSnapshotErr   error
	postAlertMsgID     int
	postReplyMsgID     int
}

type tgPostAlertCall struct {
	chatID string
	info   telegram.CheckAlertInfo
}

type tgPostReplyCall struct {
	chatID       string
	replyToMsgID int
	info         telegram.CheckAlertInfo
}

type tgSendResolveCall struct {
	info          telegram.CheckAlertInfo
	originalMsgID int
	chatID        string
}

type tgErrorSnapshotCall struct {
	chatID       string
	replyToMsgID int
	info         telegram.CheckAlertInfo
}

func (m *mockTelegramSender) PostAlert(_ context.Context, chatID string, info telegram.CheckAlertInfo) (int, error) {
	m.postAlertCalls = append(m.postAlertCalls, tgPostAlertCall{chatID: chatID, info: info})
	msgID := m.postAlertMsgID
	if msgID == 0 {
		msgID = 1000 + len(m.postAlertCalls)
	}
	return msgID, m.postAlertErr
}

func (m *mockTelegramSender) PostAlertReply(_ context.Context, chatID string, replyToMsgID int, info telegram.CheckAlertInfo) (int, error) {
	m.postReplyCalls = append(m.postReplyCalls, tgPostReplyCall{chatID: chatID, replyToMsgID: replyToMsgID, info: info})
	msgID := m.postReplyMsgID
	if msgID == 0 {
		msgID = 2000 + len(m.postReplyCalls)
	}
	return msgID, m.postReplyErr
}

func (m *mockTelegramSender) SendResolve(_ context.Context, info telegram.CheckAlertInfo, originalMsgID int, chatID string) error {
	m.sendResolveCalls = append(m.sendResolveCalls, tgSendResolveCall{info: info, originalMsgID: originalMsgID, chatID: chatID})
	return m.sendResolveErr
}

func (m *mockTelegramSender) PostErrorSnapshotReply(_ context.Context, chatID string, replyToMsgID int, info telegram.CheckAlertInfo) (int, error) {
	m.errorSnapshotCalls = append(m.errorSnapshotCalls, tgErrorSnapshotCall{chatID: chatID, replyToMsgID: replyToMsgID, info: info})
	return 3000 + len(m.errorSnapshotCalls), m.errorSnapshotErr
}

// --- Mock Telegram Repository ---

type mockTelegramRepo struct {
	mockRepo // embed the existing mockRepo for shared interface methods
	telegramThreads         map[string]models.TelegramAlertThread // checkUUID -> thread
	resolvedTelegramThreads []string                              // checkUUIDs that had ResolveTelegramThread called
	createdTelegramThreads  []createdTelegramThread
	resolveTelegramErr      error
	createTelegramErr       error
	getTelegramThreadErr    error
}

type createdTelegramThread struct {
	checkUUID string
	chatID    string
	messageID int
}

func newMockTelegramRepo() *mockTelegramRepo {
	return &mockTelegramRepo{
		mockRepo: mockRepo{
			threads:   make(map[string]models.SlackAlertThread),
			silenced:  make(map[string]bool),
			checkDefs: make(map[string]models.CheckDefinition),
		},
		telegramThreads: make(map[string]models.TelegramAlertThread),
	}
}

// Override Telegram-specific methods on mockTelegramRepo.

func (r *mockTelegramRepo) CreateTelegramThread(_ context.Context, checkUUID, chatID string, messageID int) error {
	if r.createTelegramErr != nil {
		return r.createTelegramErr
	}
	r.createdTelegramThreads = append(r.createdTelegramThreads, createdTelegramThread{
		checkUUID: checkUUID, chatID: chatID, messageID: messageID,
	})
	r.telegramThreads[checkUUID] = models.TelegramAlertThread{
		CheckUUID: checkUUID, ChatID: chatID, MessageID: messageID,
	}
	return nil
}

func (r *mockTelegramRepo) GetUnresolvedTelegramThread(_ context.Context, checkUUID string) (models.TelegramAlertThread, error) {
	if r.getTelegramThreadErr != nil {
		return models.TelegramAlertThread{}, r.getTelegramThreadErr
	}
	t, ok := r.telegramThreads[checkUUID]
	if !ok {
		return models.TelegramAlertThread{}, fmt.Errorf("no unresolved telegram thread")
	}
	return t, nil
}

func (r *mockTelegramRepo) ResolveTelegramThread(_ context.Context, checkUUID string) error {
	if r.resolveTelegramErr != nil {
		return r.resolveTelegramErr
	}
	r.resolvedTelegramThreads = append(r.resolvedTelegramThreads, checkUUID)
	delete(r.telegramThreads, checkUUID)
	return nil
}

func (r *mockTelegramRepo) GetTelegramThreadByMessage(_ context.Context, _ string, _ int) (models.TelegramAlertThread, error) {
	return models.TelegramAlertThread{}, fmt.Errorf("not found")
}
func (r *mockTelegramRepo) CreateDiscordThread(_ context.Context, _, _, _, _ string) error { return nil }
func (r *mockTelegramRepo) GetUnresolvedDiscordThread(_ context.Context, _ string) (models.DiscordAlertThread, error) {
	return models.DiscordAlertThread{}, fmt.Errorf("not found")
}
func (r *mockTelegramRepo) ResolveDiscordThread(_ context.Context, _ string) error { return nil }

// --- Tests ---

func TestTelegramSendAlert_NewFailure_CreatesNewThread(t *testing.T) {
	repo := newMockTelegramRepo()
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "connection refused"}

	alerter.SendAlert(context.Background(), checkDef, status, false)

	require.Len(t, sender.postAlertCalls, 1, "should post a new alert")
	assert.Equal(t, "chat-123", sender.postAlertCalls[0].chatID)
	assert.Len(t, sender.postReplyCalls, 0, "should not send a reply")
	assert.Len(t, repo.createdTelegramThreads, 1, "should track the new thread")
	assert.Equal(t, "check-1", repo.createdTelegramThreads[0].checkUUID)
}

func TestTelegramSendAlert_OngoingFailure_RepliesToExistingThread(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.telegramThreads["check-1"] = models.TelegramAlertThread{
		CheckUUID: "check-1", ChatID: "chat-123", MessageID: 42,
	}
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "still failing"}

	// isNewIncident=false: ongoing failure, should reply to thread
	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.postAlertCalls, 0, "should NOT post a new alert")
	require.Len(t, sender.postReplyCalls, 1, "should reply to existing thread")
	assert.Equal(t, 42, sender.postReplyCalls[0].replyToMsgID)
}

func TestTelegramSendAlert_ReFailure_CreatesNewThreadAfterResolvingStale(t *testing.T) {
	repo := newMockTelegramRepo()
	// Simulate a stale unresolved thread from the previous incident
	repo.telegramThreads["check-1"] = models.TelegramAlertThread{
		CheckUUID: "check-1", ChatID: "chat-123", MessageID: 99,
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "new failure"}

	// isNewIncident=true: check was healthy before this failure
	alerter.SendAlert(context.Background(), checkDef, status, true)

	// Should have resolved the stale thread
	require.Len(t, repo.resolvedTelegramThreads, 1, "should resolve stale thread")
	assert.Equal(t, "check-1", repo.resolvedTelegramThreads[0])

	// Should have created a new top-level alert (NOT replied to old thread)
	require.Len(t, sender.postAlertCalls, 1, "should post a NEW alert")
	assert.Len(t, sender.postReplyCalls, 0, "should NOT reply to old thread")

	// Should have tracked the new thread
	require.Len(t, repo.createdTelegramThreads, 1, "should track the new thread")
}

func TestTelegramHandleRecovery_ResolvesThread(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.telegramThreads["check-1"] = models.TelegramAlertThread{
		CheckUUID: "check-1", ChatID: "chat-123", MessageID: 42,
	}
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	require.Len(t, sender.sendResolveCalls, 1, "should send resolve to Telegram")
	assert.Equal(t, 42, sender.sendResolveCalls[0].originalMsgID)
	require.Len(t, repo.resolvedTelegramThreads, 1, "should resolve thread in DB")
}

func TestTelegramHandleRecovery_ResolvesThread_EvenIfTelegramFails(t *testing.T) {
	// Verify that DB thread is resolved even if the Telegram API call fails.
	// This is critical for preventing stale threads.
	repo := newMockTelegramRepo()
	repo.telegramThreads["check-1"] = models.TelegramAlertThread{
		CheckUUID: "check-1", ChatID: "chat-123", MessageID: 42,
	}
	sender := &mockTelegramSender{sendResolveErr: fmt.Errorf("telegram API error")}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	// Telegram call failed, but thread should still be resolved in DB
	require.Len(t, sender.sendResolveCalls, 1, "should attempt Telegram resolve")
	require.Len(t, repo.resolvedTelegramThreads, 1, "should still resolve thread in DB despite Telegram failure")
}

func TestTelegramSendAlert_Silenced_SkipsAlert(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.silenced["check-1"] = true
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "error"}

	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.postAlertCalls, 0, "should not post alert when silenced")
	assert.Len(t, sender.postReplyCalls, 0, "should not reply when silenced")
}

func TestTelegramSendAlert_FullIncidentCycle(t *testing.T) {
	// Simulate a complete incident lifecycle:
	// 1. Check fails → new thread
	// 2. Check still fails → reply to thread
	// 3. Check recovers → thread resolved
	// 4. Check fails again → NEW thread (not reply to old)

	repo := newMockTelegramRepo()
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "error"}

	// Step 1: First failure — new incident
	alerter.SendAlert(context.Background(), checkDef, status, true)
	require.Len(t, sender.postAlertCalls, 1, "step 1: should create new thread")
	assert.Len(t, sender.postReplyCalls, 0)

	// Step 2: Ongoing failure — reply to existing thread
	alerter.SendAlert(context.Background(), checkDef, status, false)
	assert.Len(t, sender.postAlertCalls, 1, "step 2: should NOT create new thread")
	require.Len(t, sender.postReplyCalls, 1, "step 2: should reply to thread")

	// Step 3: Recovery
	alerter.HandleRecovery(context.Background(), checkDef)
	require.Len(t, repo.resolvedTelegramThreads, 1, "step 3: should resolve thread")

	// Step 4: Re-failure — should create NEW thread
	alerter.SendAlert(context.Background(), checkDef, status, true)
	require.Len(t, sender.postAlertCalls, 2, "step 4: should create a NEW thread")
	assert.Len(t, sender.postReplyCalls, 1, "step 4: should NOT add more replies")
}

func TestTelegramSendAlert_PostsErrorSnapshotOnNewAlert(t *testing.T) {
	repo := newMockTelegramRepo()
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "connection refused"}

	alerter.SendAlert(context.Background(), checkDef, status, true)

	require.Len(t, sender.errorSnapshotCalls, 1, "should post error snapshot")
	assert.Equal(t, "chat-123", sender.errorSnapshotCalls[0].chatID)
	assert.Equal(t, "connection refused", sender.errorSnapshotCalls[0].info.Message)
}

func TestTelegramSendAlert_NoErrorSnapshotOnReply(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.telegramThreads["check-1"] = models.TelegramAlertThread{
		CheckUUID: "check-1", ChatID: "chat-123", MessageID: 42,
	}
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}
	status := models.CheckStatus{Message: "still failing"}

	// Ongoing failure — should NOT post error snapshot
	alerter.SendAlert(context.Background(), checkDef, status, false)

	assert.Len(t, sender.errorSnapshotCalls, 0, "should NOT post error snapshot for replies")
}

func TestTelegramHandleRecovery_PassesOriginalErrorToResolve(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.telegramThreads["check-1"] = models.TelegramAlertThread{
		CheckUUID: "check-1", ChatID: "chat-123", MessageID: 42,
	}
	sender := &mockTelegramSender{}
	alerter := newTelegramAppAlerterWithSender(sender, repo, "chat-123")

	checkDef := models.CheckDefinition{UUID: "check-1", Name: "test-check", Project: "proj"}

	alerter.HandleRecovery(context.Background(), checkDef)

	require.Len(t, sender.sendResolveCalls, 1, "should send resolve to Telegram")
	// The mock repo returns empty alert history, so OriginalError should be empty
	assert.Equal(t, "", sender.sendResolveCalls[0].info.OriginalError)
}
