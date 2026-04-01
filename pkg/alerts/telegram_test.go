package alerts

import (
	"encoding/json"
	"testing"
)

func TestNewTelegramAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"bot_token":"123:ABC","chat_id":"-100123"}`)
	a, err := NewAlerter("telegram", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ta, ok := a.(*TelegramAlerter)
	if !ok {
		t.Fatalf("expected *TelegramAlerter, got %T", a)
	}
	if ta.BotToken != "123:ABC" {
		t.Errorf("expected BotToken '123:ABC', got %q", ta.BotToken)
	}
	if ta.ChatID != "-100123" {
		t.Errorf("expected ChatID '-100123', got %q", ta.ChatID)
	}
	if ta.Type() != "telegram" {
		t.Errorf("expected Type() 'telegram', got %q", ta.Type())
	}
}

func TestNewTelegramAlerter_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  string
	}{
		{"missing bot_token", `{"chat_id":"-100123"}`},
		{"missing chat_id", `{"bot_token":"123:ABC"}`},
		{"both empty", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAlerter("telegram", json.RawMessage(tt.cfg))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestNewTelegramAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("telegram", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
