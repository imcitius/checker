package scheduler

import (
	"context"

	"github.com/sirupsen/logrus"

	"checker/pkg/db"
	"checker/pkg/models"
	"checker/internal/telegram"
	"checker/internal/web"
)

// TelegramSender abstracts the Telegram client methods used by TelegramAppAlerter.
// This enables testing without a real Telegram connection.
type TelegramSender interface {
	PostAlert(ctx context.Context, chatID string, info telegram.CheckAlertInfo) (int, error)
	PostAlertReply(ctx context.Context, chatID string, replyToMsgID int, info telegram.CheckAlertInfo) (int, error)
	SendResolve(ctx context.Context, info telegram.CheckAlertInfo, originalMsgID int, chatID string) error
	PostErrorSnapshotReply(ctx context.Context, chatID string, replyToMsgID int, info telegram.CheckAlertInfo) (int, error)
}

// TelegramAppAlerter sends Telegram alerts with thread tracking and silence support.
type TelegramAppAlerter struct {
	client        TelegramSender
	repo          db.Repository
	defaultChatID string
}

// NewTelegramAppAlerter creates a new TelegramAppAlerter. Returns nil if the client is nil.
func NewTelegramAppAlerter(client *telegram.TelegramClient, repo db.Repository, defaultChatID string) *TelegramAppAlerter {
	if client == nil {
		return nil
	}
	return &TelegramAppAlerter{
		client:        client,
		repo:          repo,
		defaultChatID: defaultChatID,
	}
}

// newTelegramAppAlerterWithSender creates a TelegramAppAlerter with a custom TelegramSender (for testing).
func newTelegramAppAlerterWithSender(sender TelegramSender, repo db.Repository, defaultChatID string) *TelegramAppAlerter {
	return &TelegramAppAlerter{
		client:        sender,
		repo:          repo,
		defaultChatID: defaultChatID,
	}
}

