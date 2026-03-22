package scheduler

import (
	"container/heap"
	"context"
	"fmt"
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

// emailAlertConfig holds the global email configuration, set during scheduler startup.
var emailAlertConfig *alerts.EmailConfig

// getEmailConfig returns the global email alert configuration, or nil if not configured.
func getEmailConfig() *alerts.EmailConfig {
	return emailAlertConfig
}

// Scheduler manages the lifecycle of health checks
type Scheduler struct {
	workerPool      *WorkerPool
	checkHeap       *CheckHeap
	checkMap        map[string]*CheckItem // Map UUID -> CheckItem
	lock            sync.Mutex
	repo            db.Repository
	slackAlerter    *SlackAlerter
	telegramAlerter *TelegramAppAlerter
}

// NewScheduler creates a new scheduler instance
func NewScheduler(repo db.Repository, slackAlerter *SlackAlerter, telegramAlerter *TelegramAppAlerter) *Scheduler {
	h := &CheckHeap{}
	heap.Init(h)
	return &Scheduler{
		workerPool:      NewWorkerPool(DefaultWorkerPoolSize, repo, slackAlerter, telegramAlerter),
		checkHeap:       h,
		checkMap:        make(map[string]*CheckItem),
		repo:            repo,
		slackAlerter:    slackAlerter,
		telegramAlerter: telegramAlerter,
	}
}

// RunScheduler starts the health check scheduler.
func RunScheduler(ctx context.Context, cfg *config.Config, repo db.Repository, slackAlerter *SlackAlerter, telegramAlerter *TelegramAppAlerter) error {
	logrus.Info("Starting event-driven health check scheduler")

	// Initialize email alert configuration from config
	if emailCfg, ok := cfg.Alerts["email"]; ok && emailCfg.SMTPHost != "" {
		emailAlertConfig = &alerts.EmailConfig{
			SMTPHost:     emailCfg.SMTPHost,
			SMTPPort:     emailCfg.SMTPPort,
			SMTPUser:     emailCfg.SMTPUser,
			SMTPPassword: emailCfg.SMTPPassword,
			From:         emailCfg.From,
			To:           emailCfg.To,
			UseTLS:       emailCfg.UseTLS,
		}
		logrus.Info("Email alerter configured")
	}

	s := NewScheduler(repo, slackAlerter, telegramAlerter)

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

// runCheckWithRetries executes runFn and, if it fails and retryCount > 0, retries
// up to retryCount additional times with retryInterval between attempts.
// Returns nil if any attempt succeeds, or the last error if all attempts fail.
func runCheckWithRetries(runFn func() (time.Duration, error), retryCount int, retryInterval string, logger *logrus.Entry) error {
	_, err := runFn()
	if err == nil {
		return nil
	}

	if retryCount <= 0 {
		return err
	}

	retryWait := parseDuration(retryInterval)
	if retryWait <= 0 {
		retryWait = 5 * time.Second // default retry interval
	}

	lastErr := err
	for attempt := 1; attempt <= retryCount; attempt++ {
		logger.Infof("Check failed, retrying (%d/%d) after %s: %v", attempt, retryCount, retryWait, lastErr)
		time.Sleep(retryWait)

		_, lastErr = runFn()
		if lastErr == nil {
			return nil // success on retry
		}
	}

	return lastErr
}

// parseDuration converts a duration string (e.g. "10s", "30s", "5m") to time.Duration.
// Sub-minute intervals (e.g. "10s", "30s") are supported but will increase DB write
// frequency proportionally — use with care in large deployments.
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

	// Submit to worker pool if enabled and not in maintenance window
	if item.CheckDef.Enabled {
		if item.CheckDef.MaintenanceUntil != nil && time.Now().Before(*item.CheckDef.MaintenanceUntil) {
			// Skip check during maintenance window — do not execute or alert
			logrus.Debugf("Skipping check %s — in maintenance until %s", item.CheckDef.UUID, item.CheckDef.MaintenanceUntil.Format(time.RFC3339))
		} else {
			s.workerPool.Submit(item.CheckDef)
		}
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
		return fmt.Errorf("fetching check definitions: %w", err)
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
func executeCheck(repo db.Repository, checkDef models.CheckDefinition, slackAlerter *SlackAlerter, telegramAlerter *TelegramAppAlerter) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": checkDef.Project,
		// "group":   checkDef.GroupName,
		"check": checkDef.Name,
		// "type":    checkDef.Type,
		// "uuid":    checkDef.UUID,
	})

	// logger.Debug("Executing check")

	// Run the check with retry logic
	isHealthy := true
	errMessage := ""
	runTime := time.Now()

	runFn := func() (time.Duration, error) {
		c := CheckerFactory(checkDef, logger)
		if c == nil {
			return 0, fmt.Errorf("failed to create checker for %s", checkDef.UUID)
		}
		return c.Run()
	}

	err := runCheckWithRetries(runFn, checkDef.RetryCount, checkDef.RetryInterval, logger)
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
		UUID:          checkDef.UUID,
		Project:       checkDef.Project,
		CheckGroup:    checkDef.GroupName,
		CheckName:     checkDef.Name,
		CheckType:     checkDef.Type,
		LastRun:       runTime,
		IsHealthy:     isHealthy,
		Message:       errMessage,
		IsEnabled:     checkDef.Enabled,
		Host:          host,
		Periodicity:   checkDef.Duration,
		LastAlertSent: checkDef.LastAlertSent,
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
		return fmt.Errorf("update check status: %w", err)
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
			sendAlerts(repo, checkStatus, checkDef, slackAlerter != nil, telegramAlerter != nil)

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

		// Telegram App alert (runs alongside existing alerts)
		if telegramAlerter != nil {
			isNewIncident := previouslyHealthy
			telegramAlerter.SendAlert(context.Background(), checkDef, checkStatus, isNewIncident)
		}

		// Process escalation policy (if assigned)
		processEscalation(repo, checkDef, checkStatus, slackAlerter)

		// Execute actors (webhook, log) — separate from alert dispatch
		if checkDef.ActorType != "" && checkDef.ActorType != "alert" {
			actor, err := ActorFactory(checkDef)
			if err != nil {
				logrus.Errorf("Failed to create actor for check %s: %v", checkStatus.UUID, err)
			} else if actor != nil {
				logrus.Infof("Executing %s actor for check %s (%s/%s)",
					checkDef.ActorType, checkStatus.UUID, checkStatus.Project, checkStatus.CheckName)
				if err := actor.Act(checkStatus.Message); err != nil {
					logrus.Errorf("Failed to execute action: %v", err)
				}
			}
		}
	}

	// Handle recovery when check transitions from unhealthy to healthy
	if isHealthy && wasUnhealthy {
		// Send recovery alerts for legacy channels (telegram, slack webhook)
		sendRecoveryAlerts(repo, checkStatus, checkDef, slackAlerter != nil, telegramAlerter != nil)

		// Handle Slack App recovery (thread resolution)
		if slackAlerter != nil {
			slackAlerter.HandleRecovery(context.Background(), checkDef)
		}

		// Handle Telegram App recovery (thread resolution)
		if telegramAlerter != nil {
			telegramAlerter.HandleRecovery(context.Background(), checkDef)
		}

		// Clear escalation notifications on recovery (reset for next incident)
		clearEscalationNotifications(repo, checkDef.UUID)
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

// getEffectiveAlertChannels returns the list of alert channels to dispatch to.
func getEffectiveAlertChannels(checkDef models.CheckDefinition) []string {
	return checkDef.AlertChannels
}

// getEffectiveSeverity returns the severity for the check, defaulting to "critical".
func getEffectiveSeverity(checkDef models.CheckDefinition) string {
	if checkDef.Severity != "" {
		return checkDef.Severity
	}
	return "critical"
}

// resolveAlerter resolves an Alerter for the given channel name from the DB alert_channels table.
func resolveAlerter(repo db.Repository, channelName string) (alerts.Alerter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := repo.GetAlertChannelByName(ctx, channelName)
	if err != nil {
		return nil, fmt.Errorf("alert channel %q not found: %w", channelName, err)
	}
	return alerts.NewAlerter(ch.Type, ch.Config)
}

// sendAlerts dispatches alerts based on the check definition using the Alerter registry.
// When slackAppActive is true, alerters with Type() == "slack" are skipped because the SlackAlerter handles it natively.
// When telegramAppActive is true, alerters with Type() == "telegram" are skipped because the TelegramAppAlerter handles it natively.
func sendAlerts(repo db.Repository, status models.CheckStatus, checkDef models.CheckDefinition, slackAppActive bool, telegramAppActive bool) {
	channels := getEffectiveAlertChannels(checkDef)
	if len(channels) == 0 {
		return
	}

	severity := getEffectiveSeverity(checkDef)
	payload := alerts.AlertPayload{
		CheckName:  status.CheckName,
		CheckUUID:  status.UUID,
		Project:    status.Project,
		CheckGroup: status.CheckGroup,
		CheckType:  status.CheckType,
		Message:    status.Message,
		Severity:   severity,
		Timestamp:  status.LastRun,
	}
	for _, channel := range channels {
		alerter, err := resolveAlerter(repo, channel)
		if err != nil {
			logrus.Errorf("Failed to resolve alerter for channel %q check %s: %v", channel, status.UUID, err)
			continue
		}
		// Slack App bypass: check resolved type, not name
		if alerter.Type() == "slack" && slackAppActive {
			continue
		}
		// Telegram App bypass: check resolved type, not name
		if alerter.Type() == "telegram" && telegramAppActive {
			continue
		}
		logrus.Infof("Sending %s notification (severity=%s) for check %s (%s/%s)",
			alerter.Type(), severity, status.UUID, status.Project, status.CheckName)
		if err := alerter.SendAlert(payload); err != nil {
			logrus.Errorf("Failed to send %s alert for check %s: %v", alerter.Type(), status.UUID, err)
		}
	}
}

// sendRecoveryAlerts dispatches recovery notifications using the Alerter registry.
// When slackAppActive is true, alerters with Type() == "slack" are skipped because the SlackAlerter handles recovery natively.
// When telegramAppActive is true, alerters with Type() == "telegram" are skipped because the TelegramAppAlerter handles recovery natively.
func sendRecoveryAlerts(repo db.Repository, status models.CheckStatus, checkDef models.CheckDefinition, slackAppActive bool, telegramAppActive bool) {
	channels := getEffectiveAlertChannels(checkDef)
	if len(channels) == 0 {
		return
	}

	payload := alerts.RecoveryPayload{
		CheckName:  status.CheckName,
		CheckUUID:  status.UUID,
		Project:    status.Project,
		CheckGroup: status.CheckGroup,
		CheckType:  status.CheckType,
		Timestamp:  status.LastRun,
	}

	for _, channel := range channels {
		alerter, err := resolveAlerter(repo, channel)
		if err != nil {
			logrus.Errorf("Failed to resolve alerter for channel %q check %s: %v", channel, status.UUID, err)
			continue
		}
		// Slack App bypass: check resolved type, not name
		if alerter.Type() == "slack" && slackAppActive {
			continue
		}
		// Telegram App bypass: check resolved type, not name
		if alerter.Type() == "telegram" && telegramAppActive {
			continue
		}
		logrus.Infof("Sending %s recovery notification for check %s (%s/%s)",
			alerter.Type(), status.UUID, status.Project, status.CheckName)
		if err := alerter.SendRecovery(payload); err != nil {
			logrus.Errorf("Failed to send %s recovery for check %s: %v", alerter.Type(), status.UUID, err)
		}
	}
}
