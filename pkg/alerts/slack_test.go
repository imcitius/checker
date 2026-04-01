package alerts

import (
	"encoding/json"
	"testing"
)

func TestNewSlackWebhookAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"webhook_url":"https://hooks.slack.com/services/T/B/X"}`)
	a, err := NewAlerter("slack", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sa, ok := a.(*SlackWebhookAlerter)
	if !ok {
		t.Fatalf("expected *SlackWebhookAlerter, got %T", a)
	}
	if sa.WebhookURL != "https://hooks.slack.com/services/T/B/X" {
		t.Errorf("unexpected WebhookURL: %q", sa.WebhookURL)
	}
	if sa.Type() != "slack" {
		t.Errorf("expected Type() 'slack', got %q", sa.Type())
	}
}

func TestNewSlackWebhookAlerter_SlackWebhookType(t *testing.T) {
	cfg := json.RawMessage(`{"webhook_url":"https://hooks.slack.com/services/T/B/X"}`)
	a, err := NewAlerter("slack_webhook", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := a.(*SlackWebhookAlerter); !ok {
		t.Fatalf("expected *SlackWebhookAlerter, got %T", a)
	}
}

func TestNewSlackWebhookAlerter_MissingURL(t *testing.T) {
	_, err := NewAlerter("slack", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewSlackWebhookAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("slack", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
