package web

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"checker/internal/config"
	"checker/internal/discord"
	"checker/internal/models"
)

// --- Mock repository for Discord handler tests ---

type mockDiscordRepo struct {
	silences            []models.AlertSilence
	createdSilences     []models.AlertSilence
	deactivatedSilences []string // "scope|target"
	threads             map[string]models.DiscordAlertThread
}

func newMockDiscordRepo() *mockDiscordRepo {
	return &mockDiscordRepo{
		createdSilences:     []models.AlertSilence{},
		deactivatedSilences: []string{},
		threads:             make(map[string]models.DiscordAlertThread),
	}
}

func (m *mockDiscordRepo) CreateSilence(_ context.Context, silence models.AlertSilence) error {
	m.createdSilences = append(m.createdSilences, silence)
	return nil
}

func (m *mockDiscordRepo) DeactivateSilence(_ context.Context, scope, target string) error {
	m.deactivatedSilences = append(m.deactivatedSilences, scope+"|"+target)
	return nil
}

func (m *mockDiscordRepo) GetUnresolvedDiscordThread(_ context.Context, checkUUID string) (models.DiscordAlertThread, error) {
	t, ok := m.threads[checkUUID]
	if !ok {
		return models.DiscordAlertThread{}, fmt.Errorf("not found")
	}
	return t, nil
}

func (m *mockDiscordRepo) GetActiveSilences(_ context.Context) ([]models.AlertSilence, error) {
	return m.silences, nil
}

