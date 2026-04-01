package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"
	"checker/internal/telegram"
)

// mockTelegramRepo implements db.Repository for testing Telegram handlers.
// Only the methods used by TelegramWebhookHandler are implemented; the rest panic.
type mockTelegramRepo struct {
	silences   []models.AlertSilence
	threads    map[string]models.TelegramAlertThread // key: "chatID:msgID"
	checks     map[string]models.CheckDefinition     // key: UUID
	unhealthy  []models.CheckDefinition

	// Track calls
	createdSilences      []models.AlertSilence
	deactivatedSilences  []string // "scope|target"
}

func newMockTelegramRepo() *mockTelegramRepo {
	return &mockTelegramRepo{
		threads:             make(map[string]models.TelegramAlertThread),
		checks:              make(map[string]models.CheckDefinition),
		createdSilences:     []models.AlertSilence{},
		deactivatedSilences: []string{},
	}
}

func (m *mockTelegramRepo) GetTelegramThreadByMessage(_ context.Context, chatID string, messageID int) (models.TelegramAlertThread, error) {
	key := fmt.Sprintf("%s:%d", chatID, messageID)
	t, ok := m.threads[key]
	if !ok {
		return models.TelegramAlertThread{}, fmt.Errorf("thread not found")
	}
	return t, nil
}

func (m *mockTelegramRepo) GetCheckDefinitionByUUID(_ context.Context, uuid string) (models.CheckDefinition, error) {
	c, ok := m.checks[uuid]
	if !ok {
		return models.CheckDefinition{}, fmt.Errorf("check not found")
	}
	return c, nil
}

func (m *mockTelegramRepo) CreateSilence(_ context.Context, silence models.AlertSilence) error {
	m.createdSilences = append(m.createdSilences, silence)
	return nil
}

func (m *mockTelegramRepo) GetActiveSilences(_ context.Context) ([]models.AlertSilence, error) {
	return m.silences, nil
}

func (m *mockTelegramRepo) DeactivateSilence(_ context.Context, scope, target string) error {
	m.deactivatedSilences = append(m.deactivatedSilences, scope+"|"+target)
	return nil
}

func (m *mockTelegramRepo) GetUnhealthyChecks(_ context.Context) ([]models.CheckDefinition, error) {
	return m.unhealthy, nil
}

