package slack

import (
	"testing"

	"github.com/slack-go/slack"
)

func TestBuildAlertBlocks_CriticalUnhealthy(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "API Health Check",
		Project:   "backend",
		Group:     "http",
		CheckType: "http",
		Frequency: "5m",
		Message:   "connection refused to https://api.example.com",
		IsHealthy: false,
		Severity:  "critical",
	}

	blocks := BuildAlertBlocks(info)

	// Should have 5 blocks: header, fields, error, context, actions
	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(blocks))
	}

	// Verify header block
	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Fatal("block 0 should be a HeaderBlock")
	}
	if header.Text.Text != "🔴 ALERT: API Health Check" {
		t.Errorf("header text = %q, want %q", header.Text.Text, "🔴 ALERT: API Health Check")
	}

	// Verify fields section
	section, ok := blocks[1].(*slack.SectionBlock)
	if !ok {
		t.Fatal("block 1 should be a SectionBlock")
	}
	if len(section.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(section.Fields))
	}
	// Check Project field
	assertFieldContains(t, section.Fields[0], "backend")
	// Check Group field
	assertFieldContains(t, section.Fields[1], "http")
	// Check Type field with emoji
	assertFieldContains(t, section.Fields[2], "🌐")
	assertFieldContains(t, section.Fields[2], "http")
	// Check Status field
	assertFieldContains(t, section.Fields[3], "🔴 Unhealthy")

	// Verify error section
	errorSection, ok := blocks[2].(*slack.SectionBlock)
	if !ok {
		t.Fatal("block 2 should be a SectionBlock")
	}
	if errorSection.Text == nil {
		t.Fatal("error section should have text")
	}
	assertContains(t, errorSection.Text.Text, "connection refused")

	// Verify context block
	ctx, ok := blocks[3].(*slack.ContextBlock)
	if !ok {
		t.Fatal("block 3 should be a ContextBlock")
	}
	if len(ctx.ContextElements.Elements) < 2 {
		t.Fatalf("expected at least 2 context elements, got %d", len(ctx.ContextElements.Elements))
	}

	// Verify actions block
	actions, ok := blocks[4].(*slack.ActionBlock)
	if !ok {
		t.Fatal("block 4 should be an ActionBlock")
	}
	if len(actions.Elements.ElementSet) != 3 {
		t.Fatalf("expected 3 action elements, got %d", len(actions.Elements.ElementSet))
	}

	// Verify action elements: two static selects + one button
	sel0 := actions.Elements.ElementSet[0].(*slack.SelectBlockElement)
	if sel0.ActionID != "silence_check" {
		t.Errorf("select 0 action_id = %q, want %q", sel0.ActionID, "silence_check")
	}
	// Verify it has 6 duration options
	if len(sel0.Options) != 6 {
		t.Errorf("silence_check select has %d options, want 6", len(sel0.Options))
	}
	// First option value should encode the check UUID with 30m duration
	if sel0.Options[0].Value != "abc-123|30m" {
		t.Errorf("silence_check first option value = %q, want %q", sel0.Options[0].Value, "abc-123|30m")
	}

	sel1 := actions.Elements.ElementSet[1].(*slack.SelectBlockElement)
	if sel1.ActionID != "silence_project" {
		t.Errorf("select 1 action_id = %q, want %q", sel1.ActionID, "silence_project")
	}
	// First option value should encode the project name with 30m duration
	if sel1.Options[0].Value != "backend|30m" {
		t.Errorf("silence_project first option value = %q, want %q", sel1.Options[0].Value, "backend|30m")
	}

	btn2 := actions.Elements.ElementSet[2].(*slack.ButtonBlockElement)
	if btn2.ActionID != "ack_alert" {
		t.Errorf("button 2 action_id = %q, want %q", btn2.ActionID, "ack_alert")
	}
}

