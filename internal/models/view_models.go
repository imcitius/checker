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
}
