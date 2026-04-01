package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SlackWebhookAlerter implements the Alerter interface for Slack webhooks.
type SlackWebhookAlerter struct {
	WebhookURL string
}

func (a *SlackWebhookAlerter) Type() string { return "slack" }

func (a *SlackWebhookAlerter) SendAlert(p AlertPayload) error {
	msg := fmt.Sprintf("[%s] Check %s (%s/%s) failed: %s",
		strings.ToUpper(p.Severity), p.CheckName, p.Project, p.CheckGroup, p.Message)
	return SendSlackAlert(a.WebhookURL, msg)
}

func (a *SlackWebhookAlerter) SendRecovery(p RecoveryPayload) error {
	msg := fmt.Sprintf("RECOVERY: Check %s (%s/%s) is healthy again", p.CheckName, p.Project, p.CheckGroup)
	return SendSlackAlert(a.WebhookURL, msg)
}

func newSlackWebhookAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing slack config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("slack requires webhook_url")
	}
	return &SlackWebhookAlerter{WebhookURL: cfg.WebhookURL}, nil
}

func init() {
	RegisterAlerter("slack", newSlackWebhookAlerter)
	RegisterAlerter("slack_webhook", newSlackWebhookAlerter)
}

// Note: SendSlackAlert uses postJSON, but SendSlackAppTest uses custom logic
// (parses response body for Slack API errors), so it stays manual.

type SlackPayload struct {
	Text string `json:"text"`
}

// SendSlackAlert sends an alert to Slack via a webhook URL or Slack API
func SendSlackAlert(webhookURL, message string) error {
	payload := SlackPayload{Text: message}
	if err := postJSON(webhookURL, payload, nil); err != nil {
		return fmt.Errorf("slack alert: %w", err)
	}
	return nil
}

// SendSlackAppTest sends a test message using the Slack Bot API (chat.postMessage).
func SendSlackAppTest(botToken, channelID, message string) error {
	// Strip # prefix if user entered channel name like "#general"
	channelID = strings.TrimPrefix(channelID, "#")

	payload := map[string]string{
		"channel": channelID,
		"text":    message,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+botToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse Slack response: %w", err)
	}
	if !result.OK {
		if result.Error == "channel_not_found" {
			return fmt.Errorf("slack API error: channel_not_found — use a channel ID (e.g. C01ABCDEF) or ensure the bot is invited to the channel")
		}
		return fmt.Errorf("slack API error: %s", result.Error)
	}
	return nil
}
