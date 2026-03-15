package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackPayload struct {
	Text string `json:"text"`
}

// SendSlackAlert sends an alert to Slack via a webhook URL or Slack API
func SendSlackAlert(webhookURL, message string) error {
	payload := SlackPayload{Text: message}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %v", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send Slack alert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("slack alert failed with status %d", resp.StatusCode)
	}
	return nil
}

// SendSlackAppTest sends a test message using the Slack Bot API (chat.postMessage).
func SendSlackAppTest(botToken, channelID, message string) error {
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
		return fmt.Errorf("slack API error: %s", result.Error)
	}
	return nil
}
