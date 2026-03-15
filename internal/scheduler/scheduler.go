package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"checker/internal/alerts"
	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
)

const (
	DefaultWorkerPoolSize = 50
	SyncInterval          = 10 * time.Second
)

// Scheduler manages the lifecycle of health checks
type Scheduler struct {
	workerPool   *WorkerPool
	checkHeap    *CheckHeap
	checkMap     map[string]*CheckItem // Map UUID -> CheckItem
	lock         sync.Mutex
	repo         db.Repository
	slackAlerter *SlackAlerter
}

// NewScheduler creates a new scheduler instance
func NewScheduler(repo db.Repository, slackAlerter *SlackAlerter) *Scheduler {
	h := &CheckHeap{}
	heap.Init(h)
	return &Scheduler{
		workerPool:   NewWorkerPool(DefaultWorkerPoolSize, repo, slackAlerter),
		checkHeap:    h,
		checkMap:     make(map[string]*CheckItem),
		repo:         repo,
		slackAlerter: slackAlerter,
	}
}

// RunScheduler starts the health check scheduler.
func RunScheduler(ctx context.Context, cfg *config.Config, repo db.Repository, slackAlerter *SlackAlerter) error {
	logrus.Info("Starting event-driven health check scheduler")

	s := NewScheduler(repo, slackAlerter)

	// Start worker pool
	s.workerPool.Start()
	defer s.workerPool.Stop()

	// Initial Sync
	if err := s.Sync(ctx); err != nil {
		logrus.Errorf("Initial sync failed: %v", err)
	}

	// Create sync ticker
	syncTicker := time.NewTicker(SyncInterval)
	defer syncTicker.Stop()

	for {
		// Determine wait time for next check
		var waitDuration time.Duration

		s.lock.Lock()
		nextItem := s.checkHeap.Peek()
		s.lock.Unlock()

		if nextItem != nil {
			waitDuration = time.Until(nextItem.NextRun)
			if waitDuration < 0 {
				waitDuration = 0
			}
		} else {
			// No checks, wait for sync
			waitDuration = SyncInterval
		}

		// Cap wait time at SyncInterval (or slightly less to ensure we catch sync signal)
		// Actually, we use select cases, so we don't need to cap it manually if we have a separate channel for sync.
		// But timer allocation optimization:

		timer := time.NewTimer(waitDuration)

		select {
		case <-ctx.Done():
			timer.Stop()
			logrus.Info("Scheduler shutting down")
			return ctx.Err()

		case <-syncTicker.C:
			timer.Stop()
			// logrus.Debug("Syncing checks...")
			if err := s.Sync(ctx); err != nil {
				logrus.Errorf("Sync failed: %v", err)
			}

		case <-timer.C:
			// Time to maybe run a check
			s.processNextCheck()
		}
	}
}

// timeoutDurationStrHelper converts generic duration string to time.Duration safely
func parseDuration(d string) time.Duration {
	dur, _ := time.ParseDuration(d)
	if dur == 0 {
		return time.Minute // Default fallback
	}
	return dur
}

func (s *Scheduler) processNextCheck() {
	s.lock.Lock()
	defer s.lock.Unlock()

	item := s.checkHeap.Peek()
	if item == nil {
		return
	}

	now := time.Now()
	if item.NextRun.After(now) {
		// Spurious wake-up or timing issue, just return and let loop recalculate wait
		return
	}

	// Pop the item
	heap.Pop(s.checkHeap)

	// Submit to worker pool if enabled
	if item.CheckDef.Enabled {
		s.workerPool.Submit(item.CheckDef)
	}

	// Schedule next run
	// Logic: NextRun = Now + Duration
	// This creates a "Fixed Delay" schedule (start to start >= duration + execution time wait)
	// Usage of 'processNextCheck' implies we just dispatched it.
	// To minimize drift, we could use item.NextRun.Add(duration), but if we are lagging, we might spiral.
	// Let's use Now + Duration for robustness.

	dur := parseDuration(item.CheckDef.Duration)
	item.NextRun = now.Add(dur)

	// Push back to heap
	heap.Push(s.checkHeap, item)
}

