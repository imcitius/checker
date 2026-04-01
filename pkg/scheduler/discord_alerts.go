package scheduler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/internal/discord"
	"github.com/imcitius/checker/pkg/models"
	"github.com/imcitius/checker/internal/web"
)

// DiscordSender abstracts the Discord client methods used by DiscordAppAlerter.
// This enables testing without a real Discord connection.
type DiscordSender interface {
	SendMessage(ctx context.Context, channelID string, payload discord.MessagePayload) (*discord.Message, error)
	EditMessage(ctx context.Context, channelID, messageID string, payload discord.MessagePayload) error
	CreateThread(ctx context.Context, channelID, messageID, name string) (*discord.Channel, error)
	SendThreadReply(ctx context.Context, threadID string, payload discord.MessagePayload) (*discord.Message, error)
}

// DiscordAppAlerter sends Discord Bot alerts with thread tracking and silence support.
type DiscordAppAlerter struct {
	client         DiscordSender
	repo           db.Repository
	defaultChannel string // Discord channel ID
}

// NewDiscordAppAlerter creates a new DiscordAppAlerter. Returns nil if the client is nil.
func NewDiscordAppAlerter(client *discord.DiscordClient, repo db.Repository, defaultChannel string) *DiscordAppAlerter {
	if client == nil {
		return nil
	}
	return &DiscordAppAlerter{
		client:         client,
		repo:           repo,
		defaultChannel: defaultChannel,
	}
}

// newDiscordAppAlerterWithSender creates a DiscordAppAlerter with a custom DiscordSender (for testing).
func newDiscordAppAlerterWithSender(sender DiscordSender, repo db.Repository, defaultChannel string) *DiscordAppAlerter {
	return &DiscordAppAlerter{
		client:         sender,
		repo:           repo,
		defaultChannel: defaultChannel,
	}
}

// OwnedTypes returns the standard alerter type strings that DiscordAppAlerter supersedes.
func (da *DiscordAppAlerter) OwnedTypes() []string { return []string{"discord"} }

