package models

import "time"

// SlackAlertThread represents a Slack message thread associated with a check alert.
type SlackAlertThread struct {
	ID         int
	CheckUUID  string
	ChannelID  string
	ThreadTs   string
	ParentTs   string
	IsResolved bool
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

// AlertSilence represents a silence rule that suppresses alerts for a given scope.
type AlertSilence struct {
	ID         int
	Scope      string     // "check" or "project"
	Target     string     // check UUID or project name
	Channel    string     // alert channel name; empty = all channels
	SilencedBy string    // Slack user ID
	SilencedAt time.Time
	ExpiresAt  *time.Time
	Reason     string
	Active     bool
}
