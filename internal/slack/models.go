// SPDX-License-Identifier: BUSL-1.1

package slack

// CheckAlertInfo contains the check information used when posting alerts to Slack.
type CheckAlertInfo struct {
	UUID          string
	Name          string
	Project       string
	Group         string
	CheckType     string
	Frequency     string // e.g. "5m"
	Message       string // error message
	IsHealthy     bool
	Severity      string // "critical", "degraded", "resolved"
	Target        string // check target: URL for HTTP, host:port for TCP/DB, host for ICMP
	OriginalError string // original error message, used when resolving to preserve context
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

// SeverityEmoji returns the header emoji based on severity/health status.
// Exported for use by handlers that need to construct fallback text.
func SeverityEmoji(info CheckAlertInfo) string {
	return severityEmoji(info)
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
