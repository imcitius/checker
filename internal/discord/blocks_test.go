package discord

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAlertMessage_Blocks_CriticalUnhealthy(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "API Health Check",
		Project:   "backend",
		Group:     "http",
		CheckType: "http",
		Frequency: "5m",
		Message:   "connection refused",
		IsHealthy: false,
		Severity:  "critical",
		Target:    "https://api.example.com",
	}

	payload := BuildAlertMessage(info)

	require.Len(t, payload.Embeds, 1)
	embed := payload.Embeds[0]

	assert.Contains(t, embed.Title, "ALERT: API Health Check")
	assert.Equal(t, ColorRed, embed.Color)
	assert.Contains(t, embed.Description, "connection refused")
	assert.NotEmpty(t, embed.Timestamp)

	// Verify fields
	fieldNames := make([]string, len(embed.Fields))
	for i, f := range embed.Fields {
		fieldNames[i] = f.Name
	}
	assert.Contains(t, fieldNames, "Project")
	assert.Contains(t, fieldNames, "Group")
	assert.Contains(t, fieldNames, "Type")
	assert.Contains(t, fieldNames, "Status")
	assert.Contains(t, fieldNames, "Target")
	assert.Contains(t, fieldNames, "Frequency")
	assert.Contains(t, fieldNames, "UUID")

	// Verify buttons
	require.Len(t, payload.Components, 1)
	row := payload.Components[0]
	assert.Equal(t, ComponentTypeActionRow, row.Type)
	require.Len(t, row.Components, 3)

	assert.Equal(t, "Acknowledge", row.Components[0].Label)
	assert.Equal(t, ButtonStylePrimary, row.Components[0].Style)
	assert.Equal(t, fmt.Sprintf("checker_ack_%s", info.UUID), row.Components[0].CustomID)

	assert.Equal(t, "Silence 1h", row.Components[1].Label)
	assert.Equal(t, "Silence 24h", row.Components[2].Label)
}

func TestBuildAlertMessage_Blocks_NoTargetNoFrequency(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "def-456",
		Name:      "Passive Check",
		Project:   "core",
		Group:     "passive",
		CheckType: "passive",
		Message:   "timed out",
		IsHealthy: false,
		Severity:  "critical",
	}

	payload := BuildAlertMessage(info)

	embed := payload.Embeds[0]
	fieldNames := make([]string, len(embed.Fields))
	for i, f := range embed.Fields {
		fieldNames[i] = f.Name
	}
	// Target and Frequency should not be present when empty.
	assert.NotContains(t, fieldNames, "Target")
	assert.NotContains(t, fieldNames, "Frequency")
}

func TestBuildAlertMessage_Blocks_EmptyErrorMessage(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "ghi-789",
		Name:      "Empty Msg Check",
		Project:   "test",
		Group:     "test",
		CheckType: "tcp",
		Message:   "",
		IsHealthy: false,
		Severity:  "critical",
	}

	payload := BuildAlertMessage(info)
	assert.Contains(t, payload.Embeds[0].Description, "No error message")
}

func TestBuildAlertMessage_Blocks_DegradedSeverity(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "deg-001",
		Name:      "Degraded Check",
		Project:   "svc",
		Group:     "api",
		CheckType: "http",
		IsHealthy: false,
		Severity:  "degraded",
		Message:   "slow response",
	}

	payload := BuildAlertMessage(info)
	// Degraded uses yellow emoji in the title.
	assert.Contains(t, payload.Embeds[0].Title, "ALERT: Degraded Check")
}

func TestBuildResolveMessage_Blocks(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "API Health Check",
		Project:   "backend",
		Group:     "http",
		CheckType: "http",
		IsHealthy: true,
		Message:   "All good now",
	}

	payload := BuildResolveMessage(info)

	require.Len(t, payload.Embeds, 1)
	embed := payload.Embeds[0]

	assert.Contains(t, embed.Title, "RESOLVED")
	assert.Contains(t, embed.Title, "API Health Check")
	assert.Equal(t, ColorGreen, embed.Color)
	assert.Equal(t, "All good now", embed.Description)
	assert.NotEmpty(t, embed.Timestamp)

	// No action buttons on resolve messages.
	assert.Empty(t, payload.Components)
}

func TestBuildResolveMessage_Blocks_EmptyMessage(t *testing.T) {
	info := CheckAlertInfo{
		Name:    "Check",
		Message: "",
	}

	payload := BuildResolveMessage(info)
	assert.Equal(t, "Check is healthy again.", payload.Embeds[0].Description)
}

func TestBuildResolvedAlertMessage_Blocks(t *testing.T) {
	info := CheckAlertInfo{
		UUID:          "abc-123",
		Name:          "API Health Check",
		Project:       "backend",
		Group:         "http",
		CheckType:     "http",
		IsHealthy:     true,
		OriginalError: "connection refused",
	}

	payload := BuildResolvedAlertMessage(info, "admin-user")

	require.Len(t, payload.Embeds, 1)
	embed := payload.Embeds[0]

	assert.Contains(t, embed.Title, "RESOLVED")
	assert.Equal(t, ColorGreen, embed.Color)
	assert.Contains(t, embed.Description, "connection refused")

	// Should have Resolved By field.
	fieldNames := make([]string, len(embed.Fields))
	for i, f := range embed.Fields {
		fieldNames[i] = f.Name
	}
	assert.Contains(t, fieldNames, "Resolved By")

	// No components on resolved messages.
	assert.Empty(t, payload.Components)
}

func TestBuildResolvedAlertMessage_Blocks_NoResolvedBy(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "Check",
		Project:   "p",
		Group:     "g",
		CheckType: "tcp",
	}

	payload := BuildResolvedAlertMessage(info, "")

	fieldNames := make([]string, len(payload.Embeds[0].Fields))
	for i, f := range payload.Embeds[0].Fields {
		fieldNames[i] = f.Name
	}
	assert.NotContains(t, fieldNames, "Resolved By")
}

func TestBuildResolvedAlertMessage_Blocks_NoOriginalError(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "Check",
		Project:   "p",
		Group:     "g",
		CheckType: "tcp",
	}

	payload := BuildResolvedAlertMessage(info, "someone")
	assert.Empty(t, payload.Embeds[0].Description)
}
