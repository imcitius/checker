// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"time"
)

const (
	ErrEmptyHost = "empty host"
	ErrEmptyPort = "port is empty"
	ErrICMPError = "icmp error: %s"
	ErrPacketLoss = "icmp error: %f percent packet loss"

	ErrHTTPClientConstruction = "http client construction error"
)

// Checker is an interface that all health checks should implement.
// It defines a universal Run method.
type Checker interface {
	// Run executes the health check and returns:
	// - a bool indicating if the check passed, and
	// - a message detailing the result.
	Run() (time.Duration, error)
}
