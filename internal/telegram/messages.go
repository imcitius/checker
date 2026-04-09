// SPDX-License-Identifier: BUSL-1.1

package telegram

import (
	"fmt"
	"time"
)

// severityEmoji returns the emoji for a severity level.
func severityEmoji(info CheckAlertInfo) string {
	if info.IsHealthy {
		return "\U0001f7e2" // 🟢
	}
	switch info.Severity {
	case "degraded":
		return "\U0001f7e1" // 🟡
	default:
		return "\U0001f534" // 🔴
	}
}

// typeEmoji returns the emoji for a check type.
func typeEmoji(checkType string) string {
	switch checkType {
	case "http":
		return "\U0001f310" // 🌐
	case "tcp":
		return "\U0001f50c" // 🔌
	case "icmp":
		return "\U0001f4e1" // 📡
	case "pgsql", "postgresql":
		return "\U0001f418" // 🐘
	case "mysql":
		return "\U0001f42c" // 🐬
	case "passive":
		return "\u231b" // ⏳
	default:
		return "\U0001f50d" // 🔍
	}
}

// statusText returns a human-readable status string with emoji.
func statusText(isHealthy bool) string {
	if isHealthy {
		return "\u2705 Healthy"
	}
	return "\u274c Unhealthy"
}

// BuildAlertHTML builds a rich HTML alert message.
func BuildAlertHTML(info CheckAlertInfo) string {
	emoji := severityEmoji(info)
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}

	html := fmt.Sprintf(
		"%s <b>ALERT: %s</b>\n\n"+
			"<b>Project:</b> %s\n"+
			"<b>Group:</b> %s\n"+
			"<b>Type:</b> %s %s\n"+
			"<b>Status:</b> %s\n\n"+
			"<pre>%s</pre>\n\n"+
			"<i>\u23f0 %s | UUID: %s",
		emoji, info.Name,
		info.Project,
		info.Group,
		typeEmoji(info.CheckType), info.CheckType,
		statusText(info.IsHealthy),
		errorMsg,
		now, info.UUID,
	)

	if info.Frequency != "" {
		html += fmt.Sprintf(" | Every %s", info.Frequency)
	}
	html += "</i>"

	return html
}

// BuildAlertReplyHTML builds an HTML message for ongoing failure replies.
func BuildAlertReplyHTML(info CheckAlertInfo) string {
	emoji := severityEmoji(info)
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}

	return fmt.Sprintf(
		"%s <b>Still failing: %s</b>\n\n"+
			"<pre>%s</pre>\n\n"+
			"<i>\u23f0 %s</i>",
		emoji, info.Name,
		errorMsg,
		now,
	)
}

// BuildResolvedAlertHTML builds HTML for editing the original alert on recovery.
// Shows green header, original error in blockquote, no buttons.
func BuildResolvedAlertHTML(info CheckAlertInfo) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	html := fmt.Sprintf(
		"\U0001f7e2 <b>RESOLVED: %s</b>\n\n"+
			"<b>Project:</b> %s\n"+
			"<b>Group:</b> %s\n"+
			"<b>Type:</b> %s %s\n"+
			"<b>Status:</b> \u2705 Healthy\n",
		info.Name,
		info.Project,
		info.Group,
		typeEmoji(info.CheckType), info.CheckType,
	)

	if info.OriginalError != "" {
		html += fmt.Sprintf("\n<blockquote>Was: %s</blockquote>\n", info.OriginalError)
	}

	html += fmt.Sprintf("\n<i>\u2705 Resolved at %s | UUID: %s</i>", now, info.UUID)

	return html
}

// BuildResolveReplyHTML builds a short "Resolved" reply message.
func BuildResolveReplyHTML(info CheckAlertInfo) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	bodyText := "Check is healthy again."
	if info.Message != "" {
		bodyText = info.Message
	}

	return fmt.Sprintf(
		"\U0001f7e2 <b>RESOLVED: %s Recovered</b>\n\n"+
			"%s\n\n"+
			"<i>\u2705 Resolved at %s</i>",
		info.Name,
		bodyText,
		now,
	)
}

// BuildErrorSnapshotHTML builds an immutable error snapshot with target details.
func BuildErrorSnapshotHTML(info CheckAlertInfo) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}

	html := fmt.Sprintf("<pre>%s</pre>\n\n<i>\u23f0 %s", errorMsg, now)

	if info.Target != "" {
		html += fmt.Sprintf(" | Target: %s", info.Target)
	}
	html += "</i>"

	return html
}

