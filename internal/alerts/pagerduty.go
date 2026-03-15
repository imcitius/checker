package alerts

import (
	"fmt"
)

// PagerDutyEventsURL is the default PagerDuty Events API v2 endpoint.
// It can be overridden in tests.
var PagerDutyEventsURL = "https://events.pagerduty.com/v2/enqueue"

// PagerDutyEvent represents a PagerDuty Events API v2 request.
type PagerDutyEvent struct {
	RoutingKey  string               `json:"routing_key"`
	EventAction string               `json:"event_action"`
	DedupKey    string               `json:"dedup_key"`
	Payload     *PagerDutyPayload    `json:"payload,omitempty"`
}

// PagerDutyPayload is the payload section of a PagerDuty event.
type PagerDutyPayload struct {
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
}

// MapSeverity maps checker severity strings to PagerDuty severity values.
// PagerDuty accepts: critical, error, warning, info.
func MapSeverity(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "warning", "degraded":
		return "warning"
	case "info":
		return "info"
	default:
		return "critical"
	}
}

// SendPagerDutyTrigger sends a trigger event to PagerDuty.
// The dedupKey should be the check UUID so PagerDuty can correlate trigger and resolve.
func SendPagerDutyTrigger(routingKey, dedupKey, checkName, message, severity string) error {
	event := PagerDutyEvent{
		RoutingKey:  routingKey,
		EventAction: "trigger",
		DedupKey:    dedupKey,
		Payload: &PagerDutyPayload{
			Summary:  fmt.Sprintf("%s is DOWN: %s", checkName, message),
			Source:   "checker",
			Severity: MapSeverity(severity),
		},
	}
	return sendPagerDutyEvent(event)
}

// SendPagerDutyResolve sends a resolve event to PagerDuty.
// The dedupKey must match the original trigger's dedupKey (check UUID).
func SendPagerDutyResolve(routingKey, dedupKey, checkName string) error {
	event := PagerDutyEvent{
		RoutingKey:  routingKey,
		EventAction: "resolve",
		DedupKey:    dedupKey,
	}
	return sendPagerDutyEvent(event)
}

func sendPagerDutyEvent(event PagerDutyEvent) error {
	if err := postJSON(PagerDutyEventsURL, event, nil); err != nil {
		return fmt.Errorf("pagerduty event: %w", err)
	}
	return nil
}
