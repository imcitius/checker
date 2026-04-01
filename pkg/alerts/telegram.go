package alerts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// TelegramAlerter implements the Alerter interface for Telegram.
type TelegramAlerter struct {
	BotToken string
	ChatID   string
}

func (a *TelegramAlerter) Type() string { return "telegram" }

func (a *TelegramAlerter) SendAlert(p AlertPayload) error {
	msg := fmt.Sprintf("[%s] Check %s (%s/%s) failed: %s",
		strings.ToUpper(p.Severity), p.CheckName, p.Project, p.CheckGroup, p.Message)
	return SendTelegramAlert(a.BotToken, a.ChatID, msg)
}

func (a *TelegramAlerter) SendRecovery(p RecoveryPayload) error {
	msg := fmt.Sprintf("RECOVERY: Check %s (%s/%s) is healthy again", p.CheckName, p.Project, p.CheckGroup)
	return SendTelegramAlert(a.BotToken, a.ChatID, msg)
}

func newTelegramAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg struct {
		BotToken string `json:"bot_token"`
		ChatID   string `json:"chat_id"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing telegram config: %w", err)
	}
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return nil, fmt.Errorf("telegram requires bot_token and chat_id")
	}
	return &TelegramAlerter{BotToken: cfg.BotToken, ChatID: cfg.ChatID}, nil
}

func init() {
	RegisterAlerter("telegram", newTelegramAlerter)
}

// SendTelegramAlert sends a message to a Telegram channel using a bot token
func SendTelegramAlert(botToken, chatID, message string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", botToken)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return fmt.Errorf("telegram alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram alert failed with status %d", resp.StatusCode)
	}
	return nil
}
