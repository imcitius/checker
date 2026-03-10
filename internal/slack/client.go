package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	slackBaseURL = "https://slack.com/api/"
)

// SlackClient provides methods to interact with the Slack Web API.
type SlackClient struct {
	botToken   string
	httpClient *http.Client
}

// NewSlackClient creates a new SlackClient with the given bot token.
func NewSlackClient(botToken string) *SlackClient {
	return &SlackClient{
		botToken: botToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PostAlert posts an alert message with a colored attachment to the specified channel.
// It returns the message timestamp (ts) which can be used for threading.
func (c *SlackClient) PostAlert(ctx context.Context, channelID string, check CheckAlertInfo) (string, error) {
	color := "danger"
	status := "FAILING"
	if check.IsHealthy {
		color = "good"
		status = "HEALTHY"
	}

	att := attachment{
		Color:    color,
		Fallback: fmt.Sprintf("[%s] %s: %s", status, check.Name, check.Message),
		Fields: []attachmentField{
			{Title: "Check Name", Value: check.Name, Short: true},
			{Title: "Project", Value: check.Project, Short: true},
			{Title: "Group", Value: check.Group, Short: true},
			{Title: "Status", Value: status, Short: true},
			{Title: "Error Message", Value: check.Message, Short: false},
			{Title: "Timestamp", Value: time.Now().UTC().Format(time.RFC3339), Short: true},
		},
	}

	req := slackRequest{
		Channel:     channelID,
		Attachments: []attachment{att},
	}

	resp, err := c.callAPI(ctx, "chat.postMessage", req)
	if err != nil {
		return "", fmt.Errorf("PostAlert: %w", err)
	}

	return resp.TS, nil
}

// PostResolution posts a resolution message as a thread reply to an existing alert.
func (c *SlackClient) PostResolution(ctx context.Context, channelID, threadTs string, check CheckAlertInfo) error {
	att := attachment{
		Color:    "good",
		Fallback: fmt.Sprintf("Resolved: Check %s is healthy again", check.Name),
		Fields: []attachmentField{
			{Title: "Resolution", Value: fmt.Sprintf("Resolved: Check %s is healthy again", check.Name), Short: false},
			{Title: "Check Name", Value: check.Name, Short: true},
			{Title: "Project", Value: check.Project, Short: true},
		},
	}

	req := slackRequest{
		Channel:     channelID,
		ThreadTS:    threadTs,
		Attachments: []attachment{att},
	}

	_, err := c.callAPI(ctx, "chat.postMessage", req)
	if err != nil {
		return fmt.Errorf("PostResolution: %w", err)
	}

	return nil
}

// UpdateMessageColor updates the color of an existing message's attachment.
// It preserves the original message content, only changing the attachment color.
func (c *SlackClient) UpdateMessageColor(ctx context.Context, channelID, messageTs string, color string, check CheckAlertInfo) error {
	status := "FAILING"
	if check.IsHealthy {
		status = "HEALTHY"
	}

	att := attachment{
		Color:    color,
		Fallback: fmt.Sprintf("[%s] %s: %s", status, check.Name, check.Message),
		Fields: []attachmentField{
			{Title: "Check Name", Value: check.Name, Short: true},
			{Title: "Project", Value: check.Project, Short: true},
			{Title: "Group", Value: check.Group, Short: true},
			{Title: "Status", Value: status, Short: true},
			{Title: "Error Message", Value: check.Message, Short: false},
			{Title: "Timestamp", Value: time.Now().UTC().Format(time.RFC3339), Short: true},
		},
	}

	req := slackRequest{
		Channel:     channelID,
		TS:          messageTs,
		Attachments: []attachment{att},
	}

	_, err := c.callAPI(ctx, "chat.update", req)
	if err != nil {
		return fmt.Errorf("UpdateMessageColor: %w", err)
	}

	return nil
}

// PostEphemeral posts an ephemeral message visible only to the specified user.
// Used for confirmation messages after silence commands.
func (c *SlackClient) PostEphemeral(ctx context.Context, channelID, userID, text string) error {
	req := slackRequest{
		Channel: channelID,
		User:    userID,
		Text:    text,
	}

	_, err := c.callAPI(ctx, "chat.postEphemeral", req)
	if err != nil {
		return fmt.Errorf("PostEphemeral: %w", err)
	}

	return nil
}

// callAPI makes an authenticated request to the Slack Web API.
func (c *SlackClient) callAPI(ctx context.Context, method string, payload interface{}) (*slackResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := slackBaseURL + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		log.Warnf("Slack API rate limited on %s", method)
		return nil, fmt.Errorf("rate limited by Slack API")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var slackResp slackResponse
	if err := json.Unmarshal(respBody, &slackResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !slackResp.OK {
		return nil, fmt.Errorf("Slack API error: %s", slackResp.Error)
	}

	return &slackResp, nil
}
