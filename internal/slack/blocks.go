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
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:*\n%s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:*\n%s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:*\n%s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Status:*\n%s", statusText(info.IsHealthy)), false, false),
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

	// e. Actions block with buttons
	silenceCheckBtn := slack.NewButtonBlockElement(
		"silence_check", info.UUID,
		slack.NewTextBlockObject(slack.PlainTextType, "Silence this check (1h)", true, false),
	)
	silenceCheckBtn.Style = slack.StyleDanger

	silenceProjectBtn := slack.NewButtonBlockElement(
		"silence_project", info.Project,
		slack.NewTextBlockObject(slack.PlainTextType, "Silence project (1h)", true, false),
	)
	silenceProjectBtn.Style = slack.StyleDanger

	ackBtn := slack.NewButtonBlockElement(
		"ack_alert", info.UUID,
		slack.NewTextBlockObject(slack.PlainTextType, "Acknowledge", true, false),
	)
	ackBtn.Style = slack.StylePrimary

	actionsBlock := slack.NewActionBlock("alert_actions",
		silenceCheckBtn,
		silenceProjectBtn,
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
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:*\n%s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:*\n%s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:*\n%s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Status:*\n%s", statusText(true)), false, false),
	}
	fieldsSection := slack.NewSectionBlock(nil, fields, nil)

	// Context with resolution time
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("✅ Resolved at %s", now), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("UUID: `%s`", info.UUID), false, false),
	)

	// No action buttons — resolved messages don't need them
	return []slack.Block{header, fieldsSection, ctx}
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
// alert message after a silence is applied. Shows a "Silenced" badge and removes action buttons.
func BuildSilencedOriginalBlocks(info CheckAlertInfo, silencedBy string) []slack.Block {
	emoji := severityEmoji(info)

	headerText := fmt.Sprintf("%s 🔇 SILENCED: %s", emoji, info.Name)
	header := slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	)

	checkTypeLabel := fmt.Sprintf("%s %s", typeEmoji(info.CheckType), info.CheckType)
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:*\n%s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:*\n%s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:*\n%s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, "*Status:*\n🔇 Silenced", false, false),
	}
	fieldsSection := slack.NewSectionBlock(nil, fields, nil)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("🔇 Silenced by <@%s> at %s", silencedBy, now), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("UUID: `%s`", info.UUID), false, false),
	)

	return []slack.Block{header, fieldsSection, ctx}
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
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project:*\n%s", info.Project), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Group:*\n%s", info.Group), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Type:*\n%s", checkTypeLabel), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Status:*\n%s", statusText(info.IsHealthy)), false, false),
	}
	fieldsSection := slack.NewSectionBlock(nil, fields, nil)

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	ctx := slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("👀 Acknowledged by <@%s> at %s", ackedBy, now), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("UUID: `%s`", info.UUID), false, false),
	)

	return []slack.Block{header, fieldsSection, ctx}
}
