package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Discord embed color constants.
const (
	ColorRed    = 15158332 // failure / DOWN
	ColorGreen  = 3066993  // recovery / RESOLVED
	ColorYellow = 16776960 // warning
)

// DiscordEmbedField represents a field inside a Discord embed.
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbed represents a single Discord embed object.
type DiscordEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
}

// DiscordPayload is the top-level payload sent to Discord webhooks.
type DiscordPayload struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

// DiscordAlertParams holds the parameters for building a Discord alert.
type DiscordAlertParams struct {
	CheckName string
	Project   string
	CheckType string
	Message   string
	IsDown    bool // true = DOWN, false = RESOLVED
}

// BuildDiscordPayload constructs a DiscordPayload for a check status change.
func BuildDiscordPayload(params DiscordAlertParams) DiscordPayload {
	var title string
	var color int

	if params.IsDown {
		title = fmt.Sprintf("🔴 %s is DOWN", params.CheckName)
		color = ColorRed
	} else {
		title = fmt.Sprintf("🟢 %s is RESOLVED", params.CheckName)
		color = ColorGreen
	}

	embed := DiscordEmbed{
		Title:       title,
		Description: params.Message,
		Color:       color,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []DiscordEmbedField{
			{Name: "Project", Value: params.Project, Inline: true},
			{Name: "Type", Value: params.CheckType, Inline: true},
		},
	}

	return DiscordPayload{Embeds: []DiscordEmbed{embed}}
}

// SendDiscordAlert posts a Discord webhook message with an embed payload.
func SendDiscordAlert(webhookURL string, payload DiscordPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send Discord alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord alert failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
