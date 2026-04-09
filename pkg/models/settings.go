// SPDX-License-Identifier: BUSL-1.1

package models

// CheckDefaults holds checker-wide default values applied to new checks.
type CheckDefaults struct {
	RetryCount       int               `json:"retry_count"`
	RetryInterval    string            `json:"retry_interval"`
	CheckInterval    string            `json:"check_interval"`
	Timeouts         map[string]string `json:"timeouts"`
	ReAlertInterval  string            `json:"re_alert_interval"`
	Severity         string            `json:"severity"`
	AlertChannels    []string          `json:"alert_channels"`
	EscalationPolicy string            `json:"escalation_policy"`
}
