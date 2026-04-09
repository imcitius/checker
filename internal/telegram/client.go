// SPDX-License-Identifier: BUSL-1.1

package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramClient provides methods to interact with the Telegram Bot API.
type TelegramClient struct {
	botToken      string
	secretToken   string // for webhook verification
	defaultChatID string
	httpClient    *http.Client
	baseURL       string // "https://api.telegram.org/bot{token}"
}

// NewTelegramClient creates a new TelegramClient with the given bot token, secret token, and default chat ID.
func NewTelegramClient(botToken, secretToken, defaultChatID string) *TelegramClient {
	return &TelegramClient{
		botToken:      botToken,
		secretToken:   secretToken,
		defaultChatID: defaultChatID,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s", botToken),
	}
}

// NewTelegramClientWithBaseURL creates a TelegramClient with a custom base URL (for testing).
func NewTelegramClientWithBaseURL(botToken, secretToken, defaultChatID, baseURL string) *TelegramClient {
	return &TelegramClient{
		botToken:      botToken,
		secretToken:   secretToken,
		defaultChatID: defaultChatID,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		baseURL:       baseURL,
	}
}

// DefaultChatID returns the configured default chat ID.
func (c *TelegramClient) DefaultChatID() string {
	return c.defaultChatID
}

// SecretToken returns the configured secret token for webhook verification.
func (c *TelegramClient) SecretToken() string {
	return c.secretToken
}

// doPost sends a JSON POST request to the Telegram Bot API and returns the result.
func (c *TelegramClient) doPost(ctx context.Context, method string, payload interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal payload: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: read response: %w", method, err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("%s: unmarshal response: %w", method, err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("%s: API error: %s", method, apiResp.Description)
	}

	return apiResp.Result, nil
}

// sendMessagePayload is the request body for sendMessage.
type sendMessagePayload struct {
	ChatID                string                `json:"chat_id"`
	Text                  string                `json:"text"`
	ParseMode             string                `json:"parse_mode,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
	ReplyToMessageID      *int                  `json:"reply_to_message_id,omitempty"`
	AllowSendingWithoutRe bool                  `json:"allow_sending_without_reply,omitempty"`
}

// SendMessage sends a message to the specified chat and returns the message ID.
func (c *TelegramClient) SendMessage(ctx context.Context, chatID, text, parseMode string, replyMarkup *InlineKeyboardMarkup, replyToMsgID *int) (int, error) {
	payload := sendMessagePayload{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	}
	if replyMarkup != nil {
		payload.ReplyMarkup = replyMarkup
	}
	if replyToMsgID != nil {
		payload.ReplyToMessageID = replyToMsgID
		payload.AllowSendingWithoutRe = true
	}

	result, err := c.doPost(ctx, "sendMessage", payload)
	if err != nil {
		return 0, err
	}

	var msg TelegramMessage
	if err := json.Unmarshal(result, &msg); err != nil {
		return 0, fmt.Errorf("sendMessage: unmarshal result: %w", err)
	}

	return msg.MessageID, nil
}

// editMessageTextPayload is the request body for editMessageText.
type editMessageTextPayload struct {
	ChatID      string                `json:"chat_id"`
	MessageID   int                   `json:"message_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// EditMessageText edits the text of an existing message.
func (c *TelegramClient) EditMessageText(ctx context.Context, chatID string, messageID int, text, parseMode string, replyMarkup *InlineKeyboardMarkup) error {
	payload := editMessageTextPayload{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: parseMode,
	}
	if replyMarkup != nil {
		payload.ReplyMarkup = replyMarkup
	}

	_, err := c.doPost(ctx, "editMessageText", payload)
	return err
}

// answerCallbackQueryPayload is the request body for answerCallbackQuery.
type answerCallbackQueryPayload struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

// AnswerCallbackQuery answers a callback query from an inline keyboard button.
func (c *TelegramClient) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string, showAlert bool) error {
	payload := answerCallbackQueryPayload{
		CallbackQueryID: callbackQueryID,
		Text:            text,
		ShowAlert:       showAlert,
	}

	_, err := c.doPost(ctx, "answerCallbackQuery", payload)
	return err
}

