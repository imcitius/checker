package models

import "time"

// TelegramAlertThread represents a Telegram message thread associated with a check alert.
type TelegramAlertThread struct {
	ID         int
	CheckUUID  string
	ChatID     string
	MessageID  int
	IsResolved bool
	CreatedAt  time.Time
	ResolvedAt *time.Time
}
