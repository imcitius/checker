// SPDX-License-Identifier: BUSL-1.1

package models

// CheckViewModel represents the data structure used by the dashboard view
type CheckViewModel struct {
	ID          string
	Name        string // maps to CheckName
	Project     string
	Healthcheck string // maps to CheckGroup
	LastResult  bool   // maps to IsHealthy
	LastExec    string // maps to LastRun
	LastPing    string // maps to LastAlertSent
	Enabled     bool   // maps to IsEnabled
	UUID        string // maps to UUID
	CheckType   string // Type of check (HTTP, TCP, etc.)
	Message     string // Error message or status message
	Host        string // Host being checked
	Periodicity string // How often the check runs
	URL         string // URL for HTTP checks
	IsSilenced  bool   // Whether alerts are silenced for this check
}
