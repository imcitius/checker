// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"time"
)

// EdgeSchedulerStats holds execution counters reported by edge probes.
// Defined in models so both checker-core and checker-cloud can use it.
type EdgeSchedulerStats struct {
	ChecksDispatched int64 `json:"checks_dispatched"` // successfully queued to worker pool
	ChecksDeferred   int64 `json:"checks_deferred"`   // worker pool full, retried later
	ChecksExecuted   int64 `json:"checks_executed"`   // completed by workers (healthy + failed)
	ChecksFailed     int64 `json:"checks_failed"`     // completed unhealthy
	TotalChecks      int   `json:"total_checks"`      // current check count in heap
}

// EdgeMessage is the base envelope for Edge WebSocket messages.
// Type values: "config_sync", "config_patch", "result", "heartbeat", "ping", "pong"
type EdgeMessage struct {
	Type string `json:"type"`
}

// EdgeConfigSync is sent SaaS -> Edge as a full config sync.
type EdgeConfigSync struct {
	Type       string                     `json:"type"` // "config_sync"
	Checks     []CheckDefinitionViewModel `json:"checks"`
	ServerTime time.Time                  `json:"server_time"`
}

// EdgeConfigPatch is sent SaaS -> Edge as an incremental config update.
// Action is one of: "add", "update", "delete".
type EdgeConfigPatch struct {
	Type   string                    `json:"type"`            // "config_patch"
	Action string                    `json:"action"`          // "add", "update", "delete"
	Check  *CheckDefinitionViewModel `json:"check,omitempty"` // nil for deletes
	UUID   string                    `json:"uuid,omitempty"`  // populated for deletes
}

// EdgeResult is sent Edge -> SaaS carrying a check execution result.
type EdgeResult struct {
	Type      string        `json:"type"`       // "result"
	CheckUUID string        `json:"check_uuid"`
	IsHealthy bool          `json:"is_healthy"`
	Message   string        `json:"message"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// EdgeHeartbeat is sent Edge -> SaaS to report liveness and capacity.
type EdgeHeartbeat struct {
	Type          string              `json:"type"`          // "heartbeat"
	Version       string              `json:"version"`
	Region        string              `json:"region"`
	WorkerCount   int                 `json:"worker_count"`
	ActiveChecks  int                 `json:"active_checks"`
	UptimeSeconds int64               `json:"uptime_seconds"`
	Stats         *EdgeSchedulerStats `json:"stats,omitempty"`
}