// setWebhookPayload is the request body for setWebhook.
type setWebhookPayload struct {
	URL         string `json:"url"`
	SecretToken string `json:"secret_token,omitempty"`
}

// SetWebhook sets the webhook URL for receiving updates.
func (c *TelegramClient) SetWebhook(ctx context.Context, url, secretToken string) error {
	payload := setWebhookPayload{
		URL:         url,
		SecretToken: secretToken,
	}

	_, err := c.doPost(ctx, "setWebhook", payload)
	return err
}

// --- High-level methods (mirror slack/client.go) ---

// PostAlert builds an HTML alert with keyboard and sends it to the specified chat.
// Returns the message ID.
func (c *TelegramClient) PostAlert(ctx context.Context, chatID string, info CheckAlertInfo) (int, error) {
	html := BuildAlertHTML(info)
	keyboard := BuildAlertKeyboard(info)
	return c.SendMessage(ctx, chatID, html, "HTML", keyboard, nil)
}

// PostAlertReply posts an alert reply linked to the original message via reply_to_message_id.
// Returns the reply message ID.
func (c *TelegramClient) PostAlertReply(ctx context.Context, chatID string, replyToMsgID int, info CheckAlertInfo) (int, error) {
	html := BuildAlertReplyHTML(info)
	return c.SendMessage(ctx, chatID, html, "HTML", nil, &replyToMsgID)
}

// PostErrorSnapshotReply posts an immutable error snapshot as a reply (no keyboard).
// Returns the reply message ID.
func (c *TelegramClient) PostErrorSnapshotReply(ctx context.Context, chatID string, replyToMsgID int, info CheckAlertInfo) (int, error) {
	html := BuildErrorSnapshotHTML(info)
	return c.SendMessage(ctx, chatID, html, "HTML", nil, &replyToMsgID)
}

// SendResolve edits the original alert to resolved state and posts a recovery reply.
func (c *TelegramClient) SendResolve(ctx context.Context, info CheckAlertInfo, originalMsgID int, chatID string) error {
	// 1. Edit original message to resolved state (no buttons)
	resolvedHTML := BuildResolvedAlertHTML(info)
	if err := c.EditMessageText(ctx, chatID, originalMsgID, resolvedHTML, "HTML", nil); err != nil {
		return fmt.Errorf("SendResolve edit original: %w", err)
	}

	// 2. Post recovery reply
	replyHTML := BuildResolveReplyHTML(info)
	_, err := c.SendMessage(ctx, chatID, replyHTML, "HTML", nil, &originalMsgID)
	if err != nil {
		return fmt.Errorf("SendResolve post reply: %w", err)
	}

	return nil
}

// SendSilenceConfirmation posts a silence confirmation as a reply.
func (c *TelegramClient) SendSilenceConfirmation(ctx context.Context, chatID string, replyToMsgID int, scope, target, duration, user string) error {
	html := BuildSilenceConfirmationHTML(scope, target, duration, user)
	_, err := c.SendMessage(ctx, chatID, html, "HTML", nil, &replyToMsgID)
	if err != nil {
		return fmt.Errorf("SendSilenceConfirmation: %w", err)
	}
	return nil
}

// SendUnsilenceConfirmation posts an unsilence confirmation as a reply.
func (c *TelegramClient) SendUnsilenceConfirmation(ctx context.Context, chatID string, replyToMsgID int, scope, target, user string) error {
	html := BuildUnsilenceConfirmationHTML(scope, target, user)
	_, err := c.SendMessage(ctx, chatID, html, "HTML", nil, &replyToMsgID)
	if err != nil {
		return fmt.Errorf("SendUnsilenceConfirmation: %w", err)
	}
	return nil
}