// Sync fetches checks from DB and updates the heap
func (s *Scheduler) Sync(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	defs, err := s.getAllChecks(ctx)
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	activeUUIDs := make(map[string]bool)

	for _, def := range defs {
		activeUUIDs[def.UUID] = true

		item, exists := s.checkMap[def.UUID]
		if exists {
			// Update existing definition
			// If duration changed, we might want to reschedule?
			// For simplicity, update ref, and it will be picked up on next run.
			// Only if Enabled changed from false->true we might want to run sooner?
			// Let's just update the def.
			item.CheckDef = def

			// If it was disabled and now enabled, ensure it's in heap?
			// We keep disabled items in heap but just don't execute them.
			// Simpler to remove disabled from heap?
			// Keeping them is easier for now to maintain NextRun state.
		} else {
			// New check
			newItem := &CheckItem{
				CheckDef: def,
				NextRun:  time.Now(), // Run immediately
			}
			s.checkMap[def.UUID] = newItem
			heap.Push(s.checkHeap, newItem)
			logrus.Infof("Scheduled new check %s", def.UUID)
		}
	}

	// Remove processed/deleted checks
	// We need to be careful removing from heap using index.
	// It's safer to rebuild user list or just mark them as removed?
	// If we simply delete from checkMap, the item is still in heap.
	// We should remove from heap.

	for uuid, item := range s.checkMap {
		if !activeUUIDs[uuid] {
			// Check was deleted or became invalid
			heap.Remove(s.checkHeap, item.Index)
			delete(s.checkMap, uuid)
			logrus.Infof("Descheduled check %s", uuid)
		}
	}

	return nil
}

func (s *Scheduler) getAllChecks(ctx context.Context) ([]models.CheckDefinition, error) {
	// We fetch ALL checks, check definitions (enabled only)
	return s.repo.GetEnabledCheckDefinitions(ctx)
}

// executeCheck runs a single check and updates its status
func executeCheck(repo db.Repository, checkDef models.CheckDefinition, slackAlerter *SlackAlerter) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": checkDef.Project,
		// "group":   checkDef.GroupName,
		"check": checkDef.Name,
		// "type":    checkDef.Type,
		// "uuid":    checkDef.UUID,
	})

	// logger.Debug("Executing check")

	// Create checker instance
	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		return fmt.Errorf("failed to create checker for %s", checkDef.UUID)
	}

	// Run the check
	isHealthy := true
	errMessage := ""
	runTime := time.Now()

	_, err := checker.Run()
	if err != nil {
		isHealthy = false
		errMessage = err.Error()
	}

	// logger.Debugf("Check finished: healthy=%v duration=%v", isHealthy, checkDuration)

	host := ""
	if checkDef.Config != nil {
		host = checkDef.Config.GetTarget()
	}

	// Create status for update
	checkStatus := models.CheckStatus{
		UUID:        checkDef.UUID,
		Project:     checkDef.Project,
		CheckGroup:  checkDef.GroupName,
		CheckName:   checkDef.Name,
		CheckType:   checkDef.Type,
		LastRun:     runTime,
		IsHealthy:   isHealthy,
		Message:     errMessage,
		IsEnabled:   checkDef.Enabled,
		Host:        host,
		Periodicity: checkDef.Duration,
	}

	// Read current health state from DB BEFORE updating, for accurate state
	// transition detection. The in-memory checkDef.IsHealthy may be stale
	// (only refreshed every 10s via Sync), which can cause HandleRecovery
	// to be missed if a check recovers and re-fails between syncs.
	var previouslyHealthy bool
	if prevDef, prevErr := repo.GetCheckDefinitionByUUID(context.Background(), checkDef.UUID); prevErr == nil {
		previouslyHealthy = prevDef.IsHealthy
	} else {
		previouslyHealthy = checkDef.IsHealthy // fallback to in-memory
	}

	// Update status in database
	if err := repo.UpdateCheckStatus(context.Background(), checkStatus); err != nil {
		logger.WithError(err).Error("Failed to update check status")
		return err
	}

	// Detect state transition for Slack recovery using the DB-sourced previous state
	wasUnhealthy := !previouslyHealthy

	// Parse ReAlertInterval for state-transition dedup logic
	var reAlertInterval time.Duration
	if checkDef.ReAlertInterval != "" {
		reAlertInterval, _ = time.ParseDuration(checkDef.ReAlertInterval)
	}

	// Handle alerts if check fails
	if !isHealthy {
		if shouldSendAlert(previouslyHealthy, reAlertInterval, checkStatus) {
			sendAlerts(checkStatus, checkDef, slackAlerter != nil)

			checkStatus.LastAlertSent = runTime
			if err := repo.UpdateCheckStatus(context.Background(), checkStatus); err != nil {
				logger.WithError(err).Error("Failed to update last alert time")
			}
		}

		// Slack App alert (runs alongside existing alerts)
		if slackAlerter != nil {
			// isNewIncident is true when the check transitions from healthy to unhealthy.
			// This tells SendAlert to create a fresh thread instead of replying to any
			// stale unresolved thread left over from a previous incident.
			isNewIncident := previouslyHealthy
			slackAlerter.SendAlert(context.Background(), checkDef, checkStatus, isNewIncident)
		}
	}

	// Handle recovery when check transitions from unhealthy to healthy
	if isHealthy && wasUnhealthy {
		// Send recovery alerts for legacy channels (telegram, slack webhook)
		sendRecoveryAlerts(checkStatus, checkDef, slackAlerter != nil)

		// Handle Slack App recovery (thread resolution)
		if slackAlerter != nil {
			slackAlerter.HandleRecovery(context.Background(), checkDef)
		}
	}

	return nil
}

