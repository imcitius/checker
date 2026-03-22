package scheduler

import (
	"context"
	"time"

	"checker/internal/alerts"
	"checker/internal/db"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
)

// processEscalation handles escalation policy logic for a failed check.
// When a check fails and has an escalation policy assigned:
//  1. Look up the escalation policy by name
//  2. For each step, check if the check has been DOWN for >= step.DelayMin minutes
//  3. If yes and this step hasn't been notified yet, send alert to step.Channel
//  4. Record the notification in escalation_notifications
func processEscalation(repo db.Repository, checkDef models.CheckDefinition, checkStatus models.CheckStatus, appAlerters []AppAlerter) {
	if checkDef.EscalationPolicyName == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Look up the escalation policy
	policy, err := repo.GetEscalationPolicyByName(ctx, checkDef.EscalationPolicyName)
	if err != nil {
		logrus.Warnf("Escalation policy %q not found for check %s: %v",
			checkDef.EscalationPolicyName, checkDef.UUID, err)
		return
	}

	// Get existing notifications for this check+policy
	existingNotifications, err := repo.GetEscalationNotifications(ctx, checkDef.UUID, policy.Name)
	if err != nil {
		logrus.Errorf("Failed to get escalation notifications for check %s: %v", checkDef.UUID, err)
		return
	}

	// Build a set of already-notified step indices
	notifiedSteps := make(map[int]bool)
	for _, n := range existingNotifications {
		notifiedSteps[n.StepIndex] = true
	}

	// Calculate how long the check has been down.
	// We use the time since the last healthy state (approximated by LastRun when unhealthy).
	// A more accurate approach would track when the check first went unhealthy,
	// but for basic escalation we use the current time minus the earliest
	// unresolved notification or the check's last_run.
	downDuration := time.Since(checkStatus.LastRun)

	// If there are existing notifications, use the earliest one to determine
	// how long the check has been in a failure state.
	if len(existingNotifications) > 0 {
		earliest := existingNotifications[0].NotifiedAt
		for _, n := range existingNotifications {
			if n.NotifiedAt.Before(earliest) {
				earliest = n.NotifiedAt
			}
		}
		downSinceEarliest := time.Since(earliest)
		if downSinceEarliest > downDuration {
			downDuration = downSinceEarliest
		}
	}

	now := time.Now()

	// Build the AlertPayload once — it's the same for all escalation steps
	severity := getEffectiveSeverity(checkDef)
	payload := alerts.AlertPayload{
		CheckName:  checkStatus.CheckName,
		CheckUUID:  checkStatus.UUID,
		Project:    checkStatus.Project,
		CheckGroup: checkStatus.CheckGroup,
		CheckType:  checkStatus.CheckType,
		Message:    checkStatus.Message,
		Severity:   severity,
		Timestamp:  checkStatus.LastRun,
	}

	ownedTypes := buildOwnedTypeSet(appAlerters)

	for i, step := range policy.Steps {
		if notifiedSteps[i] {
			continue // Already notified for this step
		}

		stepDelay := time.Duration(step.DelayMin) * time.Minute

		// For step 0 with delay 0 (immediate), always fire on first failure
		// For other steps, check if enough time has elapsed
		if stepDelay > 0 && downDuration < stepDelay {
			continue // Not enough time has elapsed for this step
		}

		logrus.Infof("Escalation policy %q step %d: sending %s alert for check %s (down for %s, delay=%dm)",
			policy.Name, i, step.Channel, checkDef.UUID, downDuration.Round(time.Second), step.DelayMin)

		alerter, err := resolveAlerter(repo, step.Channel)
		if err != nil {
			logrus.Errorf("Escalation policy %q step %d: failed to resolve channel %q: %v", policy.Name, i, step.Channel, err)
		} else if ownedTypes[alerter.Type()] {
			logrus.Debugf("Escalation policy %q step %d: skipping %s (owned by app alerter)", policy.Name, i, alerter.Type())
		} else if err := alerter.SendAlert(payload); err != nil {
			logrus.Errorf("Escalation policy %q step %d: failed to send alert: %v", policy.Name, i, err)
		}

		// Record the notification regardless of send success to avoid retrying the same step
		notification := models.EscalationNotification{
			CheckUUID:  checkDef.UUID,
			PolicyName: policy.Name,
			StepIndex:  i,
			NotifiedAt: now,
		}
		if err := repo.CreateEscalationNotification(ctx, notification); err != nil {
			logrus.Errorf("Failed to record escalation notification for check %s step %d: %v",
				checkDef.UUID, i, err)
		}
	}
}

// clearEscalationNotifications removes all escalation notification records
// for a check when it recovers, resetting the escalation state for the next incident.
func clearEscalationNotifications(repo db.Repository, checkUUID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.DeleteEscalationNotifications(ctx, checkUUID); err != nil {
		logrus.Errorf("Failed to clear escalation notifications for check %s: %v", checkUUID, err)
	}
}
