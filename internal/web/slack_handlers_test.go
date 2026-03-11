package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerifySlackSignature_Valid(t *testing.T) {
	secret := "test-signing-secret"
	body := []byte("payload=%7B%22type%22%3A%22block_actions%22%7D")
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Compute expected signature
	sigBasestring := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigBasestring))
	expectedSig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/slack/interactive", nil)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", expectedSig)

	if !VerifySlackSignature(secret, req, body) {
		t.Error("expected valid signature to be accepted")
	}
}

func TestVerifySlackSignature_InvalidSignature(t *testing.T) {
	secret := "test-signing-secret"
	body := []byte("payload=%7B%22type%22%3A%22block_actions%22%7D")
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	req := httptest.NewRequest(http.MethodPost, "/api/slack/interactive", nil)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", "v0=invalidsignature")

	if VerifySlackSignature(secret, req, body) {
		t.Error("expected invalid signature to be rejected")
	}
}

func TestVerifySlackSignature_ExpiredTimestamp(t *testing.T) {
	secret := "test-signing-secret"
	body := []byte("payload=%7B%22type%22%3A%22block_actions%22%7D")
	// Timestamp 10 minutes ago
	timestamp := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())

	sigBasestring := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigBasestring))
	expectedSig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/slack/interactive", nil)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", expectedSig)

	if VerifySlackSignature(secret, req, body) {
		t.Error("expected expired timestamp to be rejected")
	}
}

func TestVerifySlackSignature_MissingHeaders(t *testing.T) {
	secret := "test-signing-secret"
	body := []byte("test")

	tests := []struct {
		name      string
		timestamp string
		signature string
	}{
		{"missing both", "", ""},
		{"missing timestamp", "", "v0=abc"},
		{"missing signature", fmt.Sprintf("%d", time.Now().Unix()), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.timestamp != "" {
				req.Header.Set("X-Slack-Request-Timestamp", tt.timestamp)
			}
			if tt.signature != "" {
				req.Header.Set("X-Slack-Signature", tt.signature)
			}
			if VerifySlackSignature(secret, req, body) {
				t.Error("expected missing headers to be rejected")
			}
		})
	}
}

func TestVerifySlackSignature_EmptySecret(t *testing.T) {
	body := []byte("test")
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("X-Slack-Signature", "v0=abc")

	if VerifySlackSignature("", req, body) {
		t.Error("expected empty secret to be rejected")
	}
}

func TestVerifySlackSignature_WrongSecret(t *testing.T) {
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"
	body := []byte("test-body")
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Sign with correct secret
	sigBasestring := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(correctSecret))
	mac.Write([]byte(sigBasestring))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", sig)

	// Verify with wrong secret
	if VerifySlackSignature(wrongSecret, req, body) {
		t.Error("expected wrong secret to be rejected")
	}
}

func TestVerifySlackSignature_InvalidTimestampFormat(t *testing.T) {
	secret := "test-secret"
	body := []byte("test")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Slack-Request-Timestamp", "not-a-number")
	req.Header.Set("X-Slack-Signature", "v0=abc")

	if VerifySlackSignature(secret, req, body) {
		t.Error("expected invalid timestamp format to be rejected")
	}
}

func TestParseInteractionPayload_Valid(t *testing.T) {
	payloadJSON := `{"type":"block_actions","user":{"id":"U12345","username":"testuser","name":"Test User"},"channel":{"id":"C67890","name":"alerts"},"message":{"ts":"1234567890.123456","text":"test alert"},"actions":[{"action_id":"silence_check","block_id":"alert_actions","value":"check-uuid-123","type":"button"}]}`

	// URL-encode the payload
	encoded := "payload=" + urlEncode(payloadJSON)

	payload, err := ParseInteractionPayload([]byte(encoded))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload.Type != "block_actions" {
		t.Errorf("type = %q, want %q", payload.Type, "block_actions")
	}
	if payload.User.ID != "U12345" {
		t.Errorf("user.id = %q, want %q", payload.User.ID, "U12345")
	}
	if payload.User.Username != "testuser" {
		t.Errorf("user.username = %q, want %q", payload.User.Username, "testuser")
	}
	if payload.Channel.ID != "C67890" {
		t.Errorf("channel.id = %q, want %q", payload.Channel.ID, "C67890")
	}
	if payload.Message.Ts != "1234567890.123456" {
		t.Errorf("message.ts = %q, want %q", payload.Message.Ts, "1234567890.123456")
	}
	if len(payload.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(payload.Actions))
	}
	if payload.Actions[0].ActionID != "silence_check" {
		t.Errorf("action_id = %q, want %q", payload.Actions[0].ActionID, "silence_check")
	}
	if payload.Actions[0].Value != "check-uuid-123" {
		t.Errorf("action value = %q, want %q", payload.Actions[0].Value, "check-uuid-123")
	}
}

func TestParseInteractionPayload_MissingPayloadField(t *testing.T) {
	body := []byte("other_field=value")

	_, err := ParseInteractionPayload(body)
	if err == nil {
		t.Error("expected error for missing payload field")
	}
	if !strings.Contains(err.Error(), "missing payload") {
		t.Errorf("error should mention missing payload, got: %v", err)
	}
}