// Stub all other Repository methods
func (m *mockTelegramRepo) Close()                                            {}
func (m *mockTelegramRepo) GetAllCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockTelegramRepo) GetEnabledCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockTelegramRepo) CreateCheckDefinition(_ context.Context, _ models.CheckDefinition) (string, error) {
	return "", nil
}
func (m *mockTelegramRepo) UpdateCheckDefinition(_ context.Context, _ models.CheckDefinition) error {
	return nil
}
func (m *mockTelegramRepo) DeleteCheckDefinition(_ context.Context, _ string) error { return nil }
func (m *mockTelegramRepo) ToggleCheckDefinition(_ context.Context, _ string, _ bool) error {
	return nil
}
func (m *mockTelegramRepo) UpdateCheckStatus(_ context.Context, _ models.CheckStatus) error {
	return nil
}
func (m *mockTelegramRepo) SetMaintenanceWindow(_ context.Context, _ string, _ *time.Time) error {
	return nil
}
func (m *mockTelegramRepo) BulkToggleCheckDefinitions(_ context.Context, _ []string, _ bool) (int64, error) {
	return 0, nil
}
func (m *mockTelegramRepo) BulkDeleteCheckDefinitions(_ context.Context, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockTelegramRepo) GetAllProjects(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockTelegramRepo) GetAllCheckTypes(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockTelegramRepo) ConvertConfigToCheckDefinitions(_ context.Context, _ *config.Config) error {
	return nil
}
func (m *mockTelegramRepo) CountCheckDefinitions(_ context.Context) (int, error)     { return 0, nil }
func (m *mockTelegramRepo) GetAllDefaultTimeouts() map[string]string                 { return nil }
func (m *mockTelegramRepo) CreateSlackThread(_ context.Context, _, _, _, _ string) error { return nil }
func (m *mockTelegramRepo) GetUnresolvedThread(_ context.Context, _ string) (models.SlackAlertThread, error) {
	return models.SlackAlertThread{}, nil
}
func (m *mockTelegramRepo) ResolveThread(_ context.Context, _ string) error            { return nil }
func (m *mockTelegramRepo) UpdateSlackThread(_ context.Context, _, _, _ string) error  { return nil }
func (m *mockTelegramRepo) CreateTelegramThread(_ context.Context, _, _ string, _ int) error {
	return nil
}
func (m *mockTelegramRepo) GetUnresolvedTelegramThread(_ context.Context, _ string) (models.TelegramAlertThread, error) {
	return models.TelegramAlertThread{}, nil
}
func (m *mockTelegramRepo) ResolveTelegramThread(_ context.Context, _ string) error { return nil }
func (m *mockTelegramRepo) CreateDiscordThread(_ context.Context, _, _, _, _ string) error { return nil }
func (m *mockTelegramRepo) GetUnresolvedDiscordThread(_ context.Context, _ string) (models.DiscordAlertThread, error) {
	return models.DiscordAlertThread{}, fmt.Errorf("not found")
}
func (m *mockTelegramRepo) ResolveDiscordThread(_ context.Context, _ string) error { return nil }
func (m *mockTelegramRepo) IsCheckSilenced(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockTelegramRepo) IsChannelSilenced(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockTelegramRepo) DeactivateSilenceByID(_ context.Context, _ int) error { return nil }
func (m *mockTelegramRepo) CreateAlertEvent(_ context.Context, _ models.AlertEvent) error {
	return nil
}
func (m *mockTelegramRepo) ResolveAlertEvent(_ context.Context, _ string) error { return nil }
func (m *mockTelegramRepo) GetAlertHistory(_ context.Context, _, _ int, _ models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	return nil, 0, nil
}
func (m *mockTelegramRepo) GetAllEscalationPolicies(_ context.Context) ([]models.EscalationPolicy, error) {
	return nil, nil
}
func (m *mockTelegramRepo) GetEscalationPolicyByName(_ context.Context, _ string) (models.EscalationPolicy, error) {
	return models.EscalationPolicy{}, nil
}
func (m *mockTelegramRepo) CreateEscalationPolicy(_ context.Context, _ models.EscalationPolicy) error {
	return nil
}
func (m *mockTelegramRepo) UpdateEscalationPolicy(_ context.Context, _ models.EscalationPolicy) error {
	return nil
}
func (m *mockTelegramRepo) DeleteEscalationPolicy(_ context.Context, _ string) error { return nil }
func (m *mockTelegramRepo) GetEscalationNotifications(_ context.Context, _, _ string) ([]models.EscalationNotification, error) {
	return nil, nil
}
func (m *mockTelegramRepo) CreateEscalationNotification(_ context.Context, _ models.EscalationNotification) error {
	return nil
}
func (m *mockTelegramRepo) DeleteEscalationNotifications(_ context.Context, _ string) error {
	return nil
}
func (m *mockTelegramRepo) GetAllAlertChannels(_ context.Context) ([]models.AlertChannel, error) {
	return nil, nil
}
func (m *mockTelegramRepo) GetAlertChannelByName(_ context.Context, _ string) (models.AlertChannel, error) {
	return models.AlertChannel{}, nil
}
func (m *mockTelegramRepo) CreateAlertChannel(_ context.Context, _ models.AlertChannel) error {
	return nil
}
func (m *mockTelegramRepo) UpdateAlertChannel(_ context.Context, _ models.AlertChannel) error {
	return nil
}
func (m *mockTelegramRepo) DeleteAlertChannel(_ context.Context, _ string) error { return nil }
func (m *mockTelegramRepo) MigrateLegacyAlertFields(_ context.Context) (int, error) {
	return 0, nil
}
func (m *mockTelegramRepo) GetSetting(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not found")
}
func (m *mockTelegramRepo) SetSetting(_ context.Context, _, _ string) error { return nil }
func (m *mockTelegramRepo) GetCheckDefaults(_ context.Context) (models.CheckDefaults, error) {
	return models.CheckDefaults{}, nil
}
func (m *mockTelegramRepo) SaveCheckDefaults(_ context.Context, _ models.CheckDefaults) error {
	return nil
}
func (m *mockTelegramRepo) InsertCheckResult(_ context.Context, _ models.CheckResult) error {
	return nil
}
func (m *mockTelegramRepo) GetUnevaluatedCycles(_ context.Context, _ int, _ time.Duration) ([]db.UnevaluatedCycle, error) {
	return nil, nil
}
func (m *mockTelegramRepo) ClaimCycleForEvaluation(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}
func (m *mockTelegramRepo) GetCycleResults(_ context.Context, _ string, _ time.Time) ([]models.CheckResult, error) {
	return nil, nil
}
func (m *mockTelegramRepo) PurgeOldCheckResults(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

// mockTelegramClient wraps a test HTTP server to capture API calls.
type mockTelegramClient struct {
	server *httptest.Server
	client *telegram.TelegramClient
	calls  []apiCall
}

type apiCall struct {
	Method string
	Body   map[string]interface{}
}

func newMockTelegramClient(secretToken string) *mockTelegramClient {
	m := &mockTelegramClient{}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Extract method from path: /bot<token>/<method>
		parts := strings.SplitN(r.URL.Path, "/", 3)
		method := ""
		if len(parts) >= 3 {
			method = parts[2]
		}
		m.calls = append(m.calls, apiCall{Method: method, Body: body})

		// Return a valid Telegram API response
		resp := map[string]interface{}{
			"ok":     true,
			"result": map[string]interface{}{"message_id": 999, "chat": map[string]interface{}{"id": 123}},
		}
		json.NewEncoder(w).Encode(resp)
	}))

	// Create a real TelegramClient but pointed at our mock server
	m.client = telegram.NewTelegramClientWithBaseURL("test-bot-token", secretToken, "123", m.server.URL+"/bottest-bot-token")

	return m
}

