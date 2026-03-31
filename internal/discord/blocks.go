package discord

import (
	"fmt"
	"time"
)

// BuildAlertMessage constructs a rich embed message for a check alert.
// Includes a red embed with check details and action buttons for acknowledge/silence.
func BuildAlertMessage(info CheckAlertInfo) MessagePayload {
	emoji := severityEmoji(info)
	now := time.Now().UTC().Format(time.RFC3339)

	checkTypeLabel := fmt.Sprintf("%s %s", typeEmoji(info.CheckType), info.CheckType)
	errorMsg := info.Message
	if errorMsg == "" {
		errorMsg = "No error message"
	}

	embed := Embed{
		Title: fmt.Sprintf("%s ALERT: %s", emoji, info.Name),
		Color: ColorRed,
		Description: fmt.Sprintf("```%s```", errorMsg),
		Timestamp:   now,
		Fields: []EmbedField{
			{Name: "Project", Value: info.Project, Inline: true},
			{Name: "Group", Value: info.Group, Inline: true},
			{Name: "Type", Value: checkTypeLabel, Inline: true},
			{Name: "Status", Value: statusText(info.IsHealthy), Inline: true},
		},
	}

	if info.Target != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name: "Target", Value: fmt.Sprintf("`%s`", info.Target), Inline: true,
		})
	}
	if info.Frequency != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name: "Frequency", Value: fmt.Sprintf("Every %s", info.Frequency), Inline: true,
		})
	}

	embed.Fields = append(embed.Fields, EmbedField{
		Name: "UUID", Value: fmt.Sprintf("`%s`", info.UUID), Inline: false,
	})

	buttons := ActionRow{
		Type: ComponentTypeActionRow,
		Components: []Component{
			{
				Type:     ComponentTypeButton,
				Label:    "Acknowledge",
				Style:    ButtonStylePrimary,
				CustomID: fmt.Sprintf("checker_ack_%s", info.UUID),
			},
			{
				Type:     ComponentTypeButton,
				Label:    "Silence 1h",
				Style:    ButtonStyleSecondary,
				CustomID: fmt.Sprintf("checker_silence_%s_1h", info.UUID),
			},
			{
				Type:     ComponentTypeButton,
				Label:    "Silence 24h",
				Style:    ButtonStyleSecondary,
				CustomID: fmt.Sprintf("checker_silence_%s_24h", info.UUID),
			},
		},
	}

	return MessagePayload{
		Embeds:     []Embed{embed},
		Components: []ActionRow{buttons},
	}
}

// BuildResolveMessage constructs a green embed message for a check recovery notification.
// Used as a thread reply to announce the resolution.
func BuildResolveMessage(info CheckAlertInfo) MessagePayload {
	now := time.Now().UTC().Format(time.RFC3339)

	bodyText := "Check is healthy again."
	if info.Message != "" {
		bodyText = info.Message
	}

	embed := Embed{
		Title:       fmt.Sprintf("🟢 RESOLVED: %s Recovered", info.Name),
		Description: bodyText,
		Color:       ColorGreen,
		Timestamp:   now,
		Fields: []EmbedField{
			{Name: "Status", Value: statusText(true), Inline: true},
		},
	}

	return MessagePayload{
		Embeds: []Embed{embed},
	}
}

// BuildResolvedAlertMessage constructs an updated version of the original alert message
// after resolution. Changes the embed to green, removes action buttons, and optionally
// includes the original error as context.
func BuildResolvedAlertMessage(info CheckAlertInfo, resolvedBy string) MessagePayload {
	now := time.Now().UTC().Format(time.RFC3339)

	checkTypeLabel := fmt.Sprintf("%s %s", typeEmoji(info.CheckType), info.CheckType)

	description := ""
	if info.OriginalError != "" {
		description = fmt.Sprintf("*Was:* %s", info.OriginalError)
	}

	embed := Embed{
		Title:       fmt.Sprintf("🟢 RESOLVED: %s", info.Name),
		Description: description,
		Color:       ColorGreen,
		Timestamp:   now,
		Fields: []EmbedField{
			{Name: "Project", Value: info.Project, Inline: true},
			{Name: "Group", Value: info.Group, Inline: true},
			{Name: "Type", Value: checkTypeLabel, Inline: true},
			{Name: "Status", Value: statusText(true), Inline: true},
		},
	}

	if resolvedBy != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name: "Resolved By", Value: resolvedBy, Inline: true,
		})
	}

	embed.Fields = append(embed.Fields, EmbedField{
		Name: "UUID", Value: fmt.Sprintf("`%s`", info.UUID), Inline: false,
	})

	// No action buttons on resolved messages
	return MessagePayload{
		Embeds: []Embed{embed},
	}
}
