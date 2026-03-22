package slack

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

// BuildAlertBlocks constructs Block Kit blocks for an alert message.
// Layout:
//   - Header: emoji + check name
//   - Section with fields: Project, Group, Type, Status
//   - Section: error message in code block
//   - Context: timestamp, UUID, frequency
//   - Actions: Silence check, Silence project, Acknowledge
func BuildAlertBlocks(info CheckAlertInfo) []slack.Block {
	emoji := severityEmoji(info)

	// a. Header block
	headerText := fmt.Sprintf("%s ALERT: %s", emoji, info.Name)
	if info.IsHealthy {
		headerText = fmt.Sprintf("%s RESOLVED: %s", emoji, info.Name)
	}
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	)

	// b. Section with fields: Project, Group, Type, Status
	checkTypeLabel := fmt.Sprintf("%s %s", typeEmoji(info.CheckType), info.CheckType)
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:* %s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:* %s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:* %s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Status:* %s", statusText(info.IsHealthy)), false, false),
	}
	fieldsSection := slack.NewSectionBlock(nil, fields, nil)

	// c. Section: error message in code block
	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}
	errorText := fmt.Sprintf("```%s```", errorMsg)
	errorSection := slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, errorText, false, false),
		nil, nil,
	)

	// d. Context block: timestamp, UUID, frequency
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	contextElements := []slack.MixedElement{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("⏰ %s", now), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("UUID: `%s`", info.UUID), false, false),
	}
	if info.Frequency != "" {
		contextElements = append(contextElements,
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("Every %s", info.Frequency), false, false),
		)
	}
	contextBlock := slack.NewContextBlock("", contextElements...)

	// e. Actions block with silence duration selects and acknowledge button
	silenceDurations := []struct {
		Label string
		Value string
	}{
		{"30 minutes", "30m"},
		{"1 hour", "1h"},
		{"4 hours", "4h"},
		{"8 hours", "8h"},
		{"24 hours", "24h"},
		{"Indefinite", "indefinite"},
	}

	// Silence check dropdown
	checkOptions := make([]*slack.OptionBlockObject, len(silenceDurations))
	for i, d := range silenceDurations {
		checkOptions[i] = slack.NewOptionBlockObject(
			fmt.Sprintf("%s|%s", info.UUID, d.Value),
			slack.NewTextBlockObject(slack.PlainTextType, d.Label, false, false),
			nil,
		)
	}
	silenceCheckSelect := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		slack.NewTextBlockObject(slack.PlainTextType, "🔇 Silence check...", false, false),
		"silence_check",
		checkOptions...,
	)

	// Silence project dropdown
	projectOptions := make([]*slack.OptionBlockObject, len(silenceDurations))
	for i, d := range silenceDurations {
		projectOptions[i] = slack.NewOptionBlockObject(
			fmt.Sprintf("%s|%s", info.Project, d.Value),
			slack.NewTextBlockObject(slack.PlainTextType, d.Label, false, false),
			nil,
		)
	}
	silenceProjectSelect := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		slack.NewTextBlockObject(slack.PlainTextType, "🔇 Silence project...", false, false),
		"silence_project",
		projectOptions...,
	)

	ackBtn := slack.NewButtonBlockElement(
		"ack_alert", info.UUID,
		slack.NewTextBlockObject(slack.PlainTextType, "Acknowledge", true, false),
	)
	ackBtn.Style = slack.StylePrimary

	actionsBlock := slack.NewActionBlock("alert_actions",
		silenceCheckSelect,
		silenceProjectSelect,
		ackBtn,
	)

	return []slack.Block{
		header,
		fieldsSection,
		errorSection,
		contextBlock,
		actionsBlock,
	}
}

// BuildErrorSnapshotBlocks constructs Block Kit blocks for an immutable error snapshot
// thread reply. This is posted immediately after the initial alert and is never edited.
// Layout:
//   - Section: error message in code block
//   - Context: target and timestamp
func BuildErrorSnapshotBlocks(info CheckAlertInfo) []slack.Block {
	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}
	errorText := fmt.Sprintf("```%s```", errorMsg)
	errorSection := slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, errorText, false, false),
		nil, nil,
	)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	contextElements := []slack.MixedElement{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("⏰ %s", now), false, false),
	}
	if info.Target != "" {
		contextElements = append(contextElements,
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("Target: `%s`", info.Target), false, false),
		)
	}
	contextBlock := slack.NewContextBlock("", contextElements...)

	return []slack.Block{errorSection, contextBlock}
}

// BuildResolveBlocks constructs Block Kit blocks for a resolution thread reply.
func BuildResolveBlocks(info CheckAlertInfo) []slack.Block {
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, fmt.Sprintf("🟢 RESOLVED: %s Recovered", info.Name), true, false),
	)

	bodyText := "Check is healthy again."
	if info.Message != "" {
		bodyText = info.Message
	}
	body := slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, bodyText, false, false),
		nil, nil,
	)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("✅ Resolved at %s", now), false, false),
	)

	return []slack.Block{header, body, ctx}
}