func TestBuildAlertBlocks_DegradedSeverity(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "def-456",
		Name:      "DB Replication Lag",
		Project:   "database",
		Group:     "postgres",
		CheckType: "pgsql",
		Message:   "replication lag exceeds threshold",
		IsHealthy: false,
		Severity:  "degraded",
	}

	blocks := BuildAlertBlocks(info)
	header := blocks[0].(*slack.HeaderBlock)
	if header.Text.Text != "🟡 ALERT: DB Replication Lag" {
		t.Errorf("degraded header = %q, want yellow emoji", header.Text.Text)
	}

	// Verify PostgreSQL type emoji
	section := blocks[1].(*slack.SectionBlock)
	assertFieldContains(t, section.Fields[2], "🐘")
}

func TestBuildAlertBlocks_Healthy(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "ghi-789",
		Name:      "Frontend Check",
		Project:   "frontend",
		Group:     "http",
		CheckType: "http",
		IsHealthy: true,
		Severity:  "resolved",
	}

	blocks := BuildAlertBlocks(info)
	header := blocks[0].(*slack.HeaderBlock)
	if header.Text.Text != "🟢 RESOLVED: Frontend Check" {
		t.Errorf("healthy header = %q, want green resolved", header.Text.Text)
	}

	section := blocks[1].(*slack.SectionBlock)
	assertFieldContains(t, section.Fields[3], "🟢 Healthy")
}

func TestBuildAlertBlocks_NoFrequency(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "jkl-012",
		Name:      "Test Check",
		Project:   "test",
		Group:     "test",
		CheckType: "tcp",
		Message:   "connection timeout",
		IsHealthy: false,
	}

	blocks := BuildAlertBlocks(info)
	ctx := blocks[3].(*slack.ContextBlock)
	// Without frequency, should have 2 context elements (timestamp + UUID)
	if len(ctx.ContextElements.Elements) != 2 {
		t.Errorf("expected 2 context elements without frequency, got %d", len(ctx.ContextElements.Elements))
	}
}

func TestBuildAlertBlocks_WithFrequency(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "mno-345",
		Name:      "Test Check",
		Project:   "test",
		Group:     "test",
		CheckType: "tcp",
		Frequency: "30s",
		Message:   "connection timeout",
		IsHealthy: false,
	}

	blocks := BuildAlertBlocks(info)
	ctx := blocks[3].(*slack.ContextBlock)
	// With frequency, should have 3 context elements
	if len(ctx.ContextElements.Elements) != 3 {
		t.Errorf("expected 3 context elements with frequency, got %d", len(ctx.ContextElements.Elements))
	}
}

func TestBuildResolveBlocks(t *testing.T) {
	info := CheckAlertInfo{
		UUID:    "abc-123",
		Name:    "API Health Check",
		Message: "Check is healthy again. Downtime: ~10m",
	}

	blocks := BuildResolveBlocks(info)

	// Should have 3 blocks: header, body, context
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}

	header := blocks[0].(*slack.HeaderBlock)
	assertContains(t, header.Text.Text, "🟢")
	assertContains(t, header.Text.Text, "RESOLVED")
	assertContains(t, header.Text.Text, "API Health Check")

	body := blocks[1].(*slack.SectionBlock)
	assertContains(t, body.Text.Text, "Check is healthy again. Downtime: ~10m")
}

func TestBuildResolveBlocks_DefaultMessage(t *testing.T) {
	info := CheckAlertInfo{
		UUID: "abc-123",
		Name: "Test Check",
	}

	blocks := BuildResolveBlocks(info)
	body := blocks[1].(*slack.SectionBlock)
	if body.Text.Text != "Check is healthy again." {
		t.Errorf("default resolve body = %q, want %q", body.Text.Text, "Check is healthy again.")
	}
}

func TestBuildResolvedOriginalBlocks(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "API Health Check",
		Project:   "backend",
		Group:     "http",
		CheckType: "http",
		IsHealthy: true,
		Severity:  "resolved",
	}

	blocks := BuildResolvedOriginalBlocks(info)

	// Should have 3 blocks: header, fields, context (no actions)
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks (no actions), got %d", len(blocks))
	}

	header := blocks[0].(*slack.HeaderBlock)
	assertContains(t, header.Text.Text, "🟢")
	assertContains(t, header.Text.Text, "RESOLVED")

	// Verify no action block
	for _, b := range blocks {
		if _, ok := b.(*slack.ActionBlock); ok {
			t.Error("resolved original message should not have action blocks")
		}
	}

	// Verify status shows healthy
	section := blocks[1].(*slack.SectionBlock)
	assertFieldContains(t, section.Fields[3], "🟢 Healthy")
}

