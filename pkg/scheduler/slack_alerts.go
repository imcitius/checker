package scheduler

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
	"github.com/imcitius/checker/internal/slack"
	"github.com/imcitius/checker/internal/web"
)

// SlackSender abstracts the Slack client methods used by SlackAlerter.
// This enables testing without a real Slack connection.
type SlackSender interface {
	PostAlert(ctx context.Context, channelID string, info slack.CheckAlertInfo) (string, error)
	SendAlertReply(ctx context.Context, channelID, threadTS string, info slack.CheckAlertInfo) (string, error)
	SendResolve(ctx context.Context, info slack.CheckAlertInfo, originalThreadTS, channelID string) error
	PostErrorSnapshotReply(ctx context.Context, channelID, threadTS string, info slack.CheckAlertInfo) (string, error)
}

// SlackAlerter sends Slack App alerts with thread tracking and silence support.
type SlackAlerter struct {
	client         SlackSender
	repo           db.Repository
	defaultChannel string
}

// NewSlackAlerter creates a new SlackAlerter. Returns nil if the client is nil.
func NewSlackAlerter(client *slack.SlackClient, repo db.Repository, defaultChannel string) *SlackAlerter {
	if client == nil {
		return nil
	}
	return &SlackAlerter{
		client:         client,
		repo:           repo,
		defaultChannel: defaultChannel,
	}
}

// newSlackAlerterWithSender creates a SlackAlerter with a custom SlackSender (for testing).
func newSlackAlerterWithSender(sender SlackSender, repo db.Repository, defaultChannel string) *SlackAlerter {
	return &SlackAlerter{
		client:         sender,
		repo:           repo,
		defaultChannel: defaultChannel,
	}
}

// SendAlert posts a Slack alert for a failing check. If an unresolved thread exists
// and this is NOT a new incident, it posts a thread reply. If isNewIncident is true
// (check transitioned from healthy to unhealthy), any stale unresolved threads are
// resolved first and a fresh thread is created.
func (sa *SlackAlerter) SendAlert(ctx context.Context, checkDef models.CheckDefinition, status models.CheckStatus, isNewIncident bool) {
	// Check if silenced (per-channel: "slack")
	silenced, err := sa.repo.IsChannelSilenced(ctx, checkDef.UUID, checkDef.Project, "slack")
	if err != nil {
		logrus.Errorf("Failed to check silence status for %s: %v", checkDef.UUID, err)
	}
	if silenced {
		logrus.Infof("Check %s (project %s) is silenced for slack, skipping slack_app alert", checkDef.UUID, checkDef.Project)
		return
	}

	channelID := sa.defaultChannel
	if channelID == "" {
		logrus.Errorf("No Slack App channel configured for check %s", checkDef.UUID)
		return
	}

	alertInfo := slack.CheckAlertInfo{
		UUID:      checkDef.UUID,
		Name:      checkDef.Name,
		Project:   checkDef.Project,
		Group:     checkDef.GroupName,
		CheckType: checkDef.Type,
		Message:   status.Message,
		IsHealthy: false,
		Frequency: checkDef.Duration,
		Target:    checkTarget(checkDef),
	}

	// Check for existing unresolved thread (ongoing failure)
	thread, err := sa.repo.GetUnresolvedThread(ctx, checkDef.UUID)
	if err == nil && thread.ThreadTs != "" {
		if isNewIncident {
			// This is a new failure incident but there's a stale unresolved thread
			// from a previous incident. This can happen if HandleRecovery was missed
			// (e.g., due to stale in-memory state or a Slack API failure).
			// Resolve the stale thread so we create a fresh one below.
			logrus.Warnf("Check %s has a stale unresolved thread from a previous incident, resolving it before creating new thread", checkDef.UUID)
			if resolveErr := sa.repo.ResolveThread(ctx, checkDef.UUID); resolveErr != nil {
				logrus.Errorf("Failed to resolve stale thread for check %s: %v", checkDef.UUID, resolveErr)
			}
		} else {
			// Ongoing failure — reply to existing thread
			replyTs, replyErr := sa.client.SendAlertReply(ctx, thread.ChannelID, thread.ThreadTs, alertInfo)
			if replyErr != nil {
				logrus.Errorf("Failed to send Slack App thread reply for check %s: %v", checkDef.UUID, replyErr)
				return
			}
			logrus.Infof("Slack App thread reply sent for check %s (thread: %s, reply: %s)", checkDef.UUID, thread.ThreadTs, replyTs)
			return
		}
	}

	// New failure: post new top-level alert
	messageTs, err := sa.client.PostAlert(ctx, channelID, alertInfo)
	if err != nil {
		logrus.Errorf("Failed to send Slack App alert for check %s: %v", checkDef.UUID, err)
		return
	}

	logrus.Infof("Slack App alert sent for check %s (ts: %s)", checkDef.UUID, messageTs)

	// Post immutable error snapshot as the first thread reply.
	// This reply is never edited on resolve/silence/ack, preserving error context.
	snapshotTs, snapshotErr := sa.client.PostErrorSnapshotReply(ctx, channelID, messageTs, alertInfo)
	if snapshotErr != nil {
		logrus.Errorf("Failed to post error snapshot for check %s: %v", checkDef.UUID, snapshotErr)
	} else {
		logrus.Infof("Error snapshot posted for check %s (thread: %s, reply: %s)", checkDef.UUID, messageTs, snapshotTs)
	}

	// Track the thread
	if err := sa.repo.CreateSlackThread(ctx, checkDef.UUID, channelID, messageTs, messageTs); err != nil {
		logrus.Errorf("Failed to track Slack thread for check %s: %v", checkDef.UUID, err)
	}

	// Record alert event in history
	alertEvent := models.AlertEvent{
		CheckUUID: checkDef.UUID,
		CheckName: checkDef.Name,
		Project:   checkDef.Project,
		GroupName: checkDef.GroupName,
		CheckType: checkDef.Type,
		Message:   status.Message,
		AlertType: "slack",
	}
	if err := sa.repo.CreateAlertEvent(ctx, alertEvent); err != nil {
		logrus.Errorf("Failed to record alert event for check %s: %v", checkDef.UUID, err)
	}

	// Broadcast new alert to connected WebSocket clients
	web.BroadcastAlertNew(alertEvent)
}

