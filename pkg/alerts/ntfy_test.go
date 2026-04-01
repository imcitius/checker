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

	// ntfy access tokens use Bearer auth per https://docs.ntfy.sh/publish/#access-tokens
	expected := "Bearer tk_abc123"
	if authHeader != expected {
		t.Errorf("expected Authorization %q, got %q", expected, authHeader)
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

func TestSendNtfyAlert_WithClickURL(t *testing.T) {
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
			ClickURL:  "https://checker.example.com",
		},
	}

	err := alerter.SendAlert(AlertPayload{
		CheckName: "api-health",
		CheckUUID: "abc-123",
		Project:   "my-project",
		CheckType: "http",
		Message:   "Connection timeout",
		Severity:  "critical",
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	expectedURL := "https://checker.example.com/checks/abc-123"

	// Verify click URL
	if received.Click != expectedURL {
		t.Errorf("expected click %q, got %q", expectedURL, received.Click)
	}

	// Verify actions
	if len(received.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(received.Actions))
	}
	action := received.Actions[0]
	if action.Action != "view" {
		t.Errorf("expected action type 'view', got %q", action.Action)
	}
	if action.Label != "View in Checker" {
		t.Errorf("expected label 'View in Checker', got %q", action.Label)
	}
	if action.URL != expectedURL {
		t.Errorf("expected action URL %q, got %q", expectedURL, action.URL)
	}
}

func TestSendNtfyAlert_WithClickURLTrailingSlash(t *testing.T) {
	var received ntfyPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := &NtfyAlerter{
		config: ntfyConfig{
			ServerURL: server.URL,
			Topic:     "test-topic",
			ClickURL:  "https://checker.example.com/",
		},
	}

	err := alerter.SendAlert(AlertPayload{
		CheckName: "test",
		CheckUUID: "uuid-456",
		Project:   "test",
		CheckType: "http",
		Message:   "error",
		Severity:  "warning",
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	expectedURL := "https://checker.example.com/checks/uuid-456"
	if received.Click != expectedURL {
		t.Errorf("expected click %q (trailing slash trimmed), got %q", expectedURL, received.Click)
	}
}

func TestSendNtfyAlert_NoClickURL(t *testing.T) {
	var received ntfyPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
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
		CheckName: "test",
		CheckUUID: "uuid-789",
		Project:   "test",
		CheckType: "http",
		Message:   "error",
		Severity:  "critical",
	})
	if err != nil {
		t.Fatalf("SendAlert returned error: %v", err)
	}

	// No click URL or actions when click_url is not configured
	if received.Click != "" {
		t.Errorf("expected empty click, got %q", received.Click)
	}
	if len(received.Actions) != 0 {
		t.Errorf("expected no actions, got %d", len(received.Actions))
	}
}

func TestSendNtfyRecovery_WithClickURL(t *testing.T) {
	var received ntfyPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := &NtfyAlerter{
		config: ntfyConfig{
			ServerURL: server.URL,
			Topic:     "test-topic",
			ClickURL:  "https://checker.example.com",
		},
	}

	err := alerter.SendRecovery(RecoveryPayload{
		CheckName: "db-check",
		CheckUUID: "recovery-uuid",
		Project:   "backend",
		CheckType: "tcp",
	})
	if err != nil {
		t.Fatalf("SendRecovery returned error: %v", err)
	}

	expectedURL := "https://checker.example.com/checks/recovery-uuid"

	if received.Click != expectedURL {
		t.Errorf("expected click %q, got %q", expectedURL, received.Click)
	}
	if len(received.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(received.Actions))
	}
	if received.Actions[0].Label != "View in Checker" {
		t.Errorf("expected label 'View in Checker', got %q", received.Actions[0].Label)
	}
	if received.Actions[0].URL != expectedURL {
		t.Errorf("expected action URL %q, got %q", expectedURL, received.Actions[0].URL)
	}
}

func TestSendNtfyTest_WithClickURL(t *testing.T) {
	var received ntfyPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := SendNtfyTest(server.URL, "test-topic", "", "", "", "test message", "https://checker.example.com", false)
	if err != nil {
		t.Fatalf("SendNtfyTest returned error: %v", err)
	}

	expectedURL := "https://checker.example.com/checks"
	if received.Click != expectedURL {
		t.Errorf("expected click %q, got %q", expectedURL, received.Click)
	}
	if len(received.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(received.Actions))
	}
	if received.Actions[0].Action != "view" {
		t.Errorf("expected action type 'view', got %q", received.Actions[0].Action)
	}
	if received.Actions[0].Label != "View in Checker" {
		t.Errorf("expected label 'View in Checker', got %q", received.Actions[0].Label)
	}
}

func TestSendNtfyTest_NoClickURL(t *testing.T) {
	var received ntfyPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := SendNtfyTest(server.URL, "test-topic", "", "", "", "test message", "", false)
	if err != nil {
		t.Fatalf("SendNtfyTest returned error: %v", err)
	}

	if received.Click != "" {
		t.Errorf("expected empty click, got %q", received.Click)
	}
	if len(received.Actions) != 0 {
		t.Errorf("expected no actions, got %d", len(received.Actions))
	}
}

func TestNewNtfyAlerter_InvalidServerURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"double colon typo", `{"topic":"t","server_url":"https:://ntfy.example.com"}`},
		{"no scheme", `{"topic":"t","server_url":"ntfy.example.com"}`},
		{"ftp scheme", `{"topic":"t","server_url":"ftp://ntfy.example.com"}`},
		{"no host", `{"topic":"t","server_url":"https://"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAlerter("ntfy", json.RawMessage(tc.url))
			if err == nil {
				t.Fatalf("expected error for invalid server_url, got nil")
			}
			if !strings.Contains(err.Error(), "not a valid HTTP(S) URL") {
				t.Errorf("expected 'not a valid HTTP(S) URL' in error, got: %v", err)
			}
		})
	}
}

func TestNewNtfyAlerter_TrailingSlashNormalized(t *testing.T) {
	cfg := json.RawMessage(`{"topic":"t","server_url":"https://ntfy.example.com/"}`)
	a, err := NewAlerter("ntfy", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	na := a.(*NtfyAlerter)
	if na.config.ServerURL != "https://ntfy.example.com" {
		t.Errorf("expected trailing slash stripped, got %q", na.config.ServerURL)
	}
}

func TestSendNtfyTest_InvalidServerURL(t *testing.T) {
	err := SendNtfyTest("https:://ntfy.example.com", "topic", "", "", "", "test", "", false)
	if err == nil {
		t.Fatal("expected error for invalid server URL, got nil")
	}
	if !strings.Contains(err.Error(), "invalid server URL") {
		t.Errorf("expected 'invalid server URL' in error, got: %v", err)
	}
}

func TestSendNtfyTest_EmptyServerURLDefaults(t *testing.T) {
	// This should not error — empty URL defaults to https://ntfy.sh
	// We can't actually send, but we can verify it doesn't fail on URL validation
	// by checking it proceeds past validation (will fail on network, not validation)
	err := SendNtfyTest("", "topic", "", "", "", "test", "", false)
	if err != nil {
		// Should fail on network, not on URL validation
		if strings.Contains(err.Error(), "invalid server URL") {
			t.Errorf("empty URL should default to ntfy.sh, not fail validation: %v", err)
		}
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
