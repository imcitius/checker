// SPDX-License-Identifier: BUSL-1.1

package telegram

import "encoding/json"

// CheckAlertInfo holds all data needed to build alert messages.
type CheckAlertInfo struct {
	UUID, Name, Project, Group, CheckType, Frequency string
	Message, Severity, Target, OriginalError         string
	IsHealthy                                        bool
}

// APIResponse represents a Telegram Bot API response.
type APIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
}

// TelegramMessage represents a sent message.
type TelegramMessage struct {
	MessageID int  `json:"message_id"`
	Chat      Chat `json:"chat"`
}

// InlineKeyboardMarkup represents an inline keyboard for a message.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button in an inline keyboard.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// Update represents a Telegram webhook update.
type Update struct {
	UpdateID      int              `json:"update_id"`
	Message       *IncomingMessage `json:"message,omitempty"`
	CallbackQuery *CallbackQuery   `json:"callback_query,omitempty"`
}

// CallbackQuery represents an incoming callback query from a callback button.
type CallbackQuery struct {
	ID      string           `json:"id"`
	From    User             `json:"from"`
	Message *IncomingMessage `json:"message,omitempty"`
	Data    string           `json:"data"`
}

// IncomingMessage represents a received message.
type IncomingMessage struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID int64 `json:"id"`
}

// User represents a Telegram user.
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}