// shouldSendAlert determines if a DOWN alert should be sent based on state transitions.
// It only fires on healthy→unhealthy transitions, or when ReAlertInterval has elapsed
// for ongoing failures.
func shouldSendAlert(previouslyHealthy bool, reAlertInterval time.Duration, status models.CheckStatus) bool {
	// If check is healthy, no need to alert.
	if status.IsHealthy {
		return false
	}

	// State transition: healthy → unhealthy — always alert.
	if previouslyHealthy {
		return true
	}

	// Ongoing failure (unhealthy → unhealthy):
	// Only re-alert if ReAlertInterval is configured.
	if reAlertInterval <= 0 {
		return false
	}

	// If no previous alert was sent, send one.
	if status.LastAlertSent.IsZero() {
		return true
	}

	// Re-alert if enough time has passed since the last alert.
	return time.Since(status.LastAlertSent) >= reAlertInterval
}

// splitAlertDestination splits an alert destination string by colon
func splitAlertDestination(destination string) []string {
	return strings.Split(destination, ":")
}

// getEffectiveAlertChannels returns the list of alert channels to dispatch to.
// If AlertChannels is set, it returns that. Otherwise, for backward compatibility,
// if AlertType is set and ActorType is "alert", it returns a single-element list
// with the old AlertType value.
func getEffectiveAlertChannels(checkDef models.CheckDefinition) []string {
	if len(checkDef.AlertChannels) > 0 {
		return checkDef.AlertChannels
	}
	// Backward compat: use the old single-channel AlertType
	if checkDef.ActorType == "alert" && checkDef.AlertType != "" {
		return []string{checkDef.AlertType}
	}
	return nil
}

// getEffectiveSeverity returns the severity for the check, defaulting to "critical".
func getEffectiveSeverity(checkDef models.CheckDefinition) string {
	if checkDef.Severity != "" {
		return checkDef.Severity
	}
	return "critical"
}

// sendAlertToChannel dispatches a single alert to one channel type.
// When slackAppActive is true, the "slack" channel is skipped because the SlackAlerter handles it natively.
func sendAlertToChannel(channel string, status models.CheckStatus, checkDef models.CheckDefinition, severity string, slackAppActive bool) {
	logrus.Infof("Sending %s notification (severity=%s) for check %s (%s/%s)",
		channel, severity, status.UUID, status.Project, status.CheckName)

	switch channel {
	case "telegram":
		if checkDef.AlertDestination == "" {
			logrus.Errorf("Telegram alert destination is not configured for check %s", status.UUID)
			return
		}
		parts := splitAlertDestination(checkDef.AlertDestination)
		if len(parts) != 2 {
			logrus.Errorf("Invalid Telegram alert destination format for check %s (expected 'botToken:chatID')", status.UUID)
			return
		}
		botToken := parts[0]
		chatID := parts[1]

		message := fmt.Sprintf("[%s] Check %s (%s/%s) failed: %s", strings.ToUpper(severity), status.CheckName, status.Project, status.CheckGroup, status.Message)
		if err := alerts.SendTelegramAlert(botToken, chatID, message); err != nil {
			logrus.Errorf("Failed to send Telegram alert: %v", err)
		}

	case "slack":
		if slackAppActive {
			// Slack App (SlackAlerter) handles this natively — skip legacy webhook path
			return
		}
		if checkDef.AlertDestination == "" {
			logrus.Errorf("Slack webhook URL is not configured for check %s", status.UUID)
			return
		}
		message := fmt.Sprintf("[%s] Check %s (%s/%s) failed: %s", strings.ToUpper(severity), status.CheckName, status.Project, status.CheckGroup, status.Message)
		if err := alerts.SendSlackAlert(checkDef.AlertDestination, message); err != nil {
			logrus.Errorf("Failed to send Slack alert: %v", err)
		}

	case "opsgenie":
		if checkDef.AlertDestination == "" {
			logrus.Errorf("Opsgenie alert destination is not configured for check %s", status.UUID)
			return
		}
		// AlertDestination format: "apiKey:region" (region is "us" or "eu")
		parts := splitAlertDestination(checkDef.AlertDestination)
		apiKey := parts[0]
		region := "us"
		if len(parts) >= 2 && parts[1] != "" {
			region = parts[1]
		}
		client := &alerts.OpsgenieClient{APIKey: apiKey, Region: region}
		if err := client.Trigger(status.CheckName, status.UUID, status.Message, severity); err != nil {
			logrus.Errorf("Failed to send Opsgenie alert: %v", err)
		}

	default:
		logrus.Warnf("Unknown alert channel: %s for check %s (severity=%s)", channel, status.UUID, severity)
	}
}

