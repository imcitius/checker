package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertChannel_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ch := AlertChannel{
		ID:        1,
		Name:      "my-telegram",
		Type:      "telegram",
		Config:    json.RawMessage(`{"bot_token":"abc123","chat_id":"-100123"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(ch)
	assert.NoError(t, err)

	var got AlertChannel
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, ch.ID, got.ID)
	assert.Equal(t, ch.Name, got.Name)
	assert.Equal(t, ch.Type, got.Type)
	assert.JSONEq(t, string(ch.Config), string(got.Config))
}

func TestAlertChannel_ZeroValue(t *testing.T) {
	var ch AlertChannel
	assert.Equal(t, 0, ch.ID)
	assert.Empty(t, ch.Name)
	assert.Empty(t, ch.Type)
	assert.Nil(t, ch.Config)
	assert.True(t, ch.CreatedAt.IsZero())
}

func TestMaskSensitiveConfig(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		config      string
		checkField  string
		expectMask  bool
		expectFull  bool // expect "****" (short value)
	}{
		{
			name:        "telegram bot_token masked",
			channelType: "telegram",
			config:      `{"bot_token":"1234567890:ABCdefGHI","chat_id":"-100123"}`,
			checkField:  "bot_token",
			expectMask:  true,
		},
		{
			name:        "slack bot_token masked",
			channelType: "slack",
			config:      `{"bot_token":"xoxb-1234567890","signing_secret":"abcdef123456"}`,
			checkField:  "bot_token",
			expectMask:  true,
		},
		{
			name:        "email smtp_password masked",
			channelType: "email",
			config:      `{"smtp_password":"mysecretpassword","smtp_host":"smtp.example.com"}`,
			checkField:  "smtp_password",
			expectMask:  true,
		},
		{
			name:        "short value gets full mask",
			channelType: "telegram",
			config:      `{"bot_token":"abcd","chat_id":"123"}`,
			checkField:  "bot_token",
			expectFull:  true,
		},
		{
			name:        "unknown channel type returns raw",
			channelType: "unknown",
			config:      `{"secret":"should_not_change"}`,
			checkField:  "secret",
		},
		{
			name:        "empty config field not masked",
			channelType: "telegram",
			config:      `{"chat_id":"123"}`,
			checkField:  "bot_token",
		},
		{
			name:        "discord bot_token masked",
			channelType: "discord",
			config:      `{"bot_token":"MTIzNDU2Nzg5MDEyMzQ1Njc4OQ.abc.xyz"}`,
			checkField:  "bot_token",
			expectMask:  true,
		},
		{
			name:        "pagerduty routing_key masked",
			channelType: "pagerduty",
			config:      `{"routing_key":"e93facc04764012d7bfb002500d5d1a6"}`,
			checkField:  "routing_key",
			expectMask:  true,
		},
		{
			name:        "opsgenie api_key masked",
			channelType: "opsgenie",
			config:      `{"api_key":"eb243592-faa2-4ba2-a551q-1afdf565c889"}`,
			checkField:  "api_key",
			expectMask:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := json.RawMessage(tt.config)
			masked := MaskSensitiveConfig(tt.channelType, raw)

			var result map[string]interface{}
			err := json.Unmarshal(masked, &result)
			assert.NoError(t, err)

			if tt.expectMask {
				val, ok := result[tt.checkField].(string)
				assert.True(t, ok)
				assert.Contains(t, val, "****")
				assert.True(t, len(val) < len(tt.config)) // masked should be shorter or contain ****
			}
			if tt.expectFull {
				val := result[tt.checkField].(string)
				assert.Equal(t, "****", val)
			}
		})
	}
}

func TestMaskSensitiveConfig_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`not-valid-json`)
	masked := MaskSensitiveConfig("telegram", raw)
	assert.Equal(t, raw, masked)
}

func TestMaskSensitiveConfig_EmptyString(t *testing.T) {
	raw := json.RawMessage(`{"bot_token":""}`)
	masked := MaskSensitiveConfig("telegram", raw)
	var result map[string]interface{}
	err := json.Unmarshal(masked, &result)
	assert.NoError(t, err)
	// Empty string should not be masked (len == 0)
	assert.Equal(t, "", result["bot_token"])
}

func TestSensitiveFields_Coverage(t *testing.T) {
	expectedTypes := []string{"telegram", "slack", "slack_webhook", "email", "discord", "teams", "pagerduty", "opsgenie", "ntfy"}
	for _, ct := range expectedTypes {
		fields, ok := SensitiveFields[ct]
		assert.True(t, ok, "missing sensitive fields for channel type: %s", ct)
		assert.NotEmpty(t, fields, "empty sensitive fields for channel type: %s", ct)
	}
}
