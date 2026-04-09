// SPDX-License-Identifier: BUSL-1.1

package models

import "time"

// DiscordAlertThread represents a Discord thread associated with a check alert.
type DiscordAlertThread struct {
	ID         int
	CheckUUID  string
	ChannelID  string
	MessageID  string
	ThreadID   string
	IsResolved bool
	CreatedAt  time.Time
	ResolvedAt *time.Time
}
