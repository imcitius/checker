// SPDX-License-Identifier: BUSL-1.1

package alerts

import "time"

// AlertPayload carries details about a failed check to alert channels.
type AlertPayload struct {
	CheckName  string
	CheckUUID  string
	Project    string
	CheckGroup string
	CheckType  string
	Message    string // error message from the check
	Severity   string // "critical", "warning", "info"
	Timestamp  time.Time
}

// RecoveryPayload carries details about a recovered check to alert channels.
type RecoveryPayload struct {
	CheckName  string
	CheckUUID  string
	Project    string
	CheckGroup string
	CheckType  string
	Timestamp  time.Time
}

// Alerter is the common interface that all alert channel implementations must satisfy.
type Alerter interface {
	// SendAlert dispatches a failure notification to the alert channel.
	SendAlert(payload AlertPayload) error
	// SendRecovery dispatches a recovery notification to the alert channel.
	SendRecovery(payload RecoveryPayload) error
	// Type returns the channel type string, e.g. "telegram".
	Type() string
}