// HandleRecovery resolves an existing Slack thread when a check recovers.
func (sa *SlackAlerter) HandleRecovery(ctx context.Context, checkDef models.CheckDefinition) {
	thread, err := sa.repo.GetUnresolvedThread(ctx, checkDef.UUID)
	if err != nil {
		logrus.Debugf("No unresolved Slack thread for check %s: %v", checkDef.UUID, err)
		return
	}

	if thread.ThreadTs == "" {
		return
	}

	logrus.Infof("Resolving Slack thread for check %s (channel: %s, thread: %s)", checkDef.UUID, thread.ChannelID, thread.ThreadTs)

	// Fetch the original error message from alert history so it can be preserved
	// in the resolved message for context.
	var originalError string
	unresolved := false
	events, _, err := sa.repo.GetAlertHistory(ctx, 1, 0, models.AlertHistoryFilters{
		CheckUUID:  checkDef.UUID,
		IsResolved: &unresolved,
	})
	if err == nil && len(events) > 0 {
		originalError = events[0].Message
	}

	alertInfo := slack.CheckAlertInfo{
		UUID:          checkDef.UUID,
		Name:          checkDef.Name,
		Project:       checkDef.Project,
		Group:         checkDef.GroupName,
		CheckType:     checkDef.Type,
		Message:       "Check is healthy again",
		IsHealthy:     true,
		Frequency:     checkDef.Duration,
		OriginalError: originalError,
	}

	// Send Slack resolution message (cosmetic — failure here should NOT prevent
	// the DB thread from being resolved, which is what matters for future decisions).
	if err := sa.client.SendResolve(ctx, alertInfo, thread.ThreadTs, thread.ChannelID); err != nil {
		logrus.Errorf("Failed to post Slack resolution for check %s: %v (will still resolve thread in DB)", checkDef.UUID, err)
	}

	// Always resolve the thread in DB — this is the critical state change that ensures
	// future failures create a new thread instead of replying to this resolved one.
	if err := sa.repo.ResolveThread(ctx, checkDef.UUID); err != nil {
		logrus.Errorf("Failed to resolve Slack thread for check %s: %v", checkDef.UUID, err)
	}

	// Resolve alert event in history
	if err := sa.repo.ResolveAlertEvent(ctx, checkDef.UUID); err != nil {
		logrus.Errorf("Failed to resolve alert event for check %s: %v", checkDef.UUID, err)
	}

	// Broadcast alert resolved to connected WebSocket clients
	web.BroadcastAlertResolved(checkDef.UUID)

	if err := sa.repo.UpdateSlackThread(ctx, checkDef.UUID, "", ""); err != nil {
		logrus.Errorf("Failed to clear Slack thread on check_definitions for check %s: %v", checkDef.UUID, err)
	}

	// Deactivate check-level silence on recovery so new alerts can fire if the check fails again
	if err := sa.repo.DeactivateSilence(ctx, "check", checkDef.UUID); err != nil {
		logrus.Errorf("Failed to deactivate silence for check %s: %v", checkDef.UUID, err)
	} else {
		logrus.Infof("Deactivated check-level silence for recovered check %s", checkDef.UUID)
	}
}

// OwnedTypes returns the standard alerter type strings that SlackAlerter supersedes.
func (sa *SlackAlerter) OwnedTypes() []string { return []string{"slack"} }

// checkTarget computes a human-readable target string from a CheckDefinition.
func checkTarget(def models.CheckDefinition) string {
	if def.Config != nil {
		return def.Config.GetTarget()
	}
	return ""
}