// SendAlert posts a Discord alert for a failing check. If an unresolved thread exists
// and this is NOT a new incident, it posts a thread reply. If isNewIncident is true
// (check transitioned from healthy to unhealthy), any stale unresolved threads are
// resolved first and a fresh thread is created.
func (da *DiscordAppAlerter) SendAlert(ctx context.Context, checkDef models.CheckDefinition, status models.CheckStatus, isNewIncident bool) {
	// Check if silenced (per-channel: "discord")
	silenced, err := da.repo.IsChannelSilenced(ctx, checkDef.UUID, checkDef.Project, "discord")
	if err != nil {
		logrus.Errorf("Failed to check silence status for %s: %v", checkDef.UUID, err)
	}
	if silenced {
		logrus.Infof("Check %s (project %s) is silenced for discord, skipping discord alert", checkDef.UUID, checkDef.Project)
		return
	}

	channelID := da.defaultChannel
	if channelID == "" {
		logrus.Errorf("No Discord Bot channel configured for check %s", checkDef.UUID)
		return
	}

	alertInfo := discord.CheckAlertInfo{
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
	thread, err := da.repo.GetUnresolvedDiscordThread(ctx, checkDef.UUID)
	if err == nil && thread.ThreadID != "" {
		if isNewIncident {
			// This is a new failure incident but there's a stale unresolved thread
			// from a previous incident. Resolve the stale thread so we create a fresh one below.
			logrus.Warnf("Check %s has a stale unresolved discord thread from a previous incident, resolving it before creating new thread", checkDef.UUID)
			if resolveErr := da.repo.ResolveDiscordThread(ctx, checkDef.UUID); resolveErr != nil {
				logrus.Errorf("Failed to resolve stale discord thread for check %s: %v", checkDef.UUID, resolveErr)
			}
		} else {
			// Ongoing failure — reply to existing thread
			replyPayload := discord.BuildAlertMessage(alertInfo)
			// Remove buttons from thread replies (only the original message has buttons)
			replyPayload.Components = nil
			replyMsg, replyErr := da.client.SendThreadReply(ctx, thread.ThreadID, replyPayload)
			if replyErr != nil {
				logrus.Errorf("Failed to send Discord thread reply for check %s: %v", checkDef.UUID, replyErr)
				return
			}
			logrus.Infof("Discord thread reply sent for check %s (thread: %s, reply: %s)", checkDef.UUID, thread.ThreadID, replyMsg.ID)
			return
		}
	}

	// New failure: post new top-level alert message with embed + buttons
	alertPayload := discord.BuildAlertMessage(alertInfo)
	msg, err := da.client.SendMessage(ctx, channelID, alertPayload)
	if err != nil {
		logrus.Errorf("Failed to send Discord Bot alert for check %s: %v", checkDef.UUID, err)
		return
	}

	logrus.Infof("Discord Bot alert sent for check %s (msg ID: %s)", checkDef.UUID, msg.ID)

	// Create a thread from the alert message
	threadName := fmt.Sprintf("🔴 %s — incident", checkDef.Name)
	if len(threadName) > 100 {
		threadName = threadName[:100] // Discord thread name limit
	}
	thread2, err := da.client.CreateThread(ctx, channelID, msg.ID, threadName)
	if err != nil {
		logrus.Errorf("Failed to create Discord thread for check %s: %v", checkDef.UUID, err)
		// Still track the message even without a thread
		if dbErr := da.repo.CreateDiscordThread(ctx, checkDef.UUID, channelID, msg.ID, ""); dbErr != nil {
			logrus.Errorf("Failed to track Discord thread for check %s: %v", checkDef.UUID, dbErr)
		}
	} else {
		logrus.Infof("Discord thread created for check %s (thread: %s)", checkDef.UUID, thread2.ID)

		// Post error snapshot as the first thread reply
		snapshotPayload := discord.MessagePayload{
			Embeds: []discord.Embed{{
				Title:       "Error Snapshot",
				Description: fmt.Sprintf("```%s```", alertInfo.Message),
				Color:       discord.ColorGray,
				Fields: []discord.EmbedField{
					{Name: "Target", Value: fmt.Sprintf("`%s`", alertInfo.Target), Inline: true},
				},
			}},
		}
		if _, snapshotErr := da.client.SendThreadReply(ctx, thread2.ID, snapshotPayload); snapshotErr != nil {
			logrus.Errorf("Failed to post error snapshot for check %s: %v", checkDef.UUID, snapshotErr)
		}

		// Track the thread in DB
		if dbErr := da.repo.CreateDiscordThread(ctx, checkDef.UUID, channelID, msg.ID, thread2.ID); dbErr != nil {
			logrus.Errorf("Failed to track Discord thread for check %s: %v", checkDef.UUID, dbErr)
		}
	}

	// Record alert event in history
	alertEvent := models.AlertEvent{
		CheckUUID: checkDef.UUID,
		CheckName: checkDef.Name,
		Project:   checkDef.Project,
		GroupName: checkDef.GroupName,
		CheckType: checkDef.Type,
		Message:   status.Message,
		AlertType: "discord",
	}
	if err := da.repo.CreateAlertEvent(ctx, alertEvent); err != nil {
		logrus.Errorf("Failed to record alert event for check %s: %v", checkDef.UUID, err)
	}

	// Broadcast new alert to connected WebSocket clients
	web.BroadcastAlertNew(alertEvent)
}

// HandleRecovery resolves an existing Discord thread when a check recovers.
func (da *DiscordAppAlerter) HandleRecovery(ctx context.Context, checkDef models.CheckDefinition) {
	thread, err := da.repo.GetUnresolvedDiscordThread(ctx, checkDef.UUID)
	if err != nil {
		logrus.Debugf("No unresolved Discord thread for check %s: %v", checkDef.UUID, err)
		return
	}

	if thread.MessageID == "" {
		return
	}

	logrus.Infof("Resolving Discord thread for check %s (channel: %s, msg: %s, thread: %s)", checkDef.UUID, thread.ChannelID, thread.MessageID, thread.ThreadID)

	// Fetch the original error message from alert history so it can be preserved
	// in the resolved message for context.
	var originalError string
	unresolved := false
	events, _, err := da.repo.GetAlertHistory(ctx, 1, 0, models.AlertHistoryFilters{
		CheckUUID:  checkDef.UUID,
		IsResolved: &unresolved,
	})
	if err == nil && len(events) > 0 {
		originalError = events[0].Message
	}

	alertInfo := discord.CheckAlertInfo{
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

	// Edit the original alert message to green (resolved) with buttons disabled
	resolvedPayload := discord.BuildResolvedAlertMessage(alertInfo, "auto-recovery")
	if editErr := da.client.EditMessage(ctx, thread.ChannelID, thread.MessageID, resolvedPayload); editErr != nil {
		logrus.Errorf("Failed to edit Discord alert to resolved for check %s: %v (will still resolve thread in DB)", checkDef.UUID, editErr)
	}

	// Post recovery reply in thread
	if thread.ThreadID != "" {
		resolveReply := discord.BuildResolveMessage(alertInfo)
		if _, replyErr := da.client.SendThreadReply(ctx, thread.ThreadID, resolveReply); replyErr != nil {
			logrus.Errorf("Failed to post Discord recovery reply for check %s: %v", checkDef.UUID, replyErr)
		}
	}

	// Always resolve the thread in DB — this is the critical state change that ensures
	// future failures create a new thread instead of replying to this resolved one.
	if err := da.repo.ResolveDiscordThread(ctx, checkDef.UUID); err != nil {
		logrus.Errorf("Failed to resolve Discord thread for check %s: %v", checkDef.UUID, err)
	}

	// Resolve alert event in history
	if err := da.repo.ResolveAlertEvent(ctx, checkDef.UUID); err != nil {
		logrus.Errorf("Failed to resolve alert event for check %s: %v", checkDef.UUID, err)
	}

	// Broadcast alert resolved to connected WebSocket clients
	web.BroadcastAlertResolved(checkDef.UUID)

	// Deactivate check-level silence on recovery so new alerts can fire if the check fails again
	if err := da.repo.DeactivateSilence(ctx, "check", checkDef.UUID); err != nil {
		logrus.Errorf("Failed to deactivate silence for check %s: %v", checkDef.UUID, err)
	} else {
		logrus.Infof("Deactivated check-level silence for recovered check %s", checkDef.UUID)
	}
}
