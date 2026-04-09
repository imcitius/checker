// SPDX-License-Identifier: BUSL-1.1

package alerts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/imcitius/checker/internal/discord"
)

// discordBotAlerter implements the Alerter interface for Discord using the bot API.
// It reads bot_token, app_id, and default_channel from the DB channel config (json.RawMessage)
// and sends real alert/recovery messages with rich embeds.
//
// When the YAML-configured DiscordAppAlerter is also present, it takes ownership of
// the "discord" type via the ownedTypes mechanism, providing the full experience
// (threads, buttons, interaction handling). This registry alerter provides basic
// embed-based alerts for users who only configure Discord via the UI.
type discordBotAlerter struct {
	client    *discord.DiscordClient
	channelID string
}

func (a *discordBotAlerter) Type() string { return "discord" }

func (a *discordBotAlerter) SendAlert(p AlertPayload) error {
	payload := discord.BuildAlertMessage(discord.CheckAlertInfo{
		UUID:      p.CheckUUID,
		Name:      p.CheckName,
		Project:   p.Project,
		Group:     p.CheckGroup,
		CheckType: p.CheckType,
		Message:   p.Message,
		IsHealthy: false,
	})
	_, err := a.client.SendMessage(context.Background(), a.channelID, payload)
	if err != nil {
		return fmt.Errorf("discord alert: %w", err)
	}
	return nil
}

func (a *discordBotAlerter) SendRecovery(p RecoveryPayload) error {
	payload := discord.BuildResolveMessage(discord.CheckAlertInfo{
		UUID:      p.CheckUUID,
		Name:      p.CheckName,
		Project:   p.Project,
		Group:     p.CheckGroup,
		CheckType: p.CheckType,
		IsHealthy: true,
	})
	_, err := a.client.SendMessage(context.Background(), a.channelID, payload)
	if err != nil {
		return fmt.Errorf("discord recovery: %w", err)
	}
	return nil
}

func newDiscordBotAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg struct {
		BotToken       string `json:"bot_token"`
		AppID          string `json:"app_id"`
		DefaultChannel string `json:"default_channel"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing discord config: %w", err)
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("discord requires bot_token")
	}
	if cfg.DefaultChannel == "" {
		return nil, fmt.Errorf("discord requires default_channel")
	}
	client := discord.NewDiscordClient(cfg.BotToken, cfg.AppID, cfg.DefaultChannel)
	return &discordBotAlerter{client: client, channelID: cfg.DefaultChannel}, nil
}

func init() {
	RegisterAlerter("discord", newDiscordBotAlerter)
}
