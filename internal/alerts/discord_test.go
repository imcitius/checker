package alerts

import (
	"encoding/json"
	"testing"
)

func TestNewDiscordBotAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"bot_token":"test-bot-token"}`)
	a, err := NewAlerter("discord", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Type() != "discord" {
		t.Errorf("expected Type() 'discord', got %q", a.Type())
	}
}

func TestNewDiscordBotAlerter_MissingBotToken(t *testing.T) {
	_, err := NewAlerter("discord", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for missing bot_token, got nil")
	}
}

func TestNewDiscordBotAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("discord", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDiscordBotAlerter_NoOpSendAlert(t *testing.T) {
	cfg := json.RawMessage(`{"bot_token":"test-bot-token"}`)
	a, err := NewAlerter("discord", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := a.SendAlert(AlertPayload{}); err != nil {
		t.Errorf("expected nil error from no-op SendAlert, got %v", err)
	}
}

func TestDiscordBotAlerter_NoOpSendRecovery(t *testing.T) {
	cfg := json.RawMessage(`{"bot_token":"test-bot-token"}`)
	a, err := NewAlerter("discord", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := a.SendRecovery(RecoveryPayload{}); err != nil {
		t.Errorf("expected nil error from no-op SendRecovery, got %v", err)
	}
}

func TestDiscordIsRegisteredType(t *testing.T) {
	if !IsRegisteredType("discord") {
		t.Error("expected IsRegisteredType('discord') to return true")
	}
}
