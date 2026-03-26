package alerts

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendNtfyAlert_Down(t *testing.T) {
	var received ntfyPayload

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

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := &NtfyAlerter{
		config: ntfyConfig{
			ServerURL: server.URL,
			Topic:     "test-topic",
		},
	}

	err := alerter.SendAlert(AlertPayload{
		CheckName: "api-health",
		Project:   "my-project",
		CheckType: "http",
		Message:   "Connection timeout after 5s",
		Severity:  "critical",
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	// Verify topic
	if received.Topic != "test-topic" {
		t.Errorf("expected topic 'test-topic', got %q", received.Topic)
	}

	// Verify title contains DOWN
	if !strings.Contains(received.Title, "DOWN") {
		t.Errorf("expected title to contain 'DOWN', got %q", received.Title)
	}

	// Verify message contains markdown fields
	if !strings.Contains(received.Message, "**Check:**") {
		t.Errorf("expected message to contain markdown fields, got %q", received.Message)
	}
	if !strings.Contains(received.Message, "api-health") {
		t.Errorf("expected message to contain check name, got %q", received.Message)
	}
	if !strings.Contains(received.Message, "my-project") {
		t.Errorf("expected message to contain project, got %q", received.Message)
	}

	// Verify priority 5 for critical severity
	if received.Priority != 5 {
		t.Errorf("expected priority 5 for critical, got %d", received.Priority)
	}

	// Verify tags contain rotating_light
	foundTag := false
	for _, tag := range received.Tags {
		if tag == "rotating_light" {
			foundTag = true
			break
		}
	}
	if !foundTag {
		t.Errorf("expected tags to contain 'rotating_light', got %v", received.Tags)
	}

	// Verify markdown is true
	if !received.Markdown {
		t.Error("expected markdown to be true")
	}
}

func TestSendNtfyAlert_Resolved(t *testing.T) {
	var received ntfyPayload

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

	alerter := &NtfyAlerter{
		config: ntfyConfig{
			ServerURL: server.URL,
			Topic:     "test-topic",
		},
	}

	err := alerter.SendRecovery(RecoveryPayload{
		CheckName: "db-check",
		Project:   "backend",
		CheckType: "tcp",
	})
	if err != nil {
		t.Fatalf("SendRecovery returned error: %v", err)
	}

	// Verify title contains RESOLVED
	if !strings.Contains(received.Title, "RESOLVED") {
		t.Errorf("expected title to contain 'RESOLVED', got %q", received.Title)
	}

	// Verify priority 3 (normal)
	if received.Priority != 3 {
		t.Errorf("expected priority 3, got %d", received.Priority)
	}

	// Verify tags contain white_check_mark
	foundTag := false
	for _, tag := range received.Tags {
		if tag == "white_check_mark" {
			foundTag = true
			break
		}
	}
	if !foundTag {
		t.Errorf("expected tags to contain 'white_check_mark', got %v", received.Tags)
	}
}

func TestSendNtfyAlert_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	alerter := &NtfyAlerter{
		config: ntfyConfig{
			ServerURL: server.URL,
			Topic:     "test-topic",
		},
	}

	err := alerter.SendAlert(AlertPayload{
		CheckName: "test",
		Project:   "test",
		CheckType: "http",
		Message:   "error",
		Severity:  "critical",
	})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestNewNtfyAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"server_url":"https://ntfy.example.com","topic":"my-topic"}`)
	a, err := NewAlerter("ntfy", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	na, ok := a.(*NtfyAlerter)
	if !ok {
		t.Fatalf("expected *NtfyAlerter, got %T", a)
	}
	if na.config.Topic != "my-topic" {
		t.Errorf("unexpected topic: %q", na.config.Topic)
	}
	if na.config.ServerURL != "https://ntfy.example.com" {
		t.Errorf("unexpected server_url: %q", na.config.ServerURL)
	}
	if na.Type() != "ntfy" {
		t.Errorf("expected Type() 'ntfy', got %q", na.Type())
	}
}

func TestNewNtfyAlerter_MissingTopic(t *testing.T) {
	_, err := NewAlerter("ntfy", json.RawMessage(`{"server_url":"https://ntfy.sh"}`))
	if err == nil {
		t.Fatal("expected error for missing topic, got nil")
	}
}

func TestNewNtfyAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("ntfy", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestNewNtfyAlerter_DefaultServerURL(t *testing.T) {
	cfg := json.RawMessage(`{"topic":"my-topic"}`)
	a, err := NewAlerter("ntfy", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	na := a.(*NtfyAlerter)
	if na.config.ServerURL != "https://ntfy.sh" {
		t.Errorf("expected default server_url 'https://ntfy.sh', got %q", na.config.ServerURL)
	}
}

func TestSendNtfyAlert_WithTokenAuth(t *testing.T) {
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := &NtfyAlerter{
		config: ntfyConfig{
			ServerURL: server.URL,
			Topic:     "test-topic",
			Token:     "tk_abc123",
		},
	}

	err := alerter.SendAlert(AlertPayload{
		CheckName: "test",
		Project:   "test",
		CheckType: "http",
		Message:   "error",
		Severity:  "critical",
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	// ntfy tokens use Basic Auth with token as username and empty password
	expectedCreds := base64.StdEncoding.EncodeToString([]byte("tk_abc123:"))
	expected := "Basic " + expectedCreds
	if authHeader != expected {
		t.Errorf("expected Authorization %q, got %q", expected, authHeader)
	}

	// Verify the exact base64 encoding: "tk_abc123:" -> "dGtfYWJjMTIzOg=="
	if expectedCreds != "dGtfYWJjMTIzOg==" {
		t.Errorf("expected base64 encoding 'dGtfYWJjMTIzOg==', got %q", expectedCreds)
	}
}

func TestSendNtfyAlert_WithBasicAuth(t *testing.T) {
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := &NtfyAlerter{
		config: ntfyConfig{
			ServerURL: server.URL,
			Topic:     "test-topic",
			Username:  "admin",
			Password:  "secret",
		},
	}

	err := alerter.SendAlert(AlertPayload{
		CheckName: "test",
		Project:   "test",
		CheckType: "http",
		Message:   "error",
		Severity:  "warning",
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	expectedCreds := base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	expected := "Basic " + expectedCreds
	if authHeader != expected {
		t.Errorf("expected Authorization %q, got %q", expected, authHeader)
	}
}

func TestNtfyPriorityMapping(t *testing.T) {
	tests := []struct {
		severity string
		want     int
	}{
		{"critical", 5},
		{"warning", 4},
		{"", 3},
		{"info", 3},
	}

	for _, tc := range tests {
		t.Run("severity_"+tc.severity, func(t *testing.T) {
			got := ntfyPriority(tc.severity)
			if got != tc.want {
				t.Errorf("ntfyPriority(%q) = %d, want %d", tc.severity, got, tc.want)
			}
		})
	}
}
