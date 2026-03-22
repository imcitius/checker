package alerts

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendDiscordAlert_Down(t *testing.T) {
	var received DiscordPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	params := DiscordAlertParams{
		CheckName: "api-health",
		Project:   "my-project",
		CheckType: "http",
		Message:   "Connection timeout after 5s",
		IsDown:    true,
	}
	payload := BuildDiscordPayload(params)

	if err := SendDiscordAlert(server.URL, payload); err != nil {
		t.Fatalf("SendDiscordAlert returned error: %v", err)
	}

	// Verify embed structure
	if len(received.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(received.Embeds))
	}

	embed := received.Embeds[0]

	// Verify color is red for DOWN
	if embed.Color != ColorRed {
		t.Errorf("expected color %d (red), got %d", ColorRed, embed.Color)
	}

	// Verify title contains DOWN
	if embed.Title != "🔴 api-health is DOWN" {
		t.Errorf("unexpected title: %s", embed.Title)
	}

	// Verify description
	if embed.Description != "Connection timeout after 5s" {
		t.Errorf("unexpected description: %s", embed.Description)
	}

	// Verify timestamp is present
	if embed.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}

	// Verify fields
	if len(embed.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(embed.Fields))
	}
	if embed.Fields[0].Name != "Project" || embed.Fields[0].Value != "my-project" {
		t.Errorf("unexpected Project field: %+v", embed.Fields[0])
	}
	if !embed.Fields[0].Inline {
		t.Error("expected Project field to be inline")
	}
	if embed.Fields[1].Name != "Type" || embed.Fields[1].Value != "http" {
		t.Errorf("unexpected Type field: %+v", embed.Fields[1])
	}
	if !embed.Fields[1].Inline {
		t.Error("expected Type field to be inline")
	}
}

func TestSendDiscordAlert_Resolved(t *testing.T) {
	var received DiscordPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	params := DiscordAlertParams{
		CheckName: "db-check",
		Project:   "backend",
		CheckType: "tcp",
		Message:   "Connection restored",
		IsDown:    false,
	}
	payload := BuildDiscordPayload(params)

	if err := SendDiscordAlert(server.URL, payload); err != nil {
		t.Fatalf("SendDiscordAlert returned error: %v", err)
	}

	if len(received.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(received.Embeds))
	}

	embed := received.Embeds[0]

	// Verify color is green for RESOLVED
	if embed.Color != ColorGreen {
		t.Errorf("expected color %d (green), got %d", ColorGreen, embed.Color)
	}

	// Verify title contains RESOLVED
	if embed.Title != "🟢 db-check is RESOLVED" {
		t.Errorf("unexpected title: %s", embed.Title)
	}

	// Verify fields
	if len(embed.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(embed.Fields))
	}
	if embed.Fields[0].Value != "backend" {
		t.Errorf("expected Project=backend, got %s", embed.Fields[0].Value)
	}
	if embed.Fields[1].Value != "tcp" {
		t.Errorf("expected Type=tcp, got %s", embed.Fields[1].Value)
	}
}

func TestSendDiscordAlert_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	payload := BuildDiscordPayload(DiscordAlertParams{
		CheckName: "test",
		Project:   "test",
		CheckType: "http",
		Message:   "error",
		IsDown:    true,
	})

	err := SendDiscordAlert(server.URL, payload)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestNewDiscordAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"webhook_url":"https://discord.com/api/webhooks/123/abc"}`)
	a, err := NewAlerter("discord", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	da, ok := a.(*DiscordAlerter)
	if !ok {
		t.Fatalf("expected *DiscordAlerter, got %T", a)
	}
	if da.WebhookURL != "https://discord.com/api/webhooks/123/abc" {
		t.Errorf("unexpected WebhookURL: %q", da.WebhookURL)
	}
	if da.Type() != "discord" {
		t.Errorf("expected Type() 'discord', got %q", da.Type())
	}
}

func TestNewDiscordAlerter_MissingURL(t *testing.T) {
	_, err := NewAlerter("discord", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewDiscordAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("discord", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestBuildDiscordPayload_Colors(t *testing.T) {
	downPayload := BuildDiscordPayload(DiscordAlertParams{
		CheckName: "test", Project: "p", CheckType: "http", Message: "err", IsDown: true,
	})
	if downPayload.Embeds[0].Color != ColorRed {
		t.Errorf("DOWN should use red (%d), got %d", ColorRed, downPayload.Embeds[0].Color)
	}

	upPayload := BuildDiscordPayload(DiscordAlertParams{
		CheckName: "test", Project: "p", CheckType: "http", Message: "ok", IsDown: false,
	})
	if upPayload.Embeds[0].Color != ColorGreen {
		t.Errorf("RESOLVED should use green (%d), got %d", ColorGreen, upPayload.Embeds[0].Color)
	}
}
