package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Teams payload: %v", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send Teams alert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("teams alert failed with status %d", resp.StatusCode)
	}
	return nil
}
