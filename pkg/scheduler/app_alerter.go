// SPDX-License-Identifier: BUSL-1.1

package scheduler

import (
	"context"

	"github.com/imcitius/checker/pkg/models"
)

// AppAlerter is the interface for stateful alerters (Slack App, Telegram App, etc.)
// that manage thread tracking, silence checking, and alert history.
// The scheduler dispatches to all registered AppAlerters without knowing their types.
type AppAlerter interface {
	// SendAlert dispatches an alert for a failing check.
	SendAlert(ctx context.Context, checkDef models.CheckDefinition, status models.CheckStatus, isNewIncident bool)

	// HandleRecovery resolves an existing alert when a check recovers.
	HandleRecovery(ctx context.Context, checkDef models.CheckDefinition)

	// OwnedTypes returns the standard alerter type strings (e.g. "slack", "telegram")
	// that this app alerter supersedes. The scheduler skips standard alerter channels
	// whose Type() matches any owned type, preventing duplicate alerts.
	OwnedTypes() []string
}

// buildOwnedTypeSet collects all owned types from app alerters into a lookup set.
func buildOwnedTypeSet(appAlerters []AppAlerter) map[string]bool {
	owned := make(map[string]bool)
	for _, aa := range appAlerters {
		for _, t := range aa.OwnedTypes() {
			owned[t] = true
		}
	}
	return owned
}
