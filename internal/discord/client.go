// SPDX-License-Identifier: BUSL-1.1

package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// DiscordClient provides methods to interact with the Discord REST API.
type DiscordClient struct {
	botToken   string
	appID      string // Application ID for interactions
	channelID  string // Default alert channel ID
	httpClient *http.Client
	baseURL    string // Overridable for testing; defaults to baseURL constant
}

// NewDiscordClient creates a new DiscordClient with the given bot token, application ID,
// and default channel ID.
func NewDiscordClient(botToken, appID, defaultChannelID string) *DiscordClient {
	return &DiscordClient{
		botToken:  botToken,
		appID:     appID,
		channelID: defaultChannelID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
	}
}

// SetBaseURL overrides the API base URL. Intended for testing.
func (c *DiscordClient) SetBaseURL(url string) {
	c.baseURL = url
}

// DefaultChannelID returns the configured default channel ID.
func (c *DiscordClient) DefaultChannelID() string {
	return c.channelID
}

// AppID returns the configured application ID.
func (c *DiscordClient) AppID() string {
	return c.appID
}

// SendMessage sends a message to the specified Discord channel.
// POST /channels/{channelID}/messages
func (c *DiscordClient) SendMessage(ctx context.Context, channelID string, payload MessagePayload) (*Message, error) {
	url := fmt.Sprintf("%s/channels/%s/messages", c.baseURL, channelID)

	var msg Message
	if err := c.doJSON(ctx, http.MethodPost, url, payload, &msg); err != nil {
		return nil, fmt.Errorf("SendMessage: %w", err)
	}
	return &msg, nil
}

// EditMessage edits an existing message in the specified channel.
// PATCH /channels/{channelID}/messages/{messageID}
func (c *DiscordClient) EditMessage(ctx context.Context, channelID, messageID string, payload MessagePayload) error {
	url := fmt.Sprintf("%s/channels/%s/messages/%s", c.baseURL, channelID, messageID)

	if err := c.doJSON(ctx, http.MethodPatch, url, payload, nil); err != nil {
		return fmt.Errorf("EditMessage: %w", err)
	}
	return nil
}

// CreateThread creates a new thread from a message.
// POST /channels/{channelID}/messages/{messageID}/threads
func (c *DiscordClient) CreateThread(ctx context.Context, channelID, messageID, name string) (*Channel, error) {
	url := fmt.Sprintf("%s/channels/%s/messages/%s/threads", c.baseURL, channelID, messageID)

	body := struct {
		Name                string `json:"name"`
		AutoArchiveDuration int    `json:"auto_archive_duration"`
	}{
		Name:                name,
		AutoArchiveDuration: 1440, // 24 hours
	}

	var ch Channel
	if err := c.doJSON(ctx, http.MethodPost, url, body, &ch); err != nil {
		return nil, fmt.Errorf("CreateThread: %w", err)
	}
	return &ch, nil
}

// SendThreadReply sends a message in a thread. A Discord thread is a channel,
// so this is equivalent to SendMessage with the thread ID as the channel ID.
// POST /channels/{threadID}/messages
func (c *DiscordClient) SendThreadReply(ctx context.Context, threadID string, payload MessagePayload) (*Message, error) {
	return c.SendMessage(ctx, threadID, payload)
}

// RespondToInteraction responds to a Discord interaction (e.g., button click).
// POST /interactions/{interactionID}/{interactionToken}/callback
func (c *DiscordClient) RespondToInteraction(ctx context.Context, interactionID, interactionToken string, resp InteractionResponse) error {
	url := fmt.Sprintf("%s/interactions/%s/%s/callback", c.baseURL, interactionID, interactionToken)

	if err := c.doJSON(ctx, http.MethodPost, url, resp, nil); err != nil {
		return fmt.Errorf("RespondToInteraction: %w", err)
	}
	return nil
}

// doJSON performs an HTTP request with JSON body and optional JSON response decoding.
// It handles authorization, rate limiting (retry on 429), and error responses.
func (c *DiscordClient) doJSON(ctx context.Context, method, url string, body interface{}, out interface{}) error {
	const maxRetries = 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonBody))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bot "+c.botToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("execute request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if attempt < maxRetries {
				log.WithFields(log.Fields{
					"retry_after": retryAfter,
					"attempt":     attempt + 1,
					"url":         url,
				}).Warn("Discord rate limited, retrying")

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryAfter):
					continue
				}
			}
			return fmt.Errorf("rate limited after %d retries", maxRetries)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("discord API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		if out != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("exceeded max retries")
}

// parseRetryAfter parses the Retry-After header value (seconds) and returns a duration.
// Returns a 1-second default if parsing fails.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 1 * time.Second
	}
	seconds, err := strconv.ParseFloat(header, 64)
	if err != nil {
		return 1 * time.Second
	}
	return time.Duration(seconds*1000) * time.Millisecond
}
