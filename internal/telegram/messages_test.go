// SPDX-License-Identifier: BUSL-1.1

package telegram

import (
	"strings"
	"testing"
)

func TestBuildAlertHTML(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc123",
		Name:      "my-check",
		Project:   "myproject",
		Group:     "production",
		CheckType: "http",
		Frequency: "5m",
		Message:   "Connection timeout after 10s",
		Severity:  "critical",
		IsHealthy: false,
	}

	html := BuildAlertHTML(info)

	checks := []string{
		"<b>ALERT: my-check</b>",
		"<b>Project:</b> myproject",
		"<b>Group:</b> production",
		"http",
		"Unhealthy",
		"<pre>Connection timeout after 10s</pre>",
		"UUID: abc123",
		"Every 5m",
	}

	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("BuildAlertHTML missing %q\nGot: %s", want, html)
		}
	}
}

func TestBuildAlertHTML_NoMessage(t *testing.T) {
	info := CheckAlertInfo{
		Name:    "test",
		Project: "p",
		Group:   "g",
	}
	html := BuildAlertHTML(info)
	if !strings.Contains(html, "No error message") {
		t.Error("expected 'No error message' for empty Message")
	}
}

func TestBuildAlertReplyHTML(t *testing.T) {
	info := CheckAlertInfo{
		Name:      "my-check",
		Message:   "still broken",
		Severity:  "critical",
		IsHealthy: false,
	}

	html := BuildAlertReplyHTML(info)

	if !strings.Contains(html, "Still failing: my-check") {
		t.Errorf("missing 'Still failing' header in: %s", html)
	}
	if !strings.Contains(html, "<pre>still broken</pre>") {
		t.Errorf("missing error in pre block: %s", html)
	}
}

func TestBuildResolvedAlertHTML(t *testing.T) {
	info := CheckAlertInfo{
		UUID:          "abc123",
		Name:          "my-check",
		Project:       "myproject",
		Group:         "production",
		CheckType:     "http",
		IsHealthy:     true,
		OriginalError: "was broken",
	}

	html := BuildResolvedAlertHTML(info)

	if !strings.Contains(html, "RESOLVED: my-check") {
		t.Errorf("missing RESOLVED header: %s", html)
	}
	if !strings.Contains(html, "Healthy") {
		t.Errorf("missing Healthy status: %s", html)
	}
	if !strings.Contains(html, "<blockquote>Was: was broken</blockquote>") {
		t.Errorf("missing original error blockquote: %s", html)
	}
}

func TestBuildResolvedAlertHTML_NoOriginalError(t *testing.T) {
	info := CheckAlertInfo{
		Name:      "my-check",
		Project:   "p",
		Group:     "g",
		CheckType: "tcp",
		IsHealthy: true,
	}

	html := BuildResolvedAlertHTML(info)

	if strings.Contains(html, "<blockquote>") {
		t.Errorf("should not have blockquote without OriginalError: %s", html)
	}
}

func TestBuildResolveReplyHTML(t *testing.T) {
	info := CheckAlertInfo{
		Name:    "my-check",
		Message: "all good",
	}

	html := BuildResolveReplyHTML(info)

	if !strings.Contains(html, "RESOLVED: my-check Recovered") {
		t.Errorf("missing RESOLVED header: %s", html)
	}
	if !strings.Contains(html, "all good") {
		t.Errorf("missing message: %s", html)
	}
}

func TestBuildResolveReplyHTML_DefaultMessage(t *testing.T) {
	info := CheckAlertInfo{Name: "my-check"}

	html := BuildResolveReplyHTML(info)

	if !strings.Contains(html, "Check is healthy again.") {
		t.Errorf("missing default message: %s", html)
	}
}

func TestBuildErrorSnapshotHTML(t *testing.T) {
	info := CheckAlertInfo{
		Message: "connection refused",
		Target:  "https://example.com",
	}

	html := BuildErrorSnapshotHTML(info)

	if !strings.Contains(html, "<pre>connection refused</pre>") {
		t.Errorf("missing error in pre: %s", html)
	}
	if !strings.Contains(html, "Target: https://example.com") {
		t.Errorf("missing target: %s", html)
	}
}

func TestBuildErrorSnapshotHTML_NoTarget(t *testing.T) {
	info := CheckAlertInfo{Message: "err"}
	html := BuildErrorSnapshotHTML(info)
	if strings.Contains(html, "Target:") {
		t.Errorf("should not have Target without target set: %s", html)
	}
}

