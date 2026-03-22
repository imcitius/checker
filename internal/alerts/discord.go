package alerts

import (
	"encoding/json"
	"fmt"
	"time"
)

// DiscordAlerter implements the Alerter interface for Discord webhooks.
type DiscordAlerter struct {
	WebhookURL string
}

func (a *DiscordAlerter) Type() string { return "discord" }

func (a *DiscordAlerter) SendAlert(p AlertPayload) error {
	payload := BuildDiscordPayload(DiscordAlertParams{
		CheckName: p.CheckName,
		Project:   p.Project,
		CheckType: p.CheckType,
		Message:   p.Message,
		IsDown:    true,
	})
	return SendDiscordAlert(a.WebhookURL, payload)
}

func (a *DiscordAlerter) SendRecovery(p RecoveryPayload) error {
	payload := BuildDiscordPayload(DiscordAlertParams{
		CheckName: p.CheckName,
		Project:   p.Project,
		CheckType: p.CheckType,
		IsDown:    false,
	})
	return SendDiscordAlert(a.WebhookURL, payload)
}

func newDiscordAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing discord config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("discord requires webhook_url")
	}
	return &DiscordAlerter{WebhookURL: cfg.WebhookURL}, nil
}

func init() {
	RegisterAlerter("discord", newDiscordAlerter)
}

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
	if err := postJSON(webhookURL, payload, nil); err != nil {
		return fmt.Errorf("discord alert: %w", err)
	}
	return nil
}