func (m *mockTelegramClient) close() {
	m.server.Close()
}

func (m *mockTelegramClient) findCall(method string) *apiCall {
	for _, c := range m.calls {
		if c.Method == method {
			return &c
		}
	}
	return nil
}

// --- Helper to build webhook request ---

func buildTelegramWebhookRequest(t *testing.T, secretToken string, update telegram.Update) *http.Request {
	t.Helper()
	body, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal update: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretToken)
	return req
}

// --- Tests ---

func TestTelegramWebhook_InvalidSecretToken(t *testing.T) {
	repo := newMockTelegramRepo()
	handler := NewTelegramWebhookHandler("my-secret", nil, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader("{}"))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-secret")
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestTelegramWebhook_ValidSecretToken_EmptyUpdate(t *testing.T) {
	repo := newMockTelegramRepo()
	handler := NewTelegramWebhookHandler("my-secret", nil, repo)

	req := buildTelegramWebhookRequest(t, "my-secret", telegram.Update{})
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestTelegramWebhook_CallbackSilenceCheck(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.threads["123:42"] = models.TelegramAlertThread{CheckUUID: "check-uuid-1", ChatID: "123", MessageID: 42}
	repo.checks["check-uuid-1"] = models.CheckDefinition{UUID: "check-uuid-1", Name: "Test Check", Project: "myproject", GroupName: "prod", Type: "http"}

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-1",
			From: telegram.User{ID: 100, Username: "testuser", FirstName: "Test"},
			Message: &telegram.IncomingMessage{
				MessageID: 42,
				Chat:      telegram.Chat{ID: 123},
			},
			Data: "s|1h",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify silence was created
	if len(repo.createdSilences) != 1 {
		t.Fatalf("expected 1 silence created, got %d", len(repo.createdSilences))
	}
	s := repo.createdSilences[0]
	if s.Scope != "check" || s.Target != "check-uuid-1" {
		t.Errorf("unexpected silence: scope=%s target=%s", s.Scope, s.Target)
	}
	if s.ExpiresAt == nil {
		t.Error("expected expiry to be set for 1h silence")
	}

	// Verify callback was answered
	answerCall := mock.findCall("answerCallbackQuery")
	if answerCall == nil {
		t.Error("expected answerCallbackQuery call")
	}
}

func TestTelegramWebhook_CallbackAck(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.threads["123:42"] = models.TelegramAlertThread{CheckUUID: "check-uuid-1", ChatID: "123", MessageID: 42}
	repo.checks["check-uuid-1"] = models.CheckDefinition{UUID: "check-uuid-1", Name: "Test Check", Project: "myproject", Type: "http"}

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-2",
			From: telegram.User{ID: 100, Username: "testuser", FirstName: "Test"},
			Message: &telegram.IncomingMessage{
				MessageID: 42,
				Chat:      telegram.Chat{ID: 123},
			},
			Data: "ack",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify ack reply was sent (sendMessage call)
	sendCall := mock.findCall("sendMessage")
	if sendCall == nil {
		t.Error("expected sendMessage call for ack reply")
	}

	// Verify message was edited
	editCall := mock.findCall("editMessageText")
	if editCall == nil {
		t.Error("expected editMessageText call for ack badge")
	}
}

func TestTelegramWebhook_CallbackUnknownData(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.threads["123:42"] = models.TelegramAlertThread{CheckUUID: "check-uuid-1", ChatID: "123", MessageID: 42}

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-3",
			From: telegram.User{ID: 100, FirstName: "Test"},
			Message: &telegram.IncomingMessage{
				MessageID: 42,
				Chat:      telegram.Chat{ID: 123},
			},
			Data: "unknown_action",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Should answer callback with error text
	answerCall := mock.findCall("answerCallbackQuery")
	if answerCall == nil {
		t.Error("expected answerCallbackQuery call for unknown action")
	}
}

func TestTelegramWebhook_StatusCommand(t *testing.T) {
	repo := newMockTelegramRepo()
	repo.unhealthy = []models.CheckDefinition{
		{Name: "API Check", Project: "myproject", GroupName: "prod", LastMessage: "connection refused"},
	}

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		Message: &telegram.IncomingMessage{
			MessageID: 10,
			Chat:      telegram.Chat{ID: 123},
			Text:      "/status",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	sendCall := mock.findCall("sendMessage")
	if sendCall == nil {
		t.Fatal("expected sendMessage call for /status reply")
	}
	text, _ := sendCall.Body["text"].(string)
	if !strings.Contains(text, "API Check") {
		t.Errorf("expected status reply to contain check name, got: %s", text)
	}
}

func TestTelegramWebhook_StatusCommand_AllHealthy(t *testing.T) {
	repo := newMockTelegramRepo()

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		Message: &telegram.IncomingMessage{
			MessageID: 10,
			Chat:      telegram.Chat{ID: 123},
			Text:      "/status",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	sendCall := mock.findCall("sendMessage")
	if sendCall == nil {
		t.Fatal("expected sendMessage call")
	}
	text, _ := sendCall.Body["text"].(string)
	if !strings.Contains(text, "All checks healthy") {
		t.Errorf("expected 'All checks healthy', got: %s", text)
	}
}

func TestTelegramWebhook_SilenceCheckCommand(t *testing.T) {
	repo := newMockTelegramRepo()

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		Message: &telegram.IncomingMessage{
			MessageID: 10,
			Chat:      telegram.Chat{ID: 123},
			Text:      "/silence check abc-123 1h",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if len(repo.createdSilences) != 1 {
		t.Fatalf("expected 1 silence, got %d", len(repo.createdSilences))
	}

	s := repo.createdSilences[0]
	if s.Scope != "check" || s.Target != "abc-123" {
		t.Errorf("unexpected silence: scope=%s target=%s", s.Scope, s.Target)
	}
}

func TestTelegramWebhook_SilenceListCommand(t *testing.T) {
	expires := time.Now().Add(1 * time.Hour)
	repo := newMockTelegramRepo()
	repo.silences = []models.AlertSilence{
		{Scope: "check", Target: "uuid-1", SilencedBy: "@admin", ExpiresAt: &expires, Active: true},
	}

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		Message: &telegram.IncomingMessage{
			MessageID: 10,
			Chat:      telegram.Chat{ID: 123},
			Text:      "/silence list",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	sendCall := mock.findCall("sendMessage")
	if sendCall == nil {
		t.Fatal("expected sendMessage call")
	}
	text, _ := sendCall.Body["text"].(string)
	if !strings.Contains(text, "uuid-1") {
		t.Errorf("expected silence list to contain target, got: %s", text)
	}
}

func TestTelegramWebhook_UnsilenceCommand(t *testing.T) {
	repo := newMockTelegramRepo()

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		Message: &telegram.IncomingMessage{
			MessageID: 10,
			Chat:      telegram.Chat{ID: 123},
			Text:      "/unsilence check abc-123",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if len(repo.deactivatedSilences) != 1 {
		t.Fatalf("expected 1 deactivation, got %d", len(repo.deactivatedSilences))
	}
	if repo.deactivatedSilences[0] != "check|abc-123" {
		t.Errorf("unexpected deactivation: %s", repo.deactivatedSilences[0])
	}
}

func TestTelegramWebhook_HelpCommand(t *testing.T) {
	repo := newMockTelegramRepo()

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		Message: &telegram.IncomingMessage{
			MessageID: 10,
			Chat:      telegram.Chat{ID: 123},
			Text:      "/help",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	sendCall := mock.findCall("sendMessage")
	if sendCall == nil {
		t.Fatal("expected sendMessage call")
	}
	text, _ := sendCall.Body["text"].(string)
	if !strings.Contains(text, "Available commands") {
		t.Errorf("expected help text, got: %s", text)
	}
}

func TestTelegramWebhook_AlwaysReturns200(t *testing.T) {
	repo := newMockTelegramRepo()
	handler := NewTelegramWebhookHandler("my-secret", nil, repo)

	// Test with invalid JSON body
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader("not json"))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "my-secret")
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for invalid JSON, got %d", w.Code)
	}
}

func TestTelegramWebhook_CallbackThreadNotFound(t *testing.T) {
	repo := newMockTelegramRepo()
	// No threads registered

	mock := newMockTelegramClient("my-secret")
	defer mock.close()

	handler := NewTelegramWebhookHandler("my-secret", mock.client, repo)

	update := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cb-5",
			From: telegram.User{ID: 100, FirstName: "Test"},
			Message: &telegram.IncomingMessage{
				MessageID: 99,
				Chat:      telegram.Chat{ID: 123},
			},
			Data: "s|1h",
		},
	}

	req := buildTelegramWebhookRequest(t, "my-secret", update)
	w := httptest.NewRecorder()

	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Should answer callback with "Alert not found"
	answerCall := mock.findCall("answerCallbackQuery")
	if answerCall == nil {
		t.Error("expected answerCallbackQuery call")
	} else {
		text, _ := answerCall.Body["text"].(string)
		if text != "Alert not found" {
			t.Errorf("expected 'Alert not found', got: %s", text)
		}
	}
}

func TestParseTelegramDuration(t *testing.T) {
	tests := []struct {
		input      string
		wantDur    time.Duration
		wantLabel  string
		wantIndef  bool
	}{
		{"30m", 30 * time.Minute, "30m", false},
		{"1h", 1 * time.Hour, "1h", false},
		{"4h", 4 * time.Hour, "4h", false},
		{"8h", 8 * time.Hour, "8h", false},
		{"24h", 24 * time.Hour, "24h", false},
		{"indef", 0, "indefinitely", true},
		{"indefinite", 0, "indefinitely", true},
		{"unknown", 1 * time.Hour, "1h", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dur, label, indef := parseTelegramDuration(tt.input)
			if dur != tt.wantDur {
				t.Errorf("duration: got %v, want %v", dur, tt.wantDur)
			}
			if label != tt.wantLabel {
				t.Errorf("label: got %q, want %q", label, tt.wantLabel)
			}
			if indef != tt.wantIndef {
				t.Errorf("indefinite: got %v, want %v", indef, tt.wantIndef)
			}
		})
	}
}
