package models

import (
	"encoding/json"
	"time"
)

// AlertChannel represents a configured notification channel (e.g. Slack, Telegram, Email).
type AlertChannel struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// SensitiveFields maps channel type to the config keys that contain secrets.
var SensitiveFields = map[string][]string{
	"telegram":  {"bot_token"},
	"slack":         {"bot_token", "signing_secret"},
	"slack_webhook": {"webhook_url"},
	"email":     {"smtp_password"},
	"discord":     {"webhook_url"},
	"discord_bot": {"bot_token"},
	"teams":       {"webhook_url"},
	"pagerduty":   {"routing_key"},
	"opsgenie":    {"api_key"},
	"ntfy":        {"token", "password"},
}

// MaskSensitiveConfig returns a copy of config with sensitive fields masked.
func MaskSensitiveConfig(channelType string, raw json.RawMessage) json.RawMessage {
	fields, ok := SensitiveFields[channelType]
	if !ok || len(fields) == 0 {
		return raw
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return raw
	}

	for _, field := range fields {
		if val, exists := cfg[field]; exists {
			if s, ok := val.(string); ok && len(s) > 4 {
				cfg[field] = s[:2] + "****" + s[len(s)-2:]
			} else if s, ok := val.(string); ok && len(s) > 0 {
				cfg[field] = "****"
			}
		}
	}

	masked, err := json.Marshal(cfg)
	if err != nil {
		return raw
	}
	return masked
}
