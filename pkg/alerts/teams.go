// SPDX-License-Identifier: BUSL-1.1

package alerts

import (
	"encoding/json"
	"fmt"
	"time"
)

// TeamsAlerter implements the Alerter interface for Microsoft Teams.
type TeamsAlerter struct {
	WebhookURL string
}

func (a *TeamsAlerter) Type() string { return "teams" }

func (a *TeamsAlerter) SendAlert(p AlertPayload) error {
	params := TeamsAlertParams{
		CheckName:   p.CheckName,
		ProjectName: p.Project,
		Status:      "DOWN",
		Error:       p.Message,
		Time:        p.Timestamp,
	}
	return SendTeamsAlert(a.WebhookURL, params)
}

func (a *TeamsAlerter) SendRecovery(p RecoveryPayload) error {
	params := TeamsAlertParams{
		CheckName:   p.CheckName,
		ProjectName: p.Project,
		Status:      "RESOLVED",
		Error:       "",
		Time:        p.Timestamp,
	}
	return SendTeamsAlert(a.WebhookURL, params)
}

func newTeamsAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing teams config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("teams requires webhook_url")
	}
	return &TeamsAlerter{WebhookURL: cfg.WebhookURL}, nil
}

func init() {
	RegisterAlerter("teams", newTeamsAlerter)
}

// TeamsMessageCard represents a Microsoft Teams legacy MessageCard payload.
type TeamsMessageCard struct {
	Type       string                `json:"@type"`
	Context    string                `json:"@context"`
	ThemeColor string                `json:"themeColor"`
	Summary    string                `json:"summary"`
	Sections   []TeamsMessageSection `json:"sections"`
}

// TeamsMessageSection represents a section within a Teams MessageCard.
type TeamsMessageSection struct {
	ActivityTitle    string      `json:"activityTitle"`
	ActivitySubtitle string      `json:"activitySubtitle"`
	Facts            []TeamsFact `json:"facts"`
}

// TeamsFact represents a key-value fact in a Teams MessageCard section.
type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TeamsAlertParams holds the parameters needed to build a Teams alert.
type TeamsAlertParams struct {
	CheckName   string
	ProjectName string
	Status      string // "DOWN" or "RESOLVED"
	Error       string
	Time        time.Time
}

// themeColorForStatus returns the appropriate themeColor hex code.
// RED for DOWN, GREEN for RESOLVED.
func themeColorForStatus(status string) string {
	if status == "RESOLVED" {
		return "00FF00"
	}
	return "FF0000"
}

// buildTeamsPayload constructs a TeamsMessageCard from the given parameters.
func buildTeamsPayload(params TeamsAlertParams) TeamsMessageCard {
	return TeamsMessageCard{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: themeColorForStatus(params.Status),
		Summary:    fmt.Sprintf("%s is %s", params.CheckName, params.Status),
		Sections: []TeamsMessageSection{
			{
				ActivityTitle:    params.CheckName,
				ActivitySubtitle: fmt.Sprintf("Project: %s", params.ProjectName),
				Facts: []TeamsFact{
					{Name: "Status", Value: params.Status},
					{Name: "Error", Value: params.Error},
					{Name: "Time", Value: params.Time.Format(time.RFC3339)},
				},
			},
		},
	}
}

// SendTeamsAlert sends an alert to Microsoft Teams via an Incoming Webhook URL
// using the legacy MessageCard format.
func SendTeamsAlert(webhookURL string, params TeamsAlertParams) error {
	payload := buildTeamsPayload(params)
	if err := postJSON(webhookURL, payload, nil); err != nil {
		return fmt.Errorf("teams alert: %w", err)
	}
	return nil
}