func TestBuildResolvedOriginalBlocks_WithOriginalError(t *testing.T) {
	info := CheckAlertInfo{
		UUID:          "abc-123",
		Name:          "API Health Check",
		Project:       "backend",
		Group:         "http",
		CheckType:     "http",
		IsHealthy:     true,
		Severity:      "resolved",
		OriginalError: "connection refused",
	}

	blocks := BuildResolvedOriginalBlocks(info)

	// Should have 4 blocks: header, fields, error (muted), context
	if len(blocks) != 4 {
		t.Fatalf("expected 4 blocks with original error, got %d", len(blocks))
	}

	// Verify error block is present and muted
	errorSection := blocks[2].(*slack.SectionBlock)
	assertContains(t, errorSection.Text.Text, "Was: connection refused")
	assertContains(t, errorSection.Text.Text, ">") // quote block
}

func TestBuildErrorSnapshotBlocks(t *testing.T) {
	info := CheckAlertInfo{
		Message: "connection refused",
		Target:  "https://example.com/health",
	}

	blocks := BuildErrorSnapshotBlocks(info)

	// Should have 2 blocks: error section and context
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	errorSection := blocks[0].(*slack.SectionBlock)
	assertContains(t, errorSection.Text.Text, "connection refused")
}

func TestBuildErrorSnapshotBlocks_NoTarget(t *testing.T) {
	info := CheckAlertInfo{
		Message: "timeout",
	}

	blocks := BuildErrorSnapshotBlocks(info)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	// Context block should only have timestamp, no target
	ctx := blocks[1].(*slack.ContextBlock)
	if len(ctx.ContextElements.Elements) != 1 {
		t.Errorf("expected 1 context element (timestamp only), got %d", len(ctx.ContextElements.Elements))
	}
}

func TestBuildSilenceConfirmationBlocks(t *testing.T) {
	blocks := BuildSilenceConfirmationBlocks("check", "abc-123", "1h", "U12345")

	// Should have 3 blocks: header, body, context
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}

	header := blocks[0].(*slack.HeaderBlock)
	assertContains(t, header.Text.Text, "🔇")
	assertContains(t, header.Text.Text, "Silence")

	body := blocks[1].(*slack.SectionBlock)
	assertContains(t, body.Text.Text, "U12345")
	assertContains(t, body.Text.Text, "check")
	assertContains(t, body.Text.Text, "abc-123")
	assertContains(t, body.Text.Text, "1h")
}

func TestBuildSilenceConfirmationBlocks_EmptyTarget(t *testing.T) {
	blocks := BuildSilenceConfirmationBlocks("all", "", "30m", "U67890")

	body := blocks[1].(*slack.SectionBlock)
	assertContains(t, body.Text.Text, "all")
}

func TestBuildSilencedOriginalBlocks_HasUnsilenceButton(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "API Health Check",
		Project:   "backend",
		Group:     "http",
		CheckType: "http",
	}

	blocks := BuildSilencedOriginalBlocks(info, "U12345", "check", "abc-123")

	// Should have 4 blocks: header, fields, context, actions (with un-silence button)
	if len(blocks) != 4 {
		t.Fatalf("expected 4 blocks, got %d", len(blocks))
	}

	header := blocks[0].(*slack.HeaderBlock)
	assertContains(t, header.Text.Text, "🔇")
	assertContains(t, header.Text.Text, "SILENCED")

	// Verify actions block has un-silence button
	actions := blocks[3].(*slack.ActionBlock)
	if len(actions.Elements.ElementSet) != 1 {
		t.Fatalf("expected 1 action element (unsilence button), got %d", len(actions.Elements.ElementSet))
	}
	unsilenceBtn := actions.Elements.ElementSet[0].(*slack.ButtonBlockElement)
	if unsilenceBtn.ActionID != "unsilence" {
		t.Errorf("unsilence button action_id = %q, want %q", unsilenceBtn.ActionID, "unsilence")
	}
	if unsilenceBtn.Value != "check|abc-123" {
		t.Errorf("unsilence button value = %q, want %q", unsilenceBtn.Value, "check|abc-123")
	}
}