// SendAlert posts a Telegram alert for a failing check. If an unresolved thread exists
// and this is NOT a new incident, it posts a reply. If isNewIncident is true
// (check transitioned from healthy to unhealthy), any stale unresolved threads are
// resolved first and a fresh thread is created.
func (ta *TelegramAppAlerter) SendAlert(ctx context.Context, checkDef models.CheckDefinition, status models.CheckStatus, isNewIncident bool) {
	// Check if silenced (per-channel: "telegram")
	silenced, err := ta.repo.IsChannelSilenced(ctx, checkDef.UUID, checkDef.Project, "telegram")
	if err != nil {
		logrus.Errorf("Failed to check silence status for %s: %v", checkDef.UUID, err)
	}
	if silenced {
		logrus.Infof("Check %s (project %s) is silenced for telegram, skipping telegram alert", checkDef.UUID, checkDef.Project)
		return
	}

	chatID := ta.defaultChatID
	if chatID == "" {
		logrus.Errorf("No Telegram chat ID configured for check %s", checkDef.UUID)
		return
	}

	alertInfo := telegram.CheckAlertInfo{
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
	thread, err := ta.repo.GetUnresolvedTelegramThread(ctx, checkDef.UUID)
	if err == nil && thread.MessageID != 0 {
		if isNewIncident {
			// This is a new failure incident but there's a stale unresolved thread
			// from a previous incident. Resolve the stale thread so we create a fresh one below.
			logrus.Warnf("Check %s has a stale unresolved telegram thread from a previous incident, resolving it before creating new thread", checkDef.UUID)
			if resolveErr := ta.repo.ResolveTelegramThread(ctx, checkDef.UUID); resolveErr != nil {
				logrus.Errorf("Failed to resolve stale telegram thread for check %s: %v", checkDef.UUID, resolveErr)
			}
		} else {
			// Ongoing failure — reply to existing thread
			replyMsgID, replyErr := ta.client.PostAlertReply(ctx, thread.ChatID, thread.MessageID, alertInfo)
			if replyErr != nil {
				logrus.Errorf("Failed to send Telegram thread reply for check %s: %v", checkDef.UUID, replyErr)
				return
			}
			logrus.Infof("Telegram thread reply sent for check %s (original msg: %d, reply: %d)", checkDef.UUID, thread.MessageID, replyMsgID)
			return
		}
	}

	// New failure: post new top-level alert
	messageID, err := ta.client.PostAlert(ctx, chatID, alertInfo)
	if err != nil {
		logrus.Errorf("Failed to send Telegram alert for check %s: %v", checkDef.UUID, err)
		return
	}

	logrus.Infof("Telegram alert sent for check %s (msg ID: %d)", checkDef.UUID, messageID)

	// Post immutable error snapshot as the first reply.
	// This reply is never edited on resolve/silence/ack, preserving error context.
	snapshotMsgID, snapshotErr := ta.client.PostErrorSnapshotReply(ctx, chatID, messageID, alertInfo)
	if snapshotErr != nil {
		logrus.Errorf("Failed to post error snapshot for check %s: %v", checkDef.UUID, snapshotErr)
	} else {
		logrus.Infof("Error snapshot posted for check %s (original msg: %d, reply: %d)", checkDef.UUID, messageID, snapshotMsgID)
	}

	// Track the thread
	if err := ta.repo.CreateTelegramThread(ctx, checkDef.UUID, chatID, messageID); err != nil {
		logrus.Errorf("Failed to track Telegram thread for check %s: %v", checkDef.UUID, err)
	}

	// Record alert event in history
	alertEvent := models.AlertEvent{
		CheckUUID: checkDef.UUID,
		CheckName: checkDef.Name,
		Project:   checkDef.Project,
		GroupName: checkDef.GroupName,
		CheckType: checkDef.Type,
		Message:   status.Message,
		AlertType: "telegram",
	}
	if err := ta.repo.CreateAlertEvent(ctx, alertEvent); err != nil {
		logrus.Errorf("Failed to record alert event for check %s: %v", checkDef.UUID, err)
	}

	// Broadcast new alert to connected WebSocket clients
	web.BroadcastAlertNew(alertEvent)
}

// OwnedTypes returns the standard alerter type strings that TelegramAppAlerter supersedes.
func (ta *TelegramAppAlerter) OwnedTypes() []string { return []string{"telegram"} }

// HandleRecovery resolves an existing Telegram thread when a check recovers.
func (ta *TelegramAppAlerter) HandleRecovery(ctx context.Context, checkDef models.CheckDefinition) {
	thread, err := ta.repo.GetUnresolvedTelegramThread(ctx, checkDef.UUID)
	if err != nil {
		logrus.Debugf("No unresolved Telegram thread for check %s: %v", checkDef.UUID, err)
		return
	}

	if thread.MessageID == 0 {
		return
	}

	logrus.Infof("Resolving Telegram thread for check %s (chat: %s, msg: %d)", checkDef.UUID, thread.ChatID, thread.MessageID)

	// Fetch the original error message from alert history so it can be preserved
	// in the resolved message for context.
	var originalError string
	unresolved := false
	events, _, err := ta.repo.GetAlertHistory(ctx, 1, 0, models.AlertHistoryFilters{
		CheckUUID:  checkDef.UUID,
		IsResolved: &unresolved,
	})
	if err == nil && len(events) > 0 {
		originalError = events[0].Message
	}

	alertInfo := telegram.CheckAlertInfo{
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

	// Send Telegram resolution message (cosmetic — failure here should NOT prevent
	// the DB thread from being resolved, which is what matters for future decisions).
	if err := ta.client.SendResolve(ctx, alertInfo, thread.MessageID, thread.ChatID); err != nil {
		logrus.Errorf("Failed to post Telegram resolution for check %s: %v (will still resolve thread in DB)", checkDef.UUID, err)
	}

	// Always resolve the thread in DB — this is the critical state change that ensures
	// future failures create a new thread instead of replying to this resolved one.
	if err := ta.repo.ResolveTelegramThread(ctx, checkDef.UUID); err != nil {
		logrus.Errorf("Failed to resolve Telegram thread for check %s: %v", checkDef.UUID, err)
	}

	// Resolve alert event in history
	if err := ta.repo.ResolveAlertEvent(ctx, checkDef.UUID); err != nil {
		logrus.Errorf("Failed to resolve alert event for check %s: %v", checkDef.UUID, err)
	}

	// Broadcast alert resolved to connected WebSocket clients
	web.BroadcastAlertResolved(checkDef.UUID)

	// Deactivate check-level silence on recovery so new alerts can fire if the check fails again
	if err := ta.repo.DeactivateSilence(ctx, "check", checkDef.UUID); err != nil {
		logrus.Errorf("Failed to deactivate silence for check %s: %v", checkDef.UUID, err)
	} else {
		logrus.Infof("Deactivated check-level silence for recovered check %s", checkDef.UUID)
	}
}
