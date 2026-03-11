package scheduler

import (
	"context"

	"github.com/sirupsen/logrus"

	"checker/internal/db"
	"checker/internal/models"
	"checker/internal/slack"
)

// SlackAlerter sends Slack App alerts with thread tracking and silence support.
type SlackAlerter struct {
	client         *slack.SlackClient
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

// SendAlert posts a Slack alert for a failing check. If an unresolved thread exists,
// it posts a thread reply instead of a new message.
func (sa *SlackAlerter) SendAlert(ctx context.Context, checkDef models.CheckDefinition, status models.CheckStatus) {
	// Check if silenced
	silenced, err := sa.repo.IsCheckSilenced(ctx, checkDef.UUID, checkDef.Project)
	if err != nil {
		logrus.Errorf("Failed to check silence status for %s: %v", checkDef.UUID, err)
	}
	if silenced {
		logrus.Infof("Check %s (project %s) is silenced, skipping slack_app alert", checkDef.UUID, checkDef.Project)
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
	}

	// Check for existing unresolved thread (ongoing failure)
	thread, err := sa.repo.GetUnresolvedThread(ctx, checkDef.UUID)
	if err == nil && thread.ThreadTs != "" {
		replyTs, err := sa.client.SendAlertReply(ctx, thread.ChannelID, thread.ThreadTs, alertInfo)
		if err != nil {
			logrus.Errorf("Failed to send Slack App thread reply for check %s: %v", checkDef.UUID, err)
			return
		}
		logrus.Infof("Slack App thread reply sent for check %s (thread: %s, reply: %s)", checkDef.UUID, thread.ThreadTs, replyTs)
		return
	}

	// New failure: post new top-level alert
	messageTs, err := sa.client.PostAlert(ctx, channelID, alertInfo)
	if err != nil {
		logrus.Errorf("Failed to send Slack App alert for check %s: %v", checkDef.UUID, err)
		return
	}

	logrus.Infof("Slack App alert sent for check %s (ts: %s)", checkDef.UUID, messageTs)

	// Track the thread
	if err := sa.repo.CreateSlackThread(ctx, checkDef.UUID, channelID, messageTs, messageTs); err != nil {
		logrus.Errorf("Failed to track Slack thread for check %s: %v", checkDef.UUID, err)
	}
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

	alertInfo := slack.CheckAlertInfo{
		UUID:      checkDef.UUID,
		Name:      checkDef.Name,
		Project:   checkDef.Project,
		Group:     checkDef.GroupName,
		CheckType: checkDef.Type,
		Message:   "Check is healthy again",
		IsHealthy: true,
		Frequency: checkDef.Duration,
	}

	if err := sa.client.SendResolve(ctx, alertInfo, thread.ThreadTs, thread.ChannelID); err != nil {
		logrus.Errorf("Failed to post Slack resolution for check %s: %v", checkDef.UUID, err)
	}

	if err := sa.repo.ResolveThread(ctx, checkDef.UUID); err != nil {
		logrus.Errorf("Failed to resolve Slack thread for check %s: %v", checkDef.UUID, err)
	}

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
