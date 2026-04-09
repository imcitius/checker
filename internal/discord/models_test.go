// SPDX-License-Identifier: BUSL-1.1

package discord

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, 0xED4245, ColorRed)
	assert.Equal(t, 0x57F287, ColorGreen)
	assert.Equal(t, 0x95A5A6, ColorGray)

	assert.Equal(t, 1, ComponentTypeActionRow)
	assert.Equal(t, 2, ComponentTypeButton)

	assert.Equal(t, 1, ButtonStylePrimary)
	assert.Equal(t, 2, ButtonStyleSecondary)
	assert.Equal(t, 3, ButtonStyleSuccess)
	assert.Equal(t, 4, ButtonStyleDanger)
	assert.Equal(t, 5, ButtonStyleLink)

	assert.Equal(t, 4, InteractionResponseTypeMessage)
	assert.Equal(t, 7, InteractionResponseTypeUpdateMessage)
}

func TestMessagePayload_JSON(t *testing.T) {
	payload := MessagePayload{
		Content: "hello",
		Embeds: []Embed{
			{
				Title:       "Test",
				Description: "desc",
				Color:       ColorRed,
				Timestamp:   "2024-01-01T00:00:00Z",
				Fields: []EmbedField{
					{Name: "field1", Value: "val1", Inline: true},
				},
			},
		},
		Components: []ActionRow{
			{
				Type: ComponentTypeActionRow,
				Components: []Component{
					{
						Type:     ComponentTypeButton,
						Label:    "Click",
						Style:    ButtonStylePrimary,
						CustomID: "btn_1",
					},
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded MessagePayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "hello", decoded.Content)
	require.Len(t, decoded.Embeds, 1)
	assert.Equal(t, "Test", decoded.Embeds[0].Title)
	assert.Equal(t, ColorRed, decoded.Embeds[0].Color)
	require.Len(t, decoded.Embeds[0].Fields, 1)
	assert.Equal(t, "field1", decoded.Embeds[0].Fields[0].Name)
	assert.True(t, decoded.Embeds[0].Fields[0].Inline)

	require.Len(t, decoded.Components, 1)
	require.Len(t, decoded.Components[0].Components, 1)
	assert.Equal(t, "Click", decoded.Components[0].Components[0].Label)
}

func TestMessagePayload_OmitEmpty(t *testing.T) {
	payload := MessagePayload{}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	// Empty payload should produce a minimal JSON object.
	assert.Equal(t, "{}", string(data))
}

func TestMessage_JSON(t *testing.T) {
	raw := `{"id":"12345","channel_id":"67890"}`
	var msg Message
	err := json.Unmarshal([]byte(raw), &msg)
	require.NoError(t, err)
	assert.Equal(t, "12345", msg.ID)
	assert.Equal(t, "67890", msg.ChannelID)
}

func TestChannel_JSON(t *testing.T) {
	raw := `{"id":"111","name":"alerts"}`
	var ch Channel
	err := json.Unmarshal([]byte(raw), &ch)
	require.NoError(t, err)
	assert.Equal(t, "111", ch.ID)
	assert.Equal(t, "alerts", ch.Name)
}

func TestInteractionResponse_JSON(t *testing.T) {
	resp := InteractionResponse{
		Type: InteractionResponseTypeMessage,
		Data: &InteractionCallbackData{
			Content: "ack",
			Embeds:  []Embed{{Title: "Done", Color: ColorGreen}},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded InteractionResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, InteractionResponseTypeMessage, decoded.Type)
	require.NotNil(t, decoded.Data)
	assert.Equal(t, "ack", decoded.Data.Content)
	require.Len(t, decoded.Data.Embeds, 1)
	assert.Equal(t, "Done", decoded.Data.Embeds[0].Title)
}

func TestInteractionResponse_NilData(t *testing.T) {
	resp := InteractionResponse{
		Type: InteractionResponseTypeUpdateMessage,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded InteractionResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Nil(t, decoded.Data)
}

func TestComponent_LinkButton(t *testing.T) {
	comp := Component{
		Type:  ComponentTypeButton,
		Label: "Open",
		Style: ButtonStyleLink,
		URL:   "https://example.com",
	}

	data, err := json.Marshal(comp)
	require.NoError(t, err)

	var decoded Component
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ButtonStyleLink, decoded.Style)
	assert.Equal(t, "https://example.com", decoded.URL)
	assert.Empty(t, decoded.CustomID)
}

func TestTypeEmoji_Models(t *testing.T) {
	tests := []struct {
		checkType string
		expected  string
	}{
		{"http", "\U0001F310"},
		{"tcp", "\U0001F50C"},
		{"icmp", "\U0001F4E1"},
		{"pgsql", "\U0001F418"},
		{"postgresql", "\U0001F418"},
		{"mysql", "\U0001F42C"},
		{"passive", "\u23F3"},
		{"unknown", "\U0001F50D"},
		{"", "\U0001F50D"},
	}

	for _, tt := range tests {
		t.Run(tt.checkType, func(t *testing.T) {
			assert.Equal(t, tt.expected, typeEmoji(tt.checkType))
		})
	}
}

func TestSeverityEmoji_Models(t *testing.T) {
	assert.Equal(t, "\U0001F7E2", severityEmoji(CheckAlertInfo{IsHealthy: true}))
	assert.Equal(t, "\U0001F7E1", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: "degraded"}))
	assert.Equal(t, "\U0001F534", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: "critical"}))
	assert.Equal(t, "\U0001F534", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: ""}))
}

func TestStatusText_Models(t *testing.T) {
	assert.Contains(t, statusText(true), "Healthy")
	assert.Contains(t, statusText(false), "Unhealthy")
}
