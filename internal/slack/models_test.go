package slack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckAlertInfo_Fields(t *testing.T) {
	info := CheckAlertInfo{
		UUID:          "abc-123",
		Name:          "My Check",
		Project:       "proj",
		Group:         "grp",
		CheckType:     "http",
		Frequency:     "5m",
		Message:       "timeout",
		IsHealthy:     false,
		Severity:      "critical",
		Target:        "https://example.com",
		OriginalError: "original err",
	}

	assert.Equal(t, "abc-123", info.UUID)
	assert.Equal(t, "My Check", info.Name)
	assert.Equal(t, "proj", info.Project)
	assert.Equal(t, "grp", info.Group)
	assert.Equal(t, "http", info.CheckType)
	assert.Equal(t, "5m", info.Frequency)
	assert.Equal(t, "timeout", info.Message)
	assert.False(t, info.IsHealthy)
	assert.Equal(t, "critical", info.Severity)
	assert.Equal(t, "https://example.com", info.Target)
	assert.Equal(t, "original err", info.OriginalError)
}

func TestTypeEmoji_Models(t *testing.T) {
	tests := []struct {
		checkType string
		expected  string
	}{
		{"http", "\U0001F310"},
		{"tcp", "\U0001F50C"},
		{"icmp", "\U0001F4E1"},
		{"pgsql", "\U0001F418"},
		{"postgresql", "\U0001F418"},
		{"mysql", "\U0001F42C"},
		{"passive", "\u23F3"},
		{"unknown", "\U0001F50D"},
		{"", "\U0001F50D"},
	}

	for _, tt := range tests {
		t.Run(tt.checkType, func(t *testing.T) {
			assert.Equal(t, tt.expected, typeEmoji(tt.checkType))
		})
	}
}

func TestSeverityEmoji_Models(t *testing.T) {
	// Healthy -> green
	assert.Equal(t, "\U0001F7E2", severityEmoji(CheckAlertInfo{IsHealthy: true}))
	// Unhealthy degraded -> yellow
	assert.Equal(t, "\U0001F7E1", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: "degraded"}))
	// Unhealthy critical -> red
	assert.Equal(t, "\U0001F534", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: "critical"}))
	// Unhealthy default -> red
	assert.Equal(t, "\U0001F534", severityEmoji(CheckAlertInfo{IsHealthy: false, Severity: ""}))
}

func TestSeverityEmoji_Exported(t *testing.T) {
	info := CheckAlertInfo{IsHealthy: false, Severity: "critical"}
	assert.Equal(t, severityEmoji(info), SeverityEmoji(info))
}

func TestStatusText_Models(t *testing.T) {
	healthy := statusText(true)
	assert.Contains(t, healthy, "Healthy")

	unhealthy := statusText(false)
	assert.Contains(t, unhealthy, "Unhealthy")
}