// BuildResolvedOriginalBlocks constructs Block Kit blocks for updating the original
// alert message after resolution. It shows the resolved state without action buttons.
func BuildResolvedOriginalBlocks(info CheckAlertInfo) []slack.Block {
	// Header with green emoji
	headerText := fmt.Sprintf("🟢 RESOLVED: %s", info.Name)
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	)

	// Fields section (same layout as alert, but with healthy status)
	checkTypeLabel := fmt.Sprintf("%s %s", typeEmoji(info.CheckType), info.CheckType)
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:* %s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:* %s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:* %s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Status:* %s", statusText(true)), false, false),
	}
	fieldsSection := slack.NewSectionBlock(nil, fields, nil)

	// Context with resolution time
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("✅ Resolved at %s", now), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("UUID: `%s`", info.UUID), false, false),
	)

	// Include original error message (muted) if available
	blocks := []slack.Block{header, fieldsSection}
	if info.OriginalError != "" {
		errorSection := slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("> _Was: %s_", info.OriginalError), false, false),
			nil, nil,
		)
		blocks = append(blocks, errorSection)
	}
	blocks = append(blocks, ctx)

	// No action buttons — resolved messages don't need them
	return blocks
}

// BuildUnsilenceConfirmationBlocks constructs Block Kit blocks for an un-silence confirmation reply.
func BuildUnsilenceConfirmationBlocks(scope, target, user string) []slack.Block {
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, "🔊 Silence Removed", true, false),
	)

	targetDisplay := target
	if targetDisplay == "" {
		targetDisplay = "all"
	}

	bodyText := fmt.Sprintf("<@%s> removed silence on *%s* `%s`", user, scope, targetDisplay)
	body := slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, bodyText, false, false),
		nil, nil,
	)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("Removed at %s", now), false, false),
	)

	return []slack.Block{header, body, ctx}
}

// BuildSilenceConfirmationBlocks constructs Block Kit blocks for a silence confirmation reply.
func BuildSilenceConfirmationBlocks(scope, target, duration, user string) []slack.Block {
	headerText := "🔇 Silence Applied"
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	)

	targetDisplay := target
	if targetDisplay == "" {
		targetDisplay = "all"
	}

	bodyText := fmt.Sprintf("<@%s> silenced *%s* `%s` for *%s*", user, scope, targetDisplay, duration)
	body := slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, bodyText, false, false),
		nil, nil,
	)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("Applied at %s", now), false, false),
	)

	return []slack.Block{header, body, ctx}
}

// BuildSilencedOriginalBlocks constructs Block Kit blocks for updating the original
// alert message after a silence is applied. Shows a "Silenced" badge with un-silence button.
// The silenceScope should be "check" or "project", and silenceTarget is the UUID or project name.
func BuildSilencedOriginalBlocks(info CheckAlertInfo, silencedBy, silenceScope, silenceTarget string) []slack.Block {
	emoji := severityEmoji(info)

	headerText := fmt.Sprintf("%s 🔇 SILENCED: %s", emoji, info.Name)
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	)

	checkTypeLabel := fmt.Sprintf("%s %s", typeEmoji(info.CheckType), info.CheckType)
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:* %s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:* %s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:* %s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, "*Status:* 🔇 Silenced", false, false),
	}
	fieldsSection := slack.NewSectionBlock(nil, fields, nil)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("🔇 Silenced by <@%s> at %s", silencedBy, now), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("UUID: `%s`", info.UUID), false, false),
	)

	// Un-silence button so users can remove the silence
	unsilenceBtn := slack.NewButtonBlockElement(
		"unsilence", fmt.Sprintf("%s|%s", silenceScope, silenceTarget),
		slack.NewTextBlockObject(slack.PlainTextType, "🔊 Un-silence", true, false),
	)
	unsilenceBtn.Style = slack.StylePrimary
	actionsBlock := slack.NewActionBlock("silence_actions", unsilenceBtn)

	return []slack.Block{header, fieldsSection, ctx, actionsBlock}
}

// BuildAcknowledgedOriginalBlocks constructs Block Kit blocks for updating the original
// alert message after acknowledgment. Shows an "Acknowledged" badge and removes action buttons.
func BuildAcknowledgedOriginalBlocks(info CheckAlertInfo, ackedBy string) []slack.Block {
	emoji := severityEmoji(info)

	headerText := fmt.Sprintf("%s 👀 ACKNOWLEDGED: %s", emoji, info.Name)
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	)

	checkTypeLabel := fmt.Sprintf("%s %s", typeEmoji(info.CheckType), info.CheckType)
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:* %s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:* %s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:* %s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Status:* %s", statusText(info.IsHealthy)), false, false),
	}
	fieldsSection := slack.NewSectionBlock(nil, fields, nil)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("👀 Acknowledged by <@%s> at %s", ackedBy, now), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("UUID: `%s`", info.UUID), false, false),
	)

	return []slack.Block{header, fieldsSection, ctx}
}
