package slack

// CheckAlertInfo contains the check information used when posting alerts to Slack.
type CheckAlertInfo struct {
	UUID      string
	Name      string
	Project   string
	Group     string
	CheckType string
	Frequency string // e.g. "5m"
	Message   string // error message
	IsHealthy bool
	Severity  string // "critical", "degraded", "resolved"
}

// typeEmoji returns the emoji for a check type.
func typeEmoji(checkType string) string {
	switch checkType {
	case "http":
		return "🌐"
	case "tcp":
		return "🔌"
	case "icmp":
		return "📡"
	case "pgsql", "postgresql":
		return "🐘"
	case "mysql":
		return "🐬"
	case "passive":
		return "⏳"
	default:
		return "🔍"
	}
}

// severityEmoji returns the header emoji based on severity/health status.
func severityEmoji(info CheckAlertInfo) string {
	if info.IsHealthy {
		return "🟢"
	}
	switch info.Severity {
	case "degraded":
		return "🟡"
	default:
		// critical or unspecified unhealthy
		return "🔴"
	}
}

// statusText returns a human-readable status string with emoji.
func statusText(isHealthy bool) string {
	if isHealthy {
		return "🟢 Healthy"
	}
	return "🔴 Unhealthy"
}
