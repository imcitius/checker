package alerts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestThemeColorForStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"DOWN", "FF0000"},
		{"RESOLVED", "00FF00"},
		{"UNKNOWN", "FF0000"}, // default to red
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := themeColorForStatus(tt.status)
			if got != tt.expected {
				t.Errorf("themeColorForStatus(%q) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestBuildTeamsPayload(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	params := TeamsAlertParams{
		CheckName:   "api-health",
		ProjectName: "my-project",
		Status:      "DOWN",
		Error:       "connection refused",
		Time:        ts,
	}

	payload := buildTeamsPayload(params)

	if payload.Type != "MessageCard" {
		t.Errorf("expected @type MessageCard, got %s", payload.Type)
	}
	if payload.Context != "http://schema.org/extensions" {
		t.Errorf("expected @context http://schema.org/extensions, got %s", payload.Context)
	}
	if payload.ThemeColor != "FF0000" {
		t.Errorf("expected themeColor FF0000 for DOWN, got %s", payload.ThemeColor)
	}
	if payload.Summary != "api-health is DOWN" {
		t.Errorf("expected summary 'api-health is DOWN', got %s", payload.Summary)
	}
	if len(payload.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(payload.Sections))
	}

	section := payload.Sections[0]
	if section.ActivityTitle != "api-health" {
		t.Errorf("expected activityTitle 'api-health', got %s", section.ActivityTitle)
	}
	if section.ActivitySubtitle != "Project: my-project" {
		t.Errorf("expected activitySubtitle 'Project: my-project', got %s", section.ActivitySubtitle)
	}
	if len(section.Facts) != 3 {
		t.Fatalf("expected 3 facts, got %d", len(section.Facts))
	}
	if section.Facts[0].Value != "DOWN" {
		t.Errorf("expected Status fact value DOWN, got %s", section.Facts[0].Value)
	}
	if section.Facts[1].Value != "connection refused" {
		t.Errorf("expected Error fact value 'connection refused', got %s", section.Facts[1].Value)
	}
}

func TestBuildTeamsPayloadResolved(t *testing.T) {
	params := TeamsAlertParams{
		CheckName:   "db-check",
		ProjectName: "backend",
		Status:      "RESOLVED",
		Error:       "",
		Time:        time.Now(),
	}

	payload := buildTeamsPayload(params)

	if payload.ThemeColor != "00FF00" {
		t.Errorf("expected themeColor 00FF00 for RESOLVED, got %s", payload.ThemeColor)
	}
	if payload.Summary != "db-check is RESOLVED" {
		t.Errorf("expected summary 'db-check is RESOLVED', got %s", payload.Summary)
	}
}

func TestSendTeamsAlert_Success(t *testing.T) {
	var receivedPayload TeamsMessageCard

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	params := TeamsAlertParams{
		CheckName:   "web-check",
		ProjectName: "frontend",
		Status:      "DOWN",
		Error:       "timeout after 30s",
		Time:        time.Now(),
	}

	err := SendTeamsAlert(server.URL, params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if receivedPayload.Type != "MessageCard" {
		t.Errorf("expected @type MessageCard, got %s", receivedPayload.Type)
	}
	if receivedPayload.ThemeColor != "FF0000" {
		t.Errorf("expected themeColor FF0000, got %s", receivedPayload.ThemeColor)
	}
	if receivedPayload.Summary != "web-check is DOWN" {
		t.Errorf("expected summary 'web-check is DOWN', got %s", receivedPayload.Summary)
	}
}

func TestSendTeamsAlert_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	params := TeamsAlertParams{
		CheckName:   "check1",
		ProjectName: "proj1",
		Status:      "DOWN",
		Error:       "error",
		Time:        time.Now(),
	}

	err := SendTeamsAlert(server.URL, params)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "teams alert") || !strings.Contains(err.Error(), "500") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewTeamsAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"webhook_url":"https://outlook.office.com/webhook/abc"}`)
	a, err := NewAlerter("teams", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ta, ok := a.(*TeamsAlerter)
	if !ok {
		t.Fatalf("expected *TeamsAlerter, got %T", a)
	}
	if ta.WebhookURL != "https://outlook.office.com/webhook/abc" {
		t.Errorf("unexpected WebhookURL: %q", ta.WebhookURL)
	}
	if ta.Type() != "teams" {
		t.Errorf("expected Type() 'teams', got %q", ta.Type())
	}
}

func TestNewTeamsAlerter_MissingURL(t *testing.T) {
	_, err := NewAlerter("teams", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewTeamsAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("teams", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestSendTeamsAlert_InvalidURL(t *testing.T) {
	params := TeamsAlertParams{
		CheckName:   "check1",
		ProjectName: "proj1",
		Status:      "DOWN",
		Error:       "error",
		Time:        time.Now(),
	}

	err := SendTeamsAlert("http://invalid-host-that-does-not-exist.local:99999/webhook", params)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}