// sendAlerts dispatches alerts based on the check definition.
// When slackAppActive is true, the "slack" alertType is skipped because the SlackAlerter handles it natively.
// Supports multi-channel dispatch via AlertChannels with backward compatibility for single-channel AlertType.
func sendAlerts(status models.CheckStatus, checkDef models.CheckDefinition, slackAppActive bool) {
	// Skip if no actor type is defined
	if checkDef.ActorType == "" && len(checkDef.AlertChannels) == 0 {
		return
	}

	// Multi-channel alert dispatch
	channels := getEffectiveAlertChannels(checkDef)
	if len(channels) > 0 {
		severity := getEffectiveSeverity(checkDef)
		for _, channel := range channels {
			sendAlertToChannel(channel, status, checkDef, severity, slackAppActive)
		}
		return
	}

	// Handle actors (log, webhook, etc.) — non-alert actor types
	if checkDef.ActorType != "" && checkDef.ActorType != "alert" {
		actor, err := ActorFactory(checkDef)
		if err != nil {
			logrus.Errorf("Failed to create actor for check %s: %v", status.UUID, err)
			return
		}

		if actor != nil {
			logrus.Infof("Executing %s actor for check %s (%s/%s)",
				checkDef.ActorType, status.UUID, status.Project, status.CheckName)

			if err := actor.Act(status.Message); err != nil {
				logrus.Errorf("Failed to execute action: %v", err)
			}
		}
	}
}

// sendRecoveryAlerts dispatches recovery notifications for legacy alert channels (telegram, slack webhook).
// When slackAppActive is true, the "slack" alertType is skipped because the SlackAlerter handles recovery natively.
func sendRecoveryAlerts(status models.CheckStatus, checkDef models.CheckDefinition, slackAppActive bool) {
	if checkDef.ActorType != "alert" || checkDef.AlertType == "" {
		return
	}

	logrus.Infof("Sending %s recovery notification for check %s (%s/%s)",
		checkDef.AlertType, status.UUID, status.Project, status.CheckName)

	message := fmt.Sprintf("RECOVERY: Check %s (%s/%s) is healthy again", status.CheckName, status.Project, status.CheckGroup)

	switch checkDef.AlertType {
	case "telegram":
		if checkDef.AlertDestination == "" {
			return
		}
		parts := splitAlertDestination(checkDef.AlertDestination)
		if len(parts) != 2 {
			return
		}
		if err := alerts.SendTelegramAlert(parts[0], parts[1], message); err != nil {
			logrus.Errorf("Failed to send Telegram recovery alert: %v", err)
		}

	case "slack":
		if slackAppActive {
			// Slack App handles recovery natively via HandleRecovery
			return
		}
		if checkDef.AlertDestination == "" {
			return
		}
		if err := alerts.SendSlackAlert(checkDef.AlertDestination, message); err != nil {
			logrus.Errorf("Failed to send Slack recovery alert: %v", err)
		}

	case "opsgenie":
		if checkDef.AlertDestination == "" {
			return
		}
		parts := splitAlertDestination(checkDef.AlertDestination)
		apiKey := parts[0]
		region := "us"
		if len(parts) >= 2 && parts[1] != "" {
			region = parts[1]
		}
		client := &alerts.OpsgenieClient{APIKey: apiKey, Region: region}
		if err := client.Resolve(status.UUID); err != nil {
			logrus.Errorf("Failed to send Opsgenie recovery: %v", err)
		}
	}
}

// Helper to find check status (needed by web handlers mostly, but if used here strictly, keeping it)
// It was in scheduler.go but seemingly unused by scheduler itself, only by web/tests?
// Wait, web server probably calls methods in db package.
// If findCheckStatus was exported, I should keep it. It was unexported.
// It was used by nothing in the previous file iteration except itself?
// Re-checking previous file content...
// `findCheckStatus` was unused in `scheduler.go`. It might be used by tests in `scheduler_package`.
// I will keep it just in case tests need it, or remove it if I am sure.
// Tests in `scheduler_test.go` didn't seem to use it (I refactored them).
// I will omit `findCheckStatus` and `getCheckDefinitionsByDuration` if they are not used.

// Exported Accessors?
// No, previously everything was in `RunScheduler`.
