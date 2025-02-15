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