func TestParseInteractionPayload_InvalidJSON(t *testing.T) {
	body := []byte("payload=not-valid-json")

	_, err := ParseInteractionPayload(body)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseInteractionPayload_EmptyBody(t *testing.T) {
	_, err := ParseInteractionPayload([]byte(""))
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseInteractionPayload_MultipleActions(t *testing.T) {
	payloadJSON := `{"type":"block_actions","user":{"id":"U111"},"channel":{"id":"C222"},"message":{"ts":"111.222"},"actions":[{"action_id":"silence_check","value":"uuid-1","type":"button"},{"action_id":"ack_alert","value":"uuid-2","type":"button"}]}`

	encoded := "payload=" + urlEncode(payloadJSON)
	payload, err := ParseInteractionPayload([]byte(encoded))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(payload.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(payload.Actions))
	}
	if payload.Actions[0].ActionID != "silence_check" {
		t.Errorf("action[0].action_id = %q, want %q", payload.Actions[0].ActionID, "silence_check")
	}
	if payload.Actions[1].ActionID != "ack_alert" {
		t.Errorf("action[1].action_id = %q, want %q", payload.Actions[1].ActionID, "ack_alert")
	}
}

func TestParseInteractionPayload_SilenceProjectAction(t *testing.T) {
	payloadJSON := `{"type":"block_actions","user":{"id":"U999"},"channel":{"id":"C888"},"message":{"ts":"999.888"},"actions":[{"action_id":"silence_project","value":"my-project","type":"button"}]}`

	encoded := "payload=" + urlEncode(payloadJSON)
	payload, err := ParseInteractionPayload([]byte(encoded))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload.Actions[0].ActionID != "silence_project" {
		t.Errorf("action_id = %q, want %q", payload.Actions[0].ActionID, "silence_project")
	}
	if payload.Actions[0].Value != "my-project" {
		t.Errorf("action value = %q, want %q", payload.Actions[0].Value, "my-project")
	}
}

func TestHandleInteraction_MethodNotAllowed(t *testing.T) {
	handler := NewSlackInteractiveHandler("secret", nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/slack/interactive", nil)
	rr := httptest.NewRecorder()

	handler.HandleInteraction(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleInteraction_InvalidSignature(t *testing.T) {
	handler := NewSlackInteractiveHandler("secret", nil, nil)

	body := "payload=%7B%7D"
	req := httptest.NewRequest(http.MethodPost, "/api/slack/interactive", strings.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("X-Slack-Signature", "v0=invalid")

	rr := httptest.NewRecorder()
	handler.HandleInteraction(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleInteraction_ValidSignatureNoActions(t *testing.T) {
	secret := "test-secret"
	payloadJSON := `{"type":"block_actions","user":{"id":"U1"},"channel":{"id":"C1"},"message":{"ts":"1.1"},"actions":[]}`
	body := "payload=" + urlEncode(payloadJSON)

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := computeSignature(secret, timestamp, body)

	handler := NewSlackInteractiveHandler(secret, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/slack/interactive", strings.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", sig)

	rr := httptest.NewRecorder()
	handler.HandleInteraction(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestFormUnescape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello+world", "hello world"},
		{"hello%20world", "hello world"},
		{"%7B%22key%22%3A%22value%22%7D", `{"key":"value"}`},
		{"no+encoding", "no encoding"},
		{"", ""},
		{"plain", "plain"},
		{"%25", "%"},
		{"abc%2Fdef", "abc/def"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formUnescape(tt.input)
			if got != tt.want {
				t.Errorf("formUnescape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFormValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		key   string
		want  string
	}{
		{"single pair", "key=value", "key", "value"},
		{"payload field", "payload=%7B%22test%22%7D", "payload", `{"test"}`},
		{"multiple pairs", "a=1&b=2&c=3", "b", "2"},
		{"url encoded", "key=hello+world", "key", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFormValues(tt.input)
			if got := result[tt.key]; got != tt.want {
				t.Errorf("parseFormValues(%q)[%q] = %q, want %q", tt.input, tt.key, got, tt.want)
			}
		})
	}
}

func TestExtractCheckNameFromMessage(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "standard alert",
			text: "🔴 ALERT: API Health Check: connection refused",
			want: "API Health Check",
		},
		{
			name: "empty text",
			text: "",
			want: "Unknown Check",
		},
		{
			name: "simple format",
			text: "🔴 My Check: error",
			want: "My Check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := SlackInteractionPayload{
				Message: SlackMessage{Text: tt.text},
			}
			got := extractCheckNameFromMessage(payload)
			if got != tt.want {
				t.Errorf("extractCheckNameFromMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractProjectFromMessage(t *testing.T) {
	payload := SlackInteractionPayload{
		Actions: []SlackAction{
			{ActionID: "silence_project", Value: "backend"},
		},
	}
	got := extractProjectFromMessage(payload)
	if got != "backend" {
		t.Errorf("extractProjectFromMessage() = %q, want %q", got, "backend")
	}

	// No silence_project action
	payload2 := SlackInteractionPayload{
		Actions: []SlackAction{
			{ActionID: "silence_check", Value: "uuid-123"},
		},
	}
	got2 := extractProjectFromMessage(payload2)
	if got2 != "" {
		t.Errorf("extractProjectFromMessage() = %q, want empty", got2)
	}
}

// Helper functions for tests

func urlEncode(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			result.WriteByte(c)
		} else {
			result.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return result.String()
}

func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~'
}

func computeSignature(secret, timestamp, body string) string {
	sigBasestring := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigBasestring))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}
