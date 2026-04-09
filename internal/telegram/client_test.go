// SPDX-License-Identifier: BUSL-1.1

package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient creates a TelegramClient pointing at the given test server.
func newTestClient(serverURL string) *TelegramClient {
	return &TelegramClient{
		botToken:      "test-token",
		secretToken:   "test-secret",
		defaultChatID: "12345",
		httpClient:    http.DefaultClient,
		baseURL:       serverURL,
	}
}

func TestSendMessage(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)

		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage(`{"message_id": 42, "chat": {"id": 12345}}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	msgID, err := client.SendMessage(ctx, "12345", "hello", "HTML", nil, nil)
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if msgID != 42 {
		t.Errorf("expected message_id 42, got %d", msgID)
	}
	if receivedBody["chat_id"] != "12345" {
		t.Errorf("expected chat_id '12345', got %v", receivedBody["chat_id"])
	}
	if receivedBody["text"] != "hello" {
		t.Errorf("expected text 'hello', got %v", receivedBody["text"])
	}
}

func TestSendMessage_WithReplyAndKeyboard(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage(`{"message_id": 43, "chat": {"id": 12345}}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	replyTo := 10
	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "btn", CallbackData: "cb"}},
		},
	}

	msgID, err := client.SendMessage(ctx, "12345", "test", "HTML", keyboard, &replyTo)
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if msgID != 43 {
		t.Errorf("expected message_id 43, got %d", msgID)
	}

	// Check reply_to_message_id was sent
	if val, ok := receivedBody["reply_to_message_id"]; !ok {
		t.Error("expected reply_to_message_id in payload")
	} else if val.(float64) != 10 {
		t.Errorf("expected reply_to_message_id 10, got %v", val)
	}

	// Check reply_markup was sent
	if _, ok := receivedBody["reply_markup"]; !ok {
		t.Error("expected reply_markup in payload")
	}
}

func TestEditMessageText(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/editMessageText") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)

		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage(`true`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	err := client.EditMessageText(ctx, "12345", 42, "new text", "HTML", nil)
	if err != nil {
		t.Fatalf("EditMessageText error: %v", err)
	}

	if receivedBody["chat_id"] != "12345" {
		t.Errorf("expected chat_id '12345', got %v", receivedBody["chat_id"])
	}
	if receivedBody["message_id"].(float64) != 42 {
		t.Errorf("expected message_id 42, got %v", receivedBody["message_id"])
	}
	if receivedBody["text"] != "new text" {
		t.Errorf("expected text 'new text', got %v", receivedBody["text"])
	}
}

func TestAnswerCallbackQuery(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := APIResponse{OK: true, Result: json.RawMessage(`true`)}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.AnswerCallbackQuery(context.Background(), "query-123", "done", false)
	if err != nil {
		t.Fatalf("AnswerCallbackQuery error: %v", err)
	}

	if receivedBody["callback_query_id"] != "query-123" {
		t.Errorf("expected callback_query_id 'query-123', got %v", receivedBody["callback_query_id"])
	}
}

func TestSetWebhook(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := APIResponse{OK: true, Result: json.RawMessage(`true`)}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.SetWebhook(context.Background(), "https://example.com/webhook", "secret-123")
	if err != nil {
		t.Fatalf("SetWebhook error: %v", err)
	}

	if receivedBody["url"] != "https://example.com/webhook" {
		t.Errorf("expected url, got %v", receivedBody["url"])
	}
	if receivedBody["secret_token"] != "secret-123" {
		t.Errorf("expected secret_token, got %v", receivedBody["secret_token"])
	}
}

func TestPostAlert(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage(`{"message_id": 100, "chat": {"id": 12345}}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	info := CheckAlertInfo{
		UUID:      "abc123",
		Name:      "my-check",
		Project:   "myproject",
		Group:     "production",
		CheckType: "http",
		Message:   "timeout",
		Severity:  "critical",
		IsHealthy: false,
	}

	msgID, err := client.PostAlert(context.Background(), "12345", info)
	if err != nil {
		t.Fatalf("PostAlert error: %v", err)
	}
	if msgID != 100 {
		t.Errorf("expected message_id 100, got %d", msgID)
	}

	// Should have parse_mode HTML
	if receivedBody["parse_mode"] != "HTML" {
		t.Errorf("expected parse_mode HTML, got %v", receivedBody["parse_mode"])
	}

	// Should have reply_markup (keyboard)
	if _, ok := receivedBody["reply_markup"]; !ok {
		t.Error("expected reply_markup in PostAlert payload")
	}

	// Should contain alert text
	text, ok := receivedBody["text"].(string)
	if !ok {
		t.Fatal("expected text field in payload")
	}
	if !strings.Contains(text, "ALERT: my-check") {
		t.Errorf("text should contain alert header, got: %s", text)
	}
}

func TestSendResolve(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage(`{"message_id": 101, "chat": {"id": 12345}}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	info := CheckAlertInfo{
		UUID:          "abc123",
		Name:          "my-check",
		Project:       "myproject",
		Group:         "production",
		CheckType:     "http",
		IsHealthy:     true,
		OriginalError: "was broken",
	}

	err := client.SendResolve(context.Background(), info, 42, "12345")
	if err != nil {
		t.Fatalf("SendResolve error: %v", err)
	}

	// Should make 2 API calls: editMessageText + sendMessage
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			OK:          false,
			Description: "Bad Request: chat not found",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.SendMessage(context.Background(), "99999", "test", "", nil, nil)
	if err == nil {
		t.Fatal("expected error for API ok=false")
	}
	if !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("error should contain API description, got: %v", err)
	}
}

func TestNewTelegramClient(t *testing.T) {
	client := NewTelegramClient("bot-token", "secret", "12345")

	if client.DefaultChatID() != "12345" {
		t.Errorf("expected default chat ID '12345', got %q", client.DefaultChatID())
	}
	if client.SecretToken() != "secret" {
		t.Errorf("expected secret token 'secret', got %q", client.SecretToken())
	}
	if !strings.Contains(client.baseURL, "bot-token") {
		t.Errorf("expected baseURL to contain bot token, got %q", client.baseURL)
	}
}

func TestSendSilenceConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage(`{"message_id": 50, "chat": {"id": 12345}}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.SendSilenceConfirmation(context.Background(), "12345", 42, "check", "abc", "1h", "john")
	if err != nil {
		t.Fatalf("SendSilenceConfirmation error: %v", err)
	}
}

func TestSendUnsilenceConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			OK:     true,
			Result: json.RawMessage(`{"message_id": 51, "chat": {"id": 12345}}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.SendUnsilenceConfirmation(context.Background(), "12345", 42, "check", "abc", "john")
	if err != nil {
		t.Fatalf("SendUnsilenceConfirmation error: %v", err)
	}
}
