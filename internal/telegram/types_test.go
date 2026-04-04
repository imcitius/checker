package telegram

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAlertInfo_Fields(t *testing.T) {
	info := CheckAlertInfo{
		UUID:          "uuid-1",
		Name:          "check-name",
		Project:       "proj",
		Group:         "grp",
		CheckType:     "http",
		Frequency:     "30s",
		Message:       "err msg",
		Severity:      "critical",
		Target:        "https://example.com",
		OriginalError: "original",
		IsHealthy:     false,
	}

	assert.Equal(t, "uuid-1", info.UUID)
	assert.Equal(t, "check-name", info.Name)
	assert.Equal(t, "proj", info.Project)
	assert.Equal(t, "grp", info.Group)
	assert.Equal(t, "http", info.CheckType)
	assert.Equal(t, "30s", info.Frequency)
	assert.Equal(t, "err msg", info.Message)
	assert.Equal(t, "critical", info.Severity)
	assert.Equal(t, "https://example.com", info.Target)
	assert.Equal(t, "original", info.OriginalError)
	assert.False(t, info.IsHealthy)
}

func TestAPIResponse_OK(t *testing.T) {
	raw := `{"ok":true,"result":{"message_id":42},"description":""}`
	var resp APIResponse
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)

	assert.True(t, resp.OK)
	assert.NotNil(t, resp.Result)
	assert.Empty(t, resp.Description)
}

func TestAPIResponse_Error(t *testing.T) {
	raw := `{"ok":false,"description":"Bad Request: chat not found"}`
	var resp APIResponse
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)

	assert.False(t, resp.OK)
	assert.Equal(t, "Bad Request: chat not found", resp.Description)
	assert.Nil(t, resp.Result)
}

func TestTelegramMessage_JSON(t *testing.T) {
	raw := `{"message_id":123,"chat":{"id":456}}`
	var msg TelegramMessage
	err := json.Unmarshal([]byte(raw), &msg)
	require.NoError(t, err)

	assert.Equal(t, 123, msg.MessageID)
	assert.Equal(t, int64(456), msg.Chat.ID)
}

func TestInlineKeyboardMarkup_JSON(t *testing.T) {
	kb := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "Ack", CallbackData: "ack_123"},
				{Text: "Silence", CallbackData: "silence_123"},
			},
		},
	}

	data, err := json.Marshal(kb)
	require.NoError(t, err)

	var decoded InlineKeyboardMarkup
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Len(t, decoded.InlineKeyboard, 1)
	require.Len(t, decoded.InlineKeyboard[0], 2)
	assert.Equal(t, "Ack", decoded.InlineKeyboard[0][0].Text)
	assert.Equal(t, "ack_123", decoded.InlineKeyboard[0][0].CallbackData)
}

func TestUpdate_WithMessage(t *testing.T) {
	raw := `{
		"update_id": 100,
		"message": {
			"message_id": 1,
			"chat": {"id": 999},
			"text": "/start"
		}
	}`

	var upd Update
	err := json.Unmarshal([]byte(raw), &upd)
	require.NoError(t, err)

	assert.Equal(t, 100, upd.UpdateID)
	require.NotNil(t, upd.Message)
	assert.Equal(t, 1, upd.Message.MessageID)
	assert.Equal(t, int64(999), upd.Message.Chat.ID)
	assert.Equal(t, "/start", upd.Message.Text)
	assert.Nil(t, upd.CallbackQuery)
}

func TestUpdate_WithCallbackQuery(t *testing.T) {
	raw := `{
		"update_id": 200,
		"callback_query": {
			"id": "cb-id-1",
			"from": {"id": 42, "username": "testuser", "first_name": "Test"},
			"message": {
				"message_id": 10,
				"chat": {"id": 500},
				"text": "original"
			},
			"data": "ack_check_uuid"
		}
	}`

	var upd Update
	err := json.Unmarshal([]byte(raw), &upd)
	require.NoError(t, err)

	assert.Equal(t, 200, upd.UpdateID)
	assert.Nil(t, upd.Message)
	require.NotNil(t, upd.CallbackQuery)

	cb := upd.CallbackQuery
	assert.Equal(t, "cb-id-1", cb.ID)
	assert.Equal(t, int64(42), cb.From.ID)
	assert.Equal(t, "testuser", cb.From.Username)
	assert.Equal(t, "Test", cb.From.FirstName)
	assert.Equal(t, "ack_check_uuid", cb.Data)

	require.NotNil(t, cb.Message)
	assert.Equal(t, 10, cb.Message.MessageID)
	assert.Equal(t, int64(500), cb.Message.Chat.ID)
}

func TestChat_JSON(t *testing.T) {
	raw := `{"id":-1001234567890}`
	var chat Chat
	err := json.Unmarshal([]byte(raw), &chat)
	require.NoError(t, err)
	assert.Equal(t, int64(-1001234567890), chat.ID)
}

func TestUser_JSON(t *testing.T) {
	raw := `{"id":12345,"username":"john","first_name":"John"}`
	var user User
	err := json.Unmarshal([]byte(raw), &user)
	require.NoError(t, err)

	assert.Equal(t, int64(12345), user.ID)
	assert.Equal(t, "john", user.Username)
	assert.Equal(t, "John", user.FirstName)
}

func TestInlineKeyboardButton_OmitEmpty(t *testing.T) {
	btn := InlineKeyboardButton{Text: "Click"}
	data, err := json.Marshal(btn)
	require.NoError(t, err)

	// callback_data should be omitted when empty.
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)
	_, hasCallback := m["callback_data"]
	assert.False(t, hasCallback)
}