// Stub all other Repository methods
func (m *mockDiscordRepo) Close()                                            {}
func (m *mockDiscordRepo) GetAllCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockDiscordRepo) GetEnabledCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockDiscordRepo) GetCheckDefinitionByUUID(_ context.Context, _ string) (models.CheckDefinition, error) {
	return models.CheckDefinition{}, nil
}
func (m *mockDiscordRepo) CreateCheckDefinition(_ context.Context, _ models.CheckDefinition) (string, error) {
	return "", nil
}
func (m *mockDiscordRepo) UpdateCheckDefinition(_ context.Context, _ models.CheckDefinition) error {
	return nil
}
func (m *mockDiscordRepo) DeleteCheckDefinition(_ context.Context, _ string) error { return nil }
func (m *mockDiscordRepo) ToggleCheckDefinition(_ context.Context, _ string, _ bool) error {
	return nil
}
func (m *mockDiscordRepo) UpdateCheckStatus(_ context.Context, _ models.CheckStatus) error {
	return nil
}
func (m *mockDiscordRepo) SetMaintenanceWindow(_ context.Context, _ string, _ *time.Time) error {
	return nil
}
func (m *mockDiscordRepo) BulkToggleCheckDefinitions(_ context.Context, _ []string, _ bool) (int64, error) {
	return 0, nil
}
func (m *mockDiscordRepo) BulkDeleteCheckDefinitions(_ context.Context, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockDiscordRepo) GetAllProjects(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockDiscordRepo) GetAllCheckTypes(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockDiscordRepo) ConvertConfigToCheckDefinitions(_ context.Context, _ *config.Config) error {
	return nil
}
func (m *mockDiscordRepo) CountCheckDefinitions(_ context.Context) (int, error)     { return 0, nil }
func (m *mockDiscordRepo) GetAllDefaultTimeouts() map[string]string                 { return nil }
func (m *mockDiscordRepo) CreateSlackThread(_ context.Context, _, _, _, _ string) error { return nil }
func (m *mockDiscordRepo) GetUnresolvedThread(_ context.Context, _ string) (models.SlackAlertThread, error) {
	return models.SlackAlertThread{}, nil
}
func (m *mockDiscordRepo) ResolveThread(_ context.Context, _ string) error            { return nil }
func (m *mockDiscordRepo) UpdateSlackThread(_ context.Context, _, _, _ string) error  { return nil }
func (m *mockDiscordRepo) CreateTelegramThread(_ context.Context, _, _ string, _ int) error {
	return nil
}
func (m *mockDiscordRepo) GetUnresolvedTelegramThread(_ context.Context, _ string) (models.TelegramAlertThread, error) {
	return models.TelegramAlertThread{}, nil
}
func (m *mockDiscordRepo) ResolveTelegramThread(_ context.Context, _ string) error { return nil }
func (m *mockDiscordRepo) GetTelegramThreadByMessage(_ context.Context, _ string, _ int) (models.TelegramAlertThread, error) {
	return models.TelegramAlertThread{}, nil
}
func (m *mockDiscordRepo) CreateDiscordThread(_ context.Context, _, _, _, _ string) error { return nil }
func (m *mockDiscordRepo) ResolveDiscordThread(_ context.Context, _ string) error { return nil }
func (m *mockDiscordRepo) IsCheckSilenced(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockDiscordRepo) IsChannelSilenced(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockDiscordRepo) DeactivateSilenceByID(_ context.Context, _ int) error { return nil }
func (m *mockDiscordRepo) CreateAlertEvent(_ context.Context, _ models.AlertEvent) error {
	return nil
}
func (m *mockDiscordRepo) ResolveAlertEvent(_ context.Context, _ string) error { return nil }
func (m *mockDiscordRepo) GetAlertHistory(_ context.Context, _, _ int, _ models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	return nil, 0, nil
}
func (m *mockDiscordRepo) GetUnhealthyChecks(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (m *mockDiscordRepo) GetAllEscalationPolicies(_ context.Context) ([]models.EscalationPolicy, error) {
	return nil, nil
}
func (m *mockDiscordRepo) GetEscalationPolicyByName(_ context.Context, _ string) (models.EscalationPolicy, error) {
	return models.EscalationPolicy{}, nil
}
func (m *mockDiscordRepo) CreateEscalationPolicy(_ context.Context, _ models.EscalationPolicy) error {
	return nil
}
func (m *mockDiscordRepo) UpdateEscalationPolicy(_ context.Context, _ models.EscalationPolicy) error {
	return nil
}
func (m *mockDiscordRepo) DeleteEscalationPolicy(_ context.Context, _ string) error { return nil }
func (m *mockDiscordRepo) GetEscalationNotifications(_ context.Context, _, _ string) ([]models.EscalationNotification, error) {
	return nil, nil
}
func (m *mockDiscordRepo) CreateEscalationNotification(_ context.Context, _ models.EscalationNotification) error {
	return nil
}
func (m *mockDiscordRepo) DeleteEscalationNotifications(_ context.Context, _ string) error {
	return nil
}
func (m *mockDiscordRepo) GetAllAlertChannels(_ context.Context) ([]models.AlertChannel, error) {
	return nil, nil
}
func (m *mockDiscordRepo) GetAlertChannelByName(_ context.Context, _ string) (models.AlertChannel, error) {
	return models.AlertChannel{}, nil
}
func (m *mockDiscordRepo) CreateAlertChannel(_ context.Context, _ models.AlertChannel) error {
	return nil
}
func (m *mockDiscordRepo) UpdateAlertChannel(_ context.Context, _ models.AlertChannel) error {
	return nil
}
func (m *mockDiscordRepo) DeleteAlertChannel(_ context.Context, _ string) error { return nil }
func (m *mockDiscordRepo) MigrateLegacyAlertFields(_ context.Context) (int, error) {
	return 0, nil
}
func (m *mockDiscordRepo) GetSetting(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not found")
}
func (m *mockDiscordRepo) SetSetting(_ context.Context, _, _ string) error { return nil }
func (m *mockDiscordRepo) GetCheckDefaults(_ context.Context) (models.CheckDefaults, error) {
	return models.CheckDefaults{}, nil
}
func (m *mockDiscordRepo) SaveCheckDefaults(_ context.Context, _ models.CheckDefaults) error {
	return nil
}

// --- Test helpers ---

// generateTestKeyPair generates an Ed25519 key pair and returns the public key hex and the private key.
func generateTestKeyPair(t *testing.T) (string, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}
	return hex.EncodeToString(pub), priv
}

