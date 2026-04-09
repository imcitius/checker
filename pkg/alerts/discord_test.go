// SPDX-License-Identifier: BUSL-1.1

package alerts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDiscordBotAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"bot_token":"test-bot-token","default_channel":"123456"}`)
	a, err := NewAlerter("discord", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Type() != "discord" {
		t.Errorf("expected Type() 'discord', got %q", a.Type())
	}
}

func TestNewDiscordBotAlerter_WithAppID(t *testing.T) {
	cfg := json.RawMessage(`{"bot_token":"test-bot-token","app_id":"app-123","default_channel":"123456"}`)
	a, err := NewAlerter("discord", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Type() != "discord" {
		t.Errorf("expected Type() 'discord', got %q", a.Type())
	}
}

func TestNewDiscordBotAlerter_MissingBotToken(t *testing.T) {
	_, err := NewAlerter("discord", json.RawMessage(`{"default_channel":"123456"}`))
	if err == nil {
		t.Fatal("expected error for missing bot_token, got nil")
	}
}

func TestNewDiscordBotAlerter_MissingDefaultChannel(t *testing.T) {
	_, err := NewAlerter("discord", json.RawMessage(`{"bot_token":"test-bot-token"}`))
	if err == nil {
		t.Fatal("expected error for missing default_channel, got nil")
	}
}

func TestNewDiscordBotAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("discord", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDiscordIsRegisteredType(t *testing.T) {
	if !IsRegisteredType("discord") {
		t.Error("expected IsRegisteredType('discord') to return true")
	}
}

func TestDiscordBotAlerter_SendAlert(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"msg-001","channel_id":"ch-123"}`))
	}))
	defer server.Close()

	a := createTestAlerter(t, server.URL)

	err := a.SendAlert(AlertPayload{
		CheckName:  "web-check",
		CheckUUID:  "uuid-abc",
		Project:    "myproject",
		CheckGroup: "prod",
		CheckType:  "http",
		Message:    "connection refused",
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	if gotPath != "/channels/ch-123/messages" {
		t.Errorf("expected path /channels/ch-123/messages, got %s", gotPath)
	}
	if gotAuth != "Bot test-bot-token" {
		t.Errorf("expected auth 'Bot test-bot-token', got %s", gotAuth)
	}
	// Verify embeds are present (alert message uses embeds)
	embeds, ok := gotPayload["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Error("expected embeds in alert payload")
	}
}

func TestDiscordBotAlerter_SendRecovery(t *testing.T) {
	var gotPath string
	var gotPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"msg-002","channel_id":"ch-123"}`))
	}))
	defer server.Close()

	a := createTestAlerter(t, server.URL)

	err := a.SendRecovery(RecoveryPayload{
		CheckName:  "web-check",
		CheckUUID:  "uuid-abc",
		Project:    "myproject",
		CheckGroup: "prod",
		CheckType:  "http",
	})
	if err != nil {
		t.Fatalf("SendRecovery returned error: %v", err)
	}

	if gotPath != "/channels/ch-123/messages" {
		t.Errorf("expected path /channels/ch-123/messages, got %s", gotPath)
	}
	embeds, ok := gotPayload["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Error("expected embeds in recovery payload")
	}
}

func TestDiscordBotAlerter_SendAlertAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer server.Close()

	a := createTestAlerter(t, server.URL)

	err := a.SendAlert(AlertPayload{CheckName: "test"})
	if err == nil {
		t.Fatal("expected error on API failure, got nil")
	}
}

func TestDiscordBotAlerter_SendRecoveryAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer server.Close()

	a := createTestAlerter(t, server.URL)

	err := a.SendRecovery(RecoveryPayload{CheckName: "test"})
	if err == nil {
		t.Fatal("expected error on API failure, got nil")
	}
}

// createTestAlerter builds a discordBotAlerter pointing at a test server.
func createTestAlerter(t *testing.T, serverURL string) Alerter {
	t.Helper()
	cfg := json.RawMessage(`{"bot_token":"test-bot-token","app_id":"test-app","default_channel":"ch-123"}`)
	a, err := NewAlerter("discord", cfg)
	if err != nil {
		t.Fatalf("failed to create alerter: %v", err)
	}
	// Override the base URL to point at the test server
	da := a.(*discordBotAlerter)
	da.client.SetBaseURL(serverURL)
	return a
}
