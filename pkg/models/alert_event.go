package models

import "time"

// AlertEvent represents a single alert incident recorded in the alert_history table.
type AlertEvent struct {
	ID         int
	CheckUUID  string
	CheckName  string
	Project    string
	GroupName  string
	CheckType  string
	Message    string
	AlertType  string
	Region     string     `json:"region,omitempty"`
	CreatedAt  time.Time
	ResolvedAt *time.Time
	IsResolved bool
}

// AlertHistoryFilters defines optional filters for querying alert history.
type AlertHistoryFilters struct {
	Project    string
	CheckUUID  string
	IsResolved *bool
}
