// SPDX-License-Identifier: BUSL-1.1

package models

import "time"

// EscalationStep defines a single step in an escalation policy.
// Channel is the alert channel name (e.g. "telegram", "slack", "pagerduty").
// DelayMin is the number of minutes after the initial failure before this step fires.
type EscalationStep struct {
	Channel  string `json:"channel"`   // alert channel name
	DelayMin int    `json:"delay_min"` // minutes after initial failure
}

// EscalationPolicy defines an escalation policy with ordered steps.
type EscalationPolicy struct {
	ID        int              `json:"id"`
	Name      string           `json:"name"`
	Steps     []EscalationStep `json:"steps"`
	CreatedAt time.Time        `json:"created_at"`
}

// EscalationNotification records that a specific escalation step was notified
// for a given check incident.
type EscalationNotification struct {
	ID         int       `json:"id"`
	CheckUUID  string    `json:"check_uuid"`
	PolicyName string    `json:"policy_name"`
	StepIndex  int       `json:"step_index"`
	NotifiedAt time.Time `json:"notified_at"`
}
