package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpsgenieClient handles sending alerts to Opsgenie.
type OpsgenieClient struct {
	APIKey  string
	Region  string // "us" or "eu"
	HTTPDo  func(*http.Request) (*http.Response, error)
}

// opsgenieAlertPayload is the JSON body for creating an alert.
type opsgenieAlertPayload struct {
	Message     string `json:"message"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

// opsgenieClosePayload is the JSON body for closing an alert.
type opsgenieClosePayload struct {
	Note string `json:"note"`
}

// baseURL returns the Opsgenie API base URL for the configured region.
func (c *OpsgenieClient) baseURL() string {
	if c.Region == "eu" {
		return "https://api.eu.opsgenie.com"
	}
	return "https://api.opsgenie.com"
}

// httpDo returns the HTTP client function to use.
func (c *OpsgenieClient) httpDo() func(*http.Request) (*http.Response, error) {
	if c.HTTPDo != nil {
		return c.HTTPDo
	}
	return http.DefaultClient.Do
}

// MapPriority converts a checker severity level to an Opsgenie priority.
// critical → P1, warning → P2, info → P3, default → P3.
func MapPriority(severity string) string {
	switch severity {
	case "critical":
		return "P1"
	case "warning":
		return "P2"
	case "info":
		return "P3"
	default:
		return "P3"
	}
}

// Trigger creates an Opsgenie alert for a failing check.
//
//   - checkName: human-readable check name
//   - alias:     unique identifier (check UUID) used to link trigger and resolve
//   - errMsg:    error description
//   - severity:  "critical", "warning", or "info"
func (c *OpsgenieClient) Trigger(checkName, alias, errMsg, severity string) error {
	payload := opsgenieAlertPayload{
		Message:     fmt.Sprintf("%s is DOWN", checkName),
		Alias:       alias,
		Description: fmt.Sprintf("Error: %s", errMsg),
		Priority:    MapPriority(severity),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Opsgenie trigger payload: %w", err)
	}

	url := c.baseURL() + "/v2/alerts"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Opsgenie trigger request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("GenieKey %s", c.APIKey))

	resp, err := c.httpDo()(req)
	if err != nil {
		return fmt.Errorf("failed to send Opsgenie trigger request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opsgenie trigger failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Resolve closes an existing Opsgenie alert identified by alias.
func (c *OpsgenieClient) Resolve(alias string) error {
	payload := opsgenieClosePayload{
		Note: "Resolved by Checker",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Opsgenie close payload: %w", err)
	}

	url := fmt.Sprintf("%s/v2/alerts/%s/close?identifierType=alias", c.baseURL(), alias)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Opsgenie close request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("GenieKey %s", c.APIKey))

	resp, err := c.httpDo()(req)
	if err != nil {
		return fmt.Errorf("failed to send Opsgenie close request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opsgenie close failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