func TestBuildUnsilenceConfirmationBlocks(t *testing.T) {
	blocks := BuildUnsilenceConfirmationBlocks("check", "abc-123", "U12345")

	// Should have 3 blocks: header, body, context
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}

	header := blocks[0].(*slack.HeaderBlock)
	assertContains(t, header.Text.Text, "🔊")
	assertContains(t, header.Text.Text, "Silence Removed")

	body := blocks[1].(*slack.SectionBlock)
	assertContains(t, body.Text.Text, "U12345")
	assertContains(t, body.Text.Text, "check")
	assertContains(t, body.Text.Text, "abc-123")
}

func TestBuildAlertBlocks_SilenceDurationOptions(t *testing.T) {
	info := CheckAlertInfo{
		UUID:      "abc-123",
		Name:      "Test Check",
		Project:   "myproject",
		Group:     "http",
		CheckType: "http",
		IsHealthy: false,
	}

	blocks := BuildAlertBlocks(info)
	actions := blocks[4].(*slack.ActionBlock)

	// Check silence_check select has all 6 duration options
	sel := actions.Elements.ElementSet[0].(*slack.SelectBlockElement)
	expectedValues := []string{
		"abc-123|30m",
		"abc-123|1h",
		"abc-123|4h",
		"abc-123|8h",
		"abc-123|24h",
		"abc-123|indefinite",
	}
	for i, expected := range expectedValues {
		if sel.Options[i].Value != expected {
			t.Errorf("silence_check option[%d] value = %q, want %q", i, sel.Options[i].Value, expected)
		}
	}
}

func TestTypeEmoji(t *testing.T) {
	tests := []struct {
		checkType string
		want      string
	}{
		{"http", "🌐"},
		{"tcp", "🔌"},
		{"icmp", "📡"},
		{"pgsql", "🐘"},
		{"postgresql", "🐘"},
		{"mysql", "🐬"},
		{"passive", "⏳"},
		{"unknown", "🔍"},
		{"", "🔍"},
	}

	for _, tt := range tests {
		got := typeEmoji(tt.checkType)
		if got != tt.want {
			t.Errorf("typeEmoji(%q) = %q, want %q", tt.checkType, got, tt.want)
		}
	}
}

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		info CheckAlertInfo
		want string
	}{
		{CheckAlertInfo{IsHealthy: true}, "🟢"},
		{CheckAlertInfo{IsHealthy: false, Severity: "critical"}, "🔴"},
		{CheckAlertInfo{IsHealthy: false, Severity: "degraded"}, "🟡"},
		{CheckAlertInfo{IsHealthy: false, Severity: ""}, "🔴"},
		{CheckAlertInfo{IsHealthy: false}, "🔴"},
	}

	for _, tt := range tests {
		got := severityEmoji(tt.info)
		if got != tt.want {
			t.Errorf("severityEmoji(%+v) = %q, want %q", tt.info, got, tt.want)
		}
	}
}

func TestStatusText(t *testing.T) {
	if got := statusText(true); got != "🟢 Healthy" {
		t.Errorf("statusText(true) = %q, want %q", got, "🟢 Healthy")
	}
	if got := statusText(false); got != "🔴 Unhealthy" {
		t.Errorf("statusText(false) = %q, want %q", got, "🔴 Unhealthy")
	}
}

// Helper functions

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if len(s) == 0 || !contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

func assertFieldContains(t *testing.T, field *slack.TextBlockObject, substr string) {
	t.Helper()
	if field == nil {
		t.Errorf("field is nil, expected to contain %q", substr)
		return
	}
	if !contains(field.Text, substr) {
		t.Errorf("field text %q does not contain %q", field.Text, substr)
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
