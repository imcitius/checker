package alerts

import (
	"encoding/json"
	"fmt"
)

// discordBotAlerter is a stub Alerter registered so that IsRegisteredType("discord")
// returns true, allowing discord channels to be created via the UI.
// Actual alerting is handled by the DiscordAppAlerter; the ownedTypes mechanism
// in sendAlerts() skips this stub at runtime.
type discordBotAlerter struct{}

func (a *discordBotAlerter) Type() string                        { return "discord" }
func (a *discordBotAlerter) SendAlert(_ AlertPayload) error      { return nil }
func (a *discordBotAlerter) SendRecovery(_ RecoveryPayload) error { return nil }

func newDiscordBotAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg struct {
		BotToken string `json:"bot_token"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing discord config: %w", err)
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("discord requires bot_token")
	}
	return &discordBotAlerter{}, nil
}

func init() {
	RegisterAlerter("discord", newDiscordBotAlerter)
}