// signDiscordRequest signs a Discord interaction request body with the given private key.
func signDiscordRequest(t *testing.T, privKey ed25519.PrivateKey, timestamp string, body []byte) string {
	t.Helper()
	msg := []byte(timestamp + string(body))
	sig := ed25519.Sign(privKey, msg)
	return hex.EncodeToString(sig)
}

// buildSignedDiscordRequest creates an HTTP request with valid Discord signature headers.
func buildSignedDiscordRequest(t *testing.T, privKey ed25519.PrivateKey, body []byte) *http.Request {
	t.Helper()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := signDiscordRequest(t, privKey, timestamp, body)

	req := httptest.NewRequest(http.MethodPost, "/api/discord/interactions", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-Ed25519", signature)
	req.Header.Set("X-Signature-Timestamp", timestamp)
	return req
}

// --- Tests ---

func TestVerifyDiscordSignature_Valid(t *testing.T) {
	pubHex, privKey := generateTestKeyPair(t)
	pubBytes, _ := hex.DecodeString(pubHex)
	pubKey := ed25519.PublicKey(pubBytes)

	body := []byte(`{"type":1}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := signDiscordRequest(t, privKey, timestamp, body)

	req := httptest.NewRequest(http.MethodPost, "/api/discord/interactions", nil)
	req.Header.Set("X-Signature-Ed25519", signature)
	req.Header.Set("X-Signature-Timestamp", timestamp)

	if !VerifyDiscordSignature(pubKey, req, body) {
		t.Error("expected valid signature to be accepted")
	}
}

func TestVerifyDiscordSignature_InvalidSignature(t *testing.T) {
	pubHex, _ := generateTestKeyPair(t)
	pubBytes, _ := hex.DecodeString(pubHex)
	pubKey := ed25519.PublicKey(pubBytes)

	body := []byte(`{"type":1}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	req := httptest.NewRequest(http.MethodPost, "/api/discord/interactions", nil)
	req.Header.Set("X-Signature-Ed25519", "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	req.Header.Set("X-Signature-Timestamp", timestamp)

	if VerifyDiscordSignature(pubKey, req, body) {
		t.Error("expected invalid signature to be rejected")
	}
}

func TestVerifyDiscordSignature_MissingHeaders(t *testing.T) {
	pubHex, _ := generateTestKeyPair(t)
	pubBytes, _ := hex.DecodeString(pubHex)
	pubKey := ed25519.PublicKey(pubBytes)

	body := []byte(`{"type":1}`)

	tests := []struct {
		name      string
		signature string
		timestamp string
	}{
		{"missing both", "", ""},
		{"missing signature", "", fmt.Sprintf("%d", time.Now().Unix())},
		{"missing timestamp", "abcd", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.signature != "" {
				req.Header.Set("X-Signature-Ed25519", tt.signature)
			}
			if tt.timestamp != "" {
				req.Header.Set("X-Signature-Timestamp", tt.timestamp)
			}
			if VerifyDiscordSignature(pubKey, req, body) {
				t.Error("expected missing headers to be rejected")
			}
		})
	}
}

func TestVerifyDiscordSignature_EmptyPublicKey(t *testing.T) {
	body := []byte(`{"type":1}`)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Signature-Ed25519", "abcd")
	req.Header.Set("X-Signature-Timestamp", "12345")

	if VerifyDiscordSignature(nil, req, body) {
		t.Error("expected empty public key to be rejected")
	}
}

func TestDiscordInteraction_Ping(t *testing.T) {
	pubHex, privKey := generateTestKeyPair(t)
	repo := newMockDiscordRepo()
	handler := NewDiscordInteractionHandler(pubHex, nil, repo)

	body := []byte(`{"type":1}`)
	req := buildSignedDiscordRequest(t, privKey, body)
	w := httptest.NewRecorder()

	handler.HandleInteraction(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["type"] != float64(1) {
		t.Errorf("expected PONG response type 1, got %v", resp["type"])
	}
}

func TestDiscordInteraction_InvalidSignature(t *testing.T) {
	pubHex, _ := generateTestKeyPair(t)
	repo := newMockDiscordRepo()
	handler := NewDiscordInteractionHandler(pubHex, nil, repo)

	body := []byte(`{"type":1}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	req := httptest.NewRequest(http.MethodPost, "/api/discord/interactions", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-Ed25519", "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	req.Header.Set("X-Signature-Timestamp", timestamp)
	w := httptest.NewRecorder()

	handler.HandleInteraction(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDiscordInteraction_Acknowledge(t *testing.T) {
	pubHex, privKey := generateTestKeyPair(t)
	repo := newMockDiscordRepo()
	handler := NewDiscordInteractionHandler(pubHex, nil, repo)

	checkUUID := "abc-123-def"
	interaction := map[string]interface{}{
		"type":  3,
		"id":    "interaction-1",
		"token": "interaction-token",
		"data": map[string]interface{}{
			"custom_id":      fmt.Sprintf("checker_ack_%s", checkUUID),
			"component_type": 2,
		},
		"member": map[string]interface{}{
			"user": map[string]interface{}{
				"id":       "user-1",
				"username": "testuser",
			},
		},
		"message": map[string]interface{}{
			"id":         "msg-1",
			"channel_id": "chan-1",
			"embeds": []map[string]interface{}{
				{
					"title": "🔴 ALERT: Test Check",
					"color": discord.ColorRed,
					"fields": []map[string]interface{}{
						{"name": "Project", "value": "myproject", "inline": true},
						{"name": "Status", "value": "🔴 Unhealthy", "inline": true},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(interaction)
	req := buildSignedDiscordRequest(t, privKey, body)
	w := httptest.NewRecorder()

	handler.HandleInteraction(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp discord.InteractionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Type != interactionCallbackUpdateMessage {
		t.Errorf("expected response type %d, got %d", interactionCallbackUpdateMessage, resp.Type)
	}

	if resp.Data == nil {
		t.Fatal("expected response data, got nil")
	}

	// Check that "Acknowledged" field was added
	if len(resp.Data.Embeds) == 0 {
		t.Fatal("expected embeds in response")
	}
	foundAck := false
	for _, f := range resp.Data.Embeds[0].Fields {
		if f.Name == "Acknowledged" {
			foundAck = true
			if !strings.Contains(f.Value, "testuser") {
				t.Errorf("expected acknowledged field to contain username, got %s", f.Value)
			}
		}
	}
	if !foundAck {
		t.Error("expected 'Acknowledged' field in embed")
	}

	// Check that silence buttons are still present (ack doesn't remove them)
	if len(resp.Data.Components) == 0 {
		t.Fatal("expected components in response")
	}
}

func TestDiscordInteraction_Silence1h(t *testing.T) {
	pubHex, privKey := generateTestKeyPair(t)
	repo := newMockDiscordRepo()
	handler := NewDiscordInteractionHandler(pubHex, nil, repo)

	checkUUID := "abc-123-def"
	interaction := map[string]interface{}{
		"type":  3,
		"id":    "interaction-1",
		"token": "interaction-token",
		"data": map[string]interface{}{
			"custom_id":      fmt.Sprintf("checker_silence_%s_1h", checkUUID),
			"component_type": 2,
		},
		"member": map[string]interface{}{
			"user": map[string]interface{}{
				"id":       "user-1",
				"username": "testuser",
			},
		},
		"message": map[string]interface{}{
			"id":         "msg-1",
			"channel_id": "chan-1",
			"embeds": []map[string]interface{}{
				{
					"title": "🔴 ALERT: Test Check",
					"color": discord.ColorRed,
				},
			},
		},
	}

	body, _ := json.Marshal(interaction)
	req := buildSignedDiscordRequest(t, privKey, body)
	w := httptest.NewRecorder()

	handler.HandleInteraction(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify silence was created in DB
	if len(repo.createdSilences) != 1 {
		t.Fatalf("expected 1 created silence, got %d", len(repo.createdSilences))
	}
	s := repo.createdSilences[0]
	if s.Scope != "check" {
		t.Errorf("expected scope 'check', got '%s'", s.Scope)
	}
	if s.Target != checkUUID {
		t.Errorf("expected target '%s', got '%s'", checkUUID, s.Target)
	}
	if s.Channel != "discord" {
		t.Errorf("expected channel 'discord', got '%s'", s.Channel)
	}
	if s.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	} else {
		expectedExpiry := time.Now().Add(1 * time.Hour)
		diff := s.ExpiresAt.Sub(expectedExpiry)
		if diff > 5*time.Second || diff < -5*time.Second {
			t.Errorf("expected ExpiresAt around %v, got %v", expectedExpiry, *s.ExpiresAt)
		}
	}

	// Verify response has Unsilence button
	var resp discord.InteractionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil || len(resp.Data.Components) == 0 {
		t.Fatal("expected components in response")
	}
	foundUnsilence := false
	for _, row := range resp.Data.Components {
		for _, c := range row.Components {
			if strings.Contains(c.CustomID, "checker_unsilence_") {
				foundUnsilence = true
			}
		}
	}
	if !foundUnsilence {
		t.Error("expected 'Unsilence' button in response")
	}
}

func TestDiscordInteraction_Silence24h(t *testing.T) {
	pubHex, privKey := generateTestKeyPair(t)
	repo := newMockDiscordRepo()
	handler := NewDiscordInteractionHandler(pubHex, nil, repo)

	checkUUID := "abc-123-def"
	interaction := map[string]interface{}{
		"type":  3,
		"id":    "interaction-1",
		"token": "interaction-token",
		"data": map[string]interface{}{
			"custom_id":      fmt.Sprintf("checker_silence_%s_24h", checkUUID),
			"component_type": 2,
		},
		"member": map[string]interface{}{
			"user": map[string]interface{}{
				"id":       "user-1",
				"username": "testuser",
			},
		},
		"message": map[string]interface{}{
			"id":         "msg-1",
			"channel_id": "chan-1",
			"embeds":     []map[string]interface{}{},
		},
	}

	body, _ := json.Marshal(interaction)
	req := buildSignedDiscordRequest(t, privKey, body)
	w := httptest.NewRecorder()

	handler.HandleInteraction(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if len(repo.createdSilences) != 1 {
		t.Fatalf("expected 1 created silence, got %d", len(repo.createdSilences))
	}
	s := repo.createdSilences[0]
	if s.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	} else {
		expectedExpiry := time.Now().Add(24 * time.Hour)
		diff := s.ExpiresAt.Sub(expectedExpiry)
		if diff > 5*time.Second || diff < -5*time.Second {
			t.Errorf("expected ExpiresAt around %v, got %v", expectedExpiry, *s.ExpiresAt)
		}
	}
}

func TestDiscordInteraction_Unsilence(t *testing.T) {
	pubHex, privKey := generateTestKeyPair(t)
	repo := newMockDiscordRepo()
	handler := NewDiscordInteractionHandler(pubHex, nil, repo)

	checkUUID := "abc-123-def"
	interaction := map[string]interface{}{
		"type":  3,
		"id":    "interaction-1",
		"token": "interaction-token",
		"data": map[string]interface{}{
			"custom_id":      fmt.Sprintf("checker_unsilence_%s", checkUUID),
			"component_type": 2,
		},
		"member": map[string]interface{}{
			"user": map[string]interface{}{
				"id":       "user-1",
				"username": "testuser",
			},
		},
		"message": map[string]interface{}{
			"id":         "msg-1",
			"channel_id": "chan-1",
			"embeds": []map[string]interface{}{
				{
					"title": "🔴 ALERT: Test Check",
					"color": discord.ColorGray,
					"fields": []map[string]interface{}{
						{"name": "Silenced", "value": "🔇 testuser for 1h", "inline": true},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(interaction)
	req := buildSignedDiscordRequest(t, privKey, body)
	w := httptest.NewRecorder()

	handler.HandleInteraction(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify silence was deactivated
	if len(repo.deactivatedSilences) != 1 {
		t.Fatalf("expected 1 deactivated silence, got %d", len(repo.deactivatedSilences))
	}
	if repo.deactivatedSilences[0] != "check|"+checkUUID {
		t.Errorf("expected deactivated 'check|%s', got '%s'", checkUUID, repo.deactivatedSilences[0])
	}

	// Verify response restores alert buttons
	var resp discord.InteractionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil || len(resp.Data.Components) == 0 {
		t.Fatal("expected components in response")
	}
	foundAck := false
	foundSilence := false
	for _, row := range resp.Data.Components {
		for _, c := range row.Components {
			if strings.Contains(c.CustomID, "checker_ack_") {
				foundAck = true
			}
			if strings.Contains(c.CustomID, "checker_silence_") {
				foundSilence = true
			}
		}
	}
	if !foundAck {
		t.Error("expected 'Acknowledge' button restored after unsilence")
	}
	if !foundSilence {
		t.Error("expected 'Silence' buttons restored after unsilence")
	}

	// Verify "Silenced" field was removed from embeds
	if len(resp.Data.Embeds) > 0 {
		for _, f := range resp.Data.Embeds[0].Fields {
			if f.Name == "Silenced" {
				t.Error("expected 'Silenced' field to be removed after unsilence")
			}
		}
		// Verify color was restored to red
		if resp.Data.Embeds[0].Color != discord.ColorRed {
			t.Errorf("expected color restored to red (%d), got %d", discord.ColorRed, resp.Data.Embeds[0].Color)
		}
	}
}

func TestDiscordInteraction_MethodNotAllowed(t *testing.T) {
	pubHex, _ := generateTestKeyPair(t)
	handler := NewDiscordInteractionHandler(pubHex, nil, newMockDiscordRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/discord/interactions", nil)
	w := httptest.NewRecorder()

	handler.HandleInteraction(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestParseDiscordSilenceID(t *testing.T) {
	tests := []struct {
		remainder string
		wantUUID  string
		wantDur   time.Duration
		wantLabel string
	}{
		{"abc-123-def_1h", "abc-123-def", 1 * time.Hour, "1h"},
		{"abc-123-def_24h", "abc-123-def", 24 * time.Hour, "24h"},
		{"550e8400-e29b-41d4-a716-446655440000_1h", "550e8400-e29b-41d4-a716-446655440000", 1 * time.Hour, "1h"},
		{"nohyphen_1h", "nohyphen", 1 * time.Hour, "1h"},
	}

	for _, tt := range tests {
		t.Run(tt.remainder, func(t *testing.T) {
			uuid, dur, label := parseDiscordSilenceID(tt.remainder)
			if uuid != tt.wantUUID {
				t.Errorf("UUID: want %q, got %q", tt.wantUUID, uuid)
			}
			if dur != tt.wantDur {
				t.Errorf("duration: want %v, got %v", tt.wantDur, dur)
			}
			if label != tt.wantLabel {
				t.Errorf("label: want %q, got %q", tt.wantLabel, label)
			}
		})
	}
}

func TestParseDiscordSilenceID_NoUnderscore(t *testing.T) {
	uuid, _, _ := parseDiscordSilenceID("nounderscore")
	if uuid != "" {
		t.Errorf("expected empty UUID for invalid input, got %q", uuid)
	}
}

func TestGetInteractionUsername(t *testing.T) {
	tests := []struct {
		name        string
		interaction DiscordInteraction
		want        string
	}{
		{
			name: "member with nick",
			interaction: DiscordInteraction{
				Member: &DiscordMember{
					Nick: "MyNick",
					User: &DiscordUser{Username: "myuser"},
				},
			},
			want: "MyNick",
		},
		{
			name: "member without nick",
			interaction: DiscordInteraction{
				Member: &DiscordMember{
					User: &DiscordUser{Username: "myuser"},
				},
			},
			want: "myuser",
		},
		{
			name: "DM user (no member)",
			interaction: DiscordInteraction{
				User: &DiscordUser{Username: "dmuser"},
			},
			want: "dmuser",
		},
		{
			name:        "no user info",
			interaction: DiscordInteraction{},
			want:        "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getInteractionUsername(tt.interaction)
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}
