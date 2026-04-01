package models

import "time"

// CheckResult stores a single health check execution result from a specific region.
// Used by multi-region consensus to collect per-probe results before evaluation.
type CheckResult struct {
	ID          int64      `json:"id"`
	CheckUUID   string     `json:"check_uuid"`
	Region      string     `json:"region"`
	IsHealthy   bool       `json:"is_healthy"`
	Message     string     `json:"message"`
	CreatedAt   time.Time  `json:"created_at"`
	CycleKey    time.Time  `json:"cycle_key"`     // runTime.Truncate(checkDuration) — groups results per cycle
	EvaluatedAt *time.Time `json:"evaluated_at,omitempty"`
}