// BuildSilenceConfirmationHTML builds a silence confirmation reply.
func BuildSilenceConfirmationHTML(scope, target, duration, user string) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	targetDisplay := target
	if targetDisplay == "" {
		targetDisplay = "all"
	}

	return fmt.Sprintf(
		"\U0001f507 <b>Silence Applied</b>\n\n"+
			"%s silenced <b>%s</b> <code>%s</code> for <b>%s</b>\n\n"+
			"<i>Applied at %s</i>",
		user, scope, targetDisplay, duration,
		now,
	)
}

// BuildUnsilenceConfirmationHTML builds an unsilence confirmation reply.
func BuildUnsilenceConfirmationHTML(scope, target, user string) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	targetDisplay := target
	if targetDisplay == "" {
		targetDisplay = "all"
	}

	return fmt.Sprintf(
		"\U0001f50a <b>Silence Removed</b>\n\n"+
			"%s removed silence on <b>%s</b> <code>%s</code>\n\n"+
			"<i>Removed at %s</i>",
		user, scope, targetDisplay,
		now,
	)
}

// BuildSilencedAlertHTML builds an alert message with a "🔇 SILENCED" badge and no action buttons.
func BuildSilencedAlertHTML(info CheckAlertInfo) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}

	html := fmt.Sprintf(
		"\U0001f507 <b>SILENCED: %s</b>\n\n"+
			"<b>Project:</b> %s\n"+
			"<b>Group:</b> %s\n"+
			"<b>Type:</b> %s %s\n"+
			"<b>Status:</b> %s\n\n"+
			"<pre>%s</pre>\n\n"+
			"<i>\u23f0 %s | UUID: %s",
		info.Name,
		info.Project,
		info.Group,
		typeEmoji(info.CheckType), info.CheckType,
		statusText(info.IsHealthy),
		errorMsg,
		now, info.UUID,
	)

	if info.Frequency != "" {
		html += fmt.Sprintf(" | Every %s", info.Frequency)
	}
	html += "</i>"

	return html
}

// BuildAcknowledgedAlertHTML builds an alert message with a "👀 ACKNOWLEDGED" badge.
func BuildAcknowledgedAlertHTML(info CheckAlertInfo) string {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}

	html := fmt.Sprintf(
		"\U0001f440 <b>ACKNOWLEDGED: %s</b>\n\n"+
			"<b>Project:</b> %s\n"+
			"<b>Group:</b> %s\n"+
			"<b>Type:</b> %s %s\n"+
			"<b>Status:</b> %s\n\n"+
			"<pre>%s</pre>\n\n"+
			"<i>\u23f0 %s | UUID: %s",
		info.Name,
		info.Project,
		info.Group,
		typeEmoji(info.CheckType), info.CheckType,
		statusText(info.IsHealthy),
		errorMsg,
		now, info.UUID,
	)

	if info.Frequency != "" {
		html += fmt.Sprintf(" | Every %s", info.Frequency)
	}
	html += "</i>"

	return html
}

// BuildAcknowledgedAlertKeyboard builds a keyboard for acknowledged alerts (silence buttons, no ack).
func BuildAcknowledgedAlertKeyboard(info CheckAlertInfo) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "\U0001f507 Silence 1h", CallbackData: "s|1h"},
				{Text: "\U0001f507 Silence 4h", CallbackData: "s|4h"},
				{Text: "\U0001f507 Indefinite", CallbackData: "s|indef"},
			},
			{
				{Text: "\U0001f507 Silence project 1h", CallbackData: "sp|1h"},
			},
		},
	}
}

// BuildAlertKeyboard builds an inline keyboard with silence and acknowledge buttons.
func BuildAlertKeyboard(info CheckAlertInfo) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "\U0001f507 Silence 1h", CallbackData: "s|1h"},
				{Text: "\U0001f507 Silence 4h", CallbackData: "s|4h"},
				{Text: "\U0001f507 Indefinite", CallbackData: "s|indef"},
			},
			{
				{Text: "\U0001f507 Silence project 1h", CallbackData: "sp|1h"},
				{Text: "\U0001f440 Acknowledge", CallbackData: "ack"},
			},
		},
	}
}