func TestBuildSilenceConfirmationHTML(t *testing.T) {
	html := BuildSilenceConfirmationHTML("check", "abc123", "1h", "john")

	if !strings.Contains(html, "Silence Applied") {
		t.Errorf("missing header: %s", html)
	}
	if !strings.Contains(html, "john") {
		t.Errorf("missing user: %s", html)
	}
	if !strings.Contains(html, "<code>abc123</code>") {
		t.Errorf("missing target: %s", html)
	}
	if !strings.Contains(html, "<b>1h</b>") {
		t.Errorf("missing duration: %s", html)
	}
}

func TestBuildSilenceConfirmationHTML_EmptyTarget(t *testing.T) {
	html := BuildSilenceConfirmationHTML("project", "", "4h", "jane")
	if !strings.Contains(html, "<code>all</code>") {
		t.Errorf("expected 'all' for empty target: %s", html)
	}
}

func TestBuildUnsilenceConfirmationHTML(t *testing.T) {
	html := BuildUnsilenceConfirmationHTML("check", "abc123", "john")

	if !strings.Contains(html, "Silence Removed") {
		t.Errorf("missing header: %s", html)
	}
	if !strings.Contains(html, "john") {
		t.Errorf("missing user: %s", html)
	}
}

func TestBuildAlertKeyboard(t *testing.T) {
	info := CheckAlertInfo{UUID: "abc123", Project: "myproject"}

	kb := BuildAlertKeyboard(info)

	if len(kb.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 3 {
		t.Fatalf("expected 3 buttons in row 1, got %d", len(kb.InlineKeyboard[0]))
	}
	if len(kb.InlineKeyboard[1]) != 2 {
		t.Fatalf("expected 2 buttons in row 2, got %d", len(kb.InlineKeyboard[1]))
	}

	// Check callback data values
	expectedCallbacks := [][]string{
		{"s|1h", "s|4h", "s|indef"},
		{"sp|1h", "ack"},
	}

	for i, row := range kb.InlineKeyboard {
		for j, btn := range row {
			if btn.CallbackData != expectedCallbacks[i][j] {
				t.Errorf("row %d btn %d: expected callback %q, got %q",
					i, j, expectedCallbacks[i][j], btn.CallbackData)
			}
		}
	}
}

func TestCallbackDataUnder64Bytes(t *testing.T) {
	info := CheckAlertInfo{UUID: "abc123", Project: "myproject"}
	kb := BuildAlertKeyboard(info)

	for i, row := range kb.InlineKeyboard {
		for j, btn := range row {
			if len(btn.CallbackData) > 64 {
				t.Errorf("row %d btn %d: callback_data %q is %d bytes (max 64)",
					i, j, btn.CallbackData, len(btn.CallbackData))
			}
		}
	}
}

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		info CheckAlertInfo
		want string
	}{
		{CheckAlertInfo{IsHealthy: true}, "\U0001f7e2"},
		{CheckAlertInfo{IsHealthy: false, Severity: "degraded"}, "\U0001f7e1"},
		{CheckAlertInfo{IsHealthy: false, Severity: "critical"}, "\U0001f534"},
		{CheckAlertInfo{IsHealthy: false}, "\U0001f534"},
	}

	for _, tt := range tests {
		got := severityEmoji(tt.info)
		if got != tt.want {
			t.Errorf("severityEmoji(%+v) = %q, want %q", tt.info, got, tt.want)
		}
	}
}

func TestTypeEmoji(t *testing.T) {
	tests := []struct {
		checkType string
		want      string
	}{
		{"http", "\U0001f310"},
		{"tcp", "\U0001f50c"},
		{"icmp", "\U0001f4e1"},
		{"pgsql", "\U0001f418"},
		{"postgresql", "\U0001f418"},
		{"mysql", "\U0001f42c"},
		{"passive", "\u231b"},
		{"unknown", "\U0001f50d"},
	}

	for _, tt := range tests {
		got := typeEmoji(tt.checkType)
		if got != tt.want {
			t.Errorf("typeEmoji(%q) = %q, want %q", tt.checkType, got, tt.want)
		}
	}
}

func TestStatusText(t *testing.T) {
	if got := statusText(true); !strings.Contains(got, "Healthy") {
		t.Errorf("statusText(true) = %q, want Healthy", got)
	}
	if got := statusText(false); !strings.Contains(got, "Unhealthy") {
		t.Errorf("statusText(false) = %q, want Unhealthy", got)
	}
}
