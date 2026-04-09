// SPDX-License-Identifier: BUSL-1.1

package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a DiscordClient pointed at the given httptest.Server.
func newTestClient(serverURL string) *DiscordClient {
	c := NewDiscordClient("test-bot-token", "test-app-id", "default-channel")
	c.baseURL = serverURL
	return c
}

func TestSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/channels/123456/messages", r.URL.Path)

		// Verify auth header
		assert.Equal(t, "Bot test-bot-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Decode and verify payload
		var payload MessagePayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "hello world", payload.Content)

		// Return a message response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Message{
			ID:        "msg-001",
			ChannelID: "123456",
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	msg, err := client.SendMessage(context.Background(), "123456", MessagePayload{
		Content: "hello world",
	})

	require.NoError(t, err)
	assert.Equal(t, "msg-001", msg.ID)
	assert.Equal(t, "123456", msg.ChannelID)
}

func TestSendMessageWithEmbeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload MessagePayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		require.Len(t, payload.Embeds, 1)
		assert.Equal(t, "Test Alert", payload.Embeds[0].Title)
		assert.Equal(t, ColorRed, payload.Embeds[0].Color)
		assert.Len(t, payload.Embeds[0].Fields, 2)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Message{ID: "msg-002", ChannelID: "ch-1"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	msg, err := client.SendMessage(context.Background(), "ch-1", MessagePayload{
		Embeds: []Embed{{
			Title: "Test Alert",
			Color: ColorRed,
			Fields: []EmbedField{
				{Name: "Project", Value: "myproj", Inline: true},
				{Name: "Status", Value: "unhealthy", Inline: true},
			},
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, "msg-002", msg.ID)
}

func TestEditMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/channels/ch-1/messages/msg-001", r.URL.Path)
		assert.Equal(t, "Bot test-bot-token", r.Header.Get("Authorization"))

		var payload MessagePayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "updated content", payload.Content)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.EditMessage(context.Background(), "ch-1", "msg-001", MessagePayload{
		Content: "updated content",
	})

	assert.NoError(t, err)
}

func TestCreateThread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/channels/ch-1/messages/msg-001/threads", r.URL.Path)
		assert.Equal(t, "Bot test-bot-token", r.Header.Get("Authorization"))

		var body struct {
			Name                string `json:"name"`
			AutoArchiveDuration int    `json:"auto_archive_duration"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "Alert Thread", body.Name)
		assert.Equal(t, 1440, body.AutoArchiveDuration)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Channel{
			ID:   "thread-001",
			Name: "Alert Thread",
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ch, err := client.CreateThread(context.Background(), "ch-1", "msg-001", "Alert Thread")

	require.NoError(t, err)
	assert.Equal(t, "thread-001", ch.ID)
	assert.Equal(t, "Alert Thread", ch.Name)
}

func TestSendThreadReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		// Thread reply uses the thread ID as the channel
		assert.Equal(t, "/channels/thread-001/messages", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Message{ID: "msg-reply-001", ChannelID: "thread-001"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	msg, err := client.SendThreadReply(context.Background(), "thread-001", MessagePayload{
		Content: "thread reply",
	})

	require.NoError(t, err)
	assert.Equal(t, "msg-reply-001", msg.ID)
	assert.Equal(t, "thread-001", msg.ChannelID)
}

func TestRespondToInteraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/interactions/int-001/int-token-xyz/callback", r.URL.Path)
		assert.Equal(t, "Bot test-bot-token", r.Header.Get("Authorization"))

		var resp InteractionResponse
		err := json.NewDecoder(r.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, InteractionResponseTypeMessage, resp.Type)
		require.NotNil(t, resp.Data)
		assert.Equal(t, "acknowledged", resp.Data.Content)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.RespondToInteraction(context.Background(), "int-001", "int-token-xyz", InteractionResponse{
		Type: InteractionResponseTypeMessage,
		Data: &InteractionCallbackData{
			Content: "acknowledged",
		},
	})

	assert.NoError(t, err)
}

func TestRateLimitRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0.01") // 10ms
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"message": "rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Message{ID: "msg-retry", ChannelID: "ch-1"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	msg, err := client.SendMessage(context.Background(), "ch-1", MessagePayload{
		Content: "test retry",
	})

	require.NoError(t, err)
	assert.Equal(t, "msg-retry", msg.ID)
	assert.Equal(t, 2, attempts)
}

func TestRateLimitExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0.01")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message": "rate limited"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SendMessage(context.Background(), "ch-1", MessagePayload{
		Content: "test",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited after")
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "Missing Permissions"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SendMessage(context.Background(), "ch-1", MessagePayload{
		Content: "test",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.Contains(t, err.Error(), "Missing Permissions")
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60") // long retry
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := newTestClient(server.URL)
	_, err := client.SendMessage(ctx, "ch-1", MessagePayload{Content: "test"})

	require.Error(t, err)
}

func TestDefaultChannelID(t *testing.T) {
	client := NewDiscordClient("token", "app-id", "my-channel")
	assert.Equal(t, "my-channel", client.DefaultChannelID())
	assert.Equal(t, "app-id", client.AppID())
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.5", "1.5s"},
		{"0.1", "100ms"},
		{"", "1s"},
		{"invalid", "1s"},
	}

	for _, tt := range tests {
		d := parseRetryAfter(tt.input)
		assert.Equal(t, tt.expected, d.String(), "input: %q", tt.input)
	}
}

func TestBuildAlertMessage(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "test-uuid-123",
		Name:      "API Health",
		Project:   "myproject",
		Group:     "production",
		CheckType: "http",
		Frequency: "5m",
		Message:   "connection refused",
		IsHealthy: false,
		Severity:  "critical",
		Target:    "https://api.example.com/health",
	}

	payload := BuildAlertMessage(info)

	require.Len(t, payload.Embeds, 1)
	embed := payload.Embeds[0]
	assert.Contains(t, embed.Title, "ALERT: API Health")
	assert.Equal(t, ColorRed, embed.Color)
	assert.Contains(t, embed.Description, "connection refused")

	// Check fields
	fieldNames := make([]string, len(embed.Fields))
	for i, f := range embed.Fields {
		fieldNames[i] = f.Name
	}
	assert.Contains(t, fieldNames, "Project")
	assert.Contains(t, fieldNames, "Group")
	assert.Contains(t, fieldNames, "Type")
	assert.Contains(t, fieldNames, "Target")
	assert.Contains(t, fieldNames, "UUID")

	// Check buttons
	require.Len(t, payload.Components, 1)
	row := payload.Components[0]
	assert.Equal(t, ComponentTypeActionRow, row.Type)
	require.Len(t, row.Components, 3)

	// Verify button custom IDs
	assert.Equal(t, "checker_ack_test-uuid-123", row.Components[0].CustomID)
	assert.Equal(t, "checker_silence_test-uuid-123_1h", row.Components[1].CustomID)
	assert.Equal(t, "checker_silence_test-uuid-123_24h", row.Components[2].CustomID)
}

func TestBuildResolveMessage(t *testing.T) {
	info := CheckAlertInfo{
		Name:    "API Health",
		Message: "check recovered successfully",
	}

	payload := BuildResolveMessage(info)

	require.Len(t, payload.Embeds, 1)
	embed := payload.Embeds[0]
	assert.Contains(t, embed.Title, "RESOLVED")
	assert.Equal(t, ColorGreen, embed.Color)
	assert.Equal(t, "check recovered successfully", embed.Description)
	assert.Empty(t, payload.Components) // No buttons
}

func TestBuildResolveMessageDefaultBody(t *testing.T) {
	info := CheckAlertInfo{
		Name: "API Health",
	}

	payload := BuildResolveMessage(info)
	assert.Equal(t, "Check is healthy again.", payload.Embeds[0].Description)
}

func TestBuildResolvedAlertMessage(t *testing.T) {
	info := CheckAlertInfo{
		UUID:          "test-uuid-456",
		Name:          "DB Check",
		Project:       "myproject",
		Group:         "staging",
		CheckType:     "pgsql",
		OriginalError: "connection refused",
	}

	payload := BuildResolvedAlertMessage(info, "admin-user")

	require.Len(t, payload.Embeds, 1)
	embed := payload.Embeds[0]
	assert.Contains(t, embed.Title, "RESOLVED: DB Check")
	assert.Equal(t, ColorGreen, embed.Color)
	assert.Contains(t, embed.Description, "connection refused")
	assert.Empty(t, payload.Components) // No action buttons

	// Check that resolvedBy is in fields
	found := false
	for _, f := range embed.Fields {
		if f.Name == "Resolved By" {
			found = true
			assert.Equal(t, "admin-user", f.Value)
		}
	}
	assert.True(t, found, "should have Resolved By field")
}

func TestSendMessageWithComponents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload MessagePayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		require.Len(t, payload.Components, 1)
		row := payload.Components[0]
		assert.Equal(t, ComponentTypeActionRow, row.Type)
		require.Len(t, row.Components, 2)
		assert.Equal(t, ComponentTypeButton, row.Components[0].Type)
		assert.Equal(t, "btn-1", row.Components[0].CustomID)
		assert.Equal(t, ButtonStylePrimary, row.Components[0].Style)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Message{ID: "msg-comp", ChannelID: "ch-1"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SendMessage(context.Background(), "ch-1", MessagePayload{
		Content: "Click a button",
		Components: []ActionRow{{
			Type: ComponentTypeActionRow,
			Components: []Component{
				{Type: ComponentTypeButton, Label: "OK", Style: ButtonStylePrimary, CustomID: "btn-1"},
				{Type: ComponentTypeButton, Label: "Visit", Style: ButtonStyleLink, URL: "https://example.com"},
			},
		}},
	})

	require.NoError(t, err)
}

func TestHelperFunctions(t *testing.T) {
	// typeEmoji
	assert.Equal(t, "🌐", typeEmoji("http"))
	assert.Equal(t, "🔌", typeEmoji("tcp"))
	assert.Equal(t, "📡", typeEmoji("icmp"))
	assert.Equal(t, "🐘", typeEmoji("pgsql"))
	assert.Equal(t, "🐘", typeEmoji("postgresql"))
	assert.Equal(t, "🐬", typeEmoji("mysql"))
	assert.Equal(t, "⏳", typeEmoji("passive"))
	assert.Equal(t, "🔍", typeEmoji("unknown"))

	// severityEmoji
	assert.Equal(t, "🟢", severityEmoji(CheckAlertInfo{IsHealthy: true}))
	assert.Equal(t, "🔴", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: "critical"}))
	assert.Equal(t, "🟡", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: "degraded"}))
	assert.Equal(t, "🔴", severityEmoji(CheckAlertInfo{IsHealthy: false}))

	// statusText
	assert.True(t, strings.Contains(statusText(true), "Healthy"))
	assert.True(t, strings.Contains(statusText(false), "Unhealthy"))
}
