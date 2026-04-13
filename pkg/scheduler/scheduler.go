// SPDX-License-Identifier: BUSL-1.1

package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/imcitius/checker/pkg/alerts"
	"github.com/imcitius/checker/pkg/config"
	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
	checkersentry "github.com/imcitius/checker/internal/sentry"

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
	appAlerters     []AppAlerter
	triggerCh       chan string // buffered channel for immediate check triggers
}

// NewScheduler creates a new scheduler instance
func NewScheduler(repo db.Repository, appAlerters []AppAlerter, consensusRegion string) *Scheduler {
	h := &CheckHeap{}
	heap.Init(h)
	return &Scheduler{
		workerPool:  NewWorkerPool(DefaultWorkerPoolSize, repo, appAlerters, consensusRegion),
		checkHeap:   h,
		checkMap:    make(map[string]*CheckItem),
		repo:        repo,
		appAlerters: appAlerters,
		triggerCh:   make(chan string, 1),
	}
}

// TriggerCheck queues an immediate execution of the check with the given UUID.
// This is a non-blocking operation: if a trigger is already pending, this is a no-op.
func (s *Scheduler) TriggerCheck(uuid string) {
	select {
	case s.triggerCh <- uuid:
		logrus.Infof("Queued immediate trigger for check %s", uuid)
	default:
		logrus.Debugf("Trigger channel full, skipping trigger for check %s", uuid)
	}
}

// RunScheduler starts the health check scheduler.
// If s is nil, a new Scheduler is created internally (backward-compatible).
// Pass an existing *Scheduler created via NewScheduler when you need a reference
// to call TriggerCheck from outside (e.g. from HTTP handlers).
func RunScheduler(ctx context.Context, cfg *config.Config, repo db.Repository, appAlerters []AppAlerter, s *Scheduler) error {
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

	// Load checker-wide defaults
	if defaults, err := repo.GetCheckDefaults(ctx); err == nil {
		if len(defaults.AlertChannels) > 0 {
			defaultAlertChannels = defaults.AlertChannels
			logrus.Infof("Default alert channels: %v", defaultAlertChannels)
		}
		if defaults.ReAlertInterval != "" {
			if d, parseErr := time.ParseDuration(defaults.ReAlertInterval); parseErr == nil && d > 0 {
				defaultReAlertInterval = d
				logrus.Infof("Default re-alert interval: %s", d)
			}
		}
	}

	if s == nil {
		s = NewScheduler(repo, appAlerters, cfg.Consensus.Region)
	}

	// Start worker pool
	s.workerPool.Start()
	defer s.workerPool.Stop()

	// Start consensus sweeper in multi-region mode
	if cfg.IsMultiRegion() {
		evalInterval := parseDuration(cfg.Consensus.EvaluationInterval)
		if evalInterval <= 0 {
			evalInterval = 10 * time.Second
		}
		timeout := parseDuration(cfg.Consensus.Timeout)
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		go RunConsensusSweeper(ctx, cfg.Consensus.Region, cfg.Consensus.MinRegions, evalInterval, timeout, repo, appAlerters)
	}

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

		case uuid := <-s.triggerCh:
			timer.Stop()
			logrus.Infof("Immediate trigger: syncing and executing check %s", uuid)
			// Sync to pick up the newly created check (adds it to heap with NextRun=now)
			if err := s.Sync(ctx); err != nil {
				logrus.Errorf("Sync failed during trigger for check %s: %v", uuid, err)
			}
			// Set the triggered check's NextRun to now so it is processed first
			s.lock.Lock()
			if item, exists := s.checkMap[uuid]; exists {
				item.NextRun = time.Now()
				heap.Init(s.checkHeap)
			}
			s.lock.Unlock()
			// Dispatch the triggered check immediately
			s.processNextCheck()

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

	// Submit to worker pool if enabled.
	// Note: maintenance mode does NOT prevent execution — checks still run so
	// that recovery is detected when maintenance ends. Alerts are suppressed
	// in processCheckResult instead.
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Reload default alert channels from settings
	if defaults, err := s.repo.GetCheckDefaults(ctx); err == nil {
		if len(defaults.AlertChannels) > 0 {
			defaultAlertChannels = defaults.AlertChannels
			logrus.Debugf("Sync: reloaded default alert channels: %v", defaultAlertChannels)
		}
		if defaults.ReAlertInterval != "" {
			if d, parseErr := time.ParseDuration(defaults.ReAlertInterval); parseErr == nil && d > 0 {
				defaultReAlertInterval = d
			}
		}
	} else {
		logrus.WithError(err).Warn("Sync: failed to reload check defaults")
	}

	defs, err := s.getAllChecks(ctx)
	if err != nil {
		return fmt.Errorf("fetching check definitions: %w", err)
	}

	logrus.Debugf("Sync: loaded %d enabled check definitions from DB", len(defs))

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

	removed := 0
	for uuid, item := range s.checkMap {
		if !activeUUIDs[uuid] {
			// Check was deleted or became invalid
			heap.Remove(s.checkHeap, item.Index)
			delete(s.checkMap, uuid)
			logrus.Infof("Descheduled check %s", uuid)
			removed++
		}
	}

	logrus.Debugf("Sync complete: %d checks in heap, %d removed", len(s.checkMap), removed)

	return nil
}

func (s *Scheduler) getAllChecks(ctx context.Context) ([]models.CheckDefinition, error) {
	// We fetch ALL checks, check definitions (enabled only)
	return s.repo.GetEnabledCheckDefinitions(ctx)
}

// executeCheck runs a single check and updates its status.
// When consensusRegion is non-empty, it writes to check_results and returns without alerting.
func executeCheck(repo db.Repository, checkDef models.CheckDefinition, appAlerters []AppAlerter, consensusRegion string) error {
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

	// Multi-region mode: write result to check_results table, skip alerting.
	// The consensus sweeper will evaluate and alert later.
	if consensusRegion != "" {
		dur := parseDuration(checkDef.Duration)
		if dur <= 0 {
			dur = 30 * time.Second
		}
		cycleKey := runTime.Truncate(dur)
		result := models.CheckResult{
			CheckUUID: checkDef.UUID,
			Region:    consensusRegion,
			IsHealthy: isHealthy,
			Message:   errMessage,
			CreatedAt: runTime,
			CycleKey:  cycleKey,
		}
		if err := repo.InsertCheckResult(context.Background(), result); err != nil {
			logger.WithError(err).Error("Failed to insert check result")
			checkersentry.CaptureError(err, map[string]string{
				"check.uuid": checkDef.UUID, "check.name": checkDef.Name, "check.type": checkDef.Type, "op": "insert_check_result",
			})
			return fmt.Errorf("insert check result: %w", err)
		}
		return nil
	}

	// Single-instance mode: update status and alert directly.
	processCheckResult(repo, checkDef, checkStatus, isHealthy, runTime, appAlerters, logger)
	return nil
}

// processCheckResult handles status update, alert decisions, and alert dispatch.
// Used by both the legacy single-instance path and the consensus sweeper.
func processCheckResult(repo db.Repository, checkDef models.CheckDefinition, checkStatus models.CheckStatus, isHealthy bool, runTime time.Time, appAlerters []AppAlerter, logger *logrus.Entry) {
	// Read current health state from DB BEFORE updating, for accurate state
	// transition detection. The in-memory checkDef.IsHealthy may be stale
	// (only refreshed every 10s via Sync), which can cause HandleRecovery
	// to be missed if a check recovers and re-fails between syncs.
	var previouslyHealthy bool
	if prevDef, prevErr := repo.GetCheckDefinitionByUUID(context.Background(), checkDef.UUID); prevErr == nil {
		previouslyHealthy = prevDef.IsHealthy
		// Use DB-loaded AlertChannels which includes tenant defaults
		// (the adapter populates them via GetCheckDefaults when empty).
		// The incoming checkDef may have empty AlertChannels from the
		// consensus sweeper or edge result path.
		if len(checkDef.AlertChannels) == 0 && len(prevDef.AlertChannels) > 0 {
			checkDef.AlertChannels = prevDef.AlertChannels
		}
	} else {
		previouslyHealthy = checkDef.IsHealthy // fallback to in-memory
	}

	// Update status in database
	if err := repo.UpdateCheckStatus(context.Background(), checkStatus); err != nil {
		logger.WithError(err).Error("Failed to update check status")
		checkersentry.CaptureError(err, map[string]string{
			"check.uuid": checkDef.UUID, "check.name": checkDef.Name, "op": "update_check_status",
		})
		return
	}

	// Detect state transition for Slack recovery using the DB-sourced previous state
	wasUnhealthy := !previouslyHealthy

	// Check hierarchical maintenance mode: if any level (project, group, or
	// check itself) is in maintenance, suppress all alerts but keep executing.
	inMaintenance := checkDef.MaintenanceUntil != nil && checkDef.MaintenanceUntil.After(time.Now())
	if !inMaintenance {
		// Check project/group-level maintenance
		projSettings, _ := repo.GetProjectSettings(context.Background(), checkDef.Project)
		if projSettings != nil && projSettings.IsInMaintenance() {
			inMaintenance = true
			logger.WithField("project", checkDef.Project).Info("processCheckResult: project in maintenance — suppressing alerts")
		}
		if !inMaintenance {
			grpSettings, _ := repo.GetGroupSettings(context.Background(), checkDef.Project, checkDef.GroupName)
			if grpSettings != nil && grpSettings.IsInMaintenance() {
				inMaintenance = true
				logger.WithFields(logrus.Fields{"project": checkDef.Project, "group": checkDef.GroupName}).Info("processCheckResult: group in maintenance — suppressing alerts")
			}
		}
	}
	if inMaintenance {
		logger.Info("processCheckResult: check/project/group in maintenance — alerts suppressed, status updated")
		return
	}

	// Parse ReAlertInterval for state-transition dedup logic.
	// Falls back to the system default (1h) when neither the check nor
	// check defaults specify a value. This prevents alert spam from
	// ongoing failures while ensuring re-alerts eventually fire.
	var reAlertInterval time.Duration
	if checkDef.ReAlertInterval != "" {
		reAlertInterval, _ = time.ParseDuration(checkDef.ReAlertInterval)
	}
	if reAlertInterval <= 0 && defaultReAlertInterval > 0 {
		reAlertInterval = defaultReAlertInterval
	}

	// Handle alerts if check fails
	if !isHealthy {
		effectiveChannels := getEffectiveAlertChannels(checkDef)
		logger.WithFields(logrus.Fields{
			"previously_healthy": previouslyHealthy,
			"re_alert_interval":  reAlertInterval,
			"alert_channels":     effectiveChannels,
			"last_alert_sent":    checkStatus.LastAlertSent,
		}).Info("processCheckResult: check unhealthy, evaluating alert dispatch")

		if shouldSendAlert(previouslyHealthy, reAlertInterval, checkStatus) {
			logger.Info("processCheckResult: shouldSendAlert=true, dispatching standard alerts")
			sendAlerts(repo, checkStatus, checkDef, appAlerters)

			checkStatus.LastAlertSent = runTime
			if err := repo.UpdateCheckStatus(context.Background(), checkStatus); err != nil {
				logger.WithError(err).Error("Failed to update last alert time")
			}
		} else if configChangedSinceLastAlert(checkDef, checkStatus) {
			logger.Info("processCheckResult: config changed since last alert, dispatching")
			sendAlerts(repo, checkStatus, checkDef, appAlerters)

			checkStatus.LastAlertSent = runTime
			if err := repo.UpdateCheckStatus(context.Background(), checkStatus); err != nil {
				logger.WithError(err).Error("Failed to update last alert time")
			}
		} else {
			logger.Info("processCheckResult: shouldSendAlert=false, skipping standard alerts")
		}

		// App alerters — only fire if the check has a matching channel type selected
		// and the check is not silenced.
		isNewIncident := previouslyHealthy
		checkSilenced, silErr := repo.IsCheckSilenced(context.Background(), checkDef.UUID, checkDef.Project)
		if silErr != nil {
			logger.WithError(silErr).Warn("processCheckResult: failed to check silence for app alerters")
			// Don't suppress on error
		}
		if checkSilenced {
			logger.Info("processCheckResult: check is silenced — skipping app alerters")
		} else if len(effectiveChannels) == 0 {
			logger.Warn("processCheckResult: no alert channels — skipping app alerters")
		} else {
			selectedTypes := resolveSelectedChannelTypes(repo, checkDef)
			logger.WithFields(logrus.Fields{
				"selected_types": selectedTypes,
				"app_alerters":   len(appAlerters),
				"is_new_incident": isNewIncident,
			}).Info("processCheckResult: evaluating app alerters")
			for _, aa := range appAlerters {
				if shouldAppAlerterFire(aa, selectedTypes) {
					logger.Infof("processCheckResult: firing app alerter (owned types: %v)", aa.OwnedTypes())
					aa.SendAlert(context.Background(), checkDef, checkStatus, isNewIncident)
				} else {
					logger.Debugf("processCheckResult: skipping app alerter (owned types: %v, not in selected)", aa.OwnedTypes())
				}
			}
		}

		// Process escalation policy (if assigned)
		processEscalation(repo, checkDef, checkStatus, appAlerters)

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
		// Send recovery alerts for legacy channels
		sendRecoveryAlerts(repo, checkStatus, checkDef, appAlerters)

		// Handle app alerter recovery — only fire if the check has a matching channel type selected
		// and the check is not silenced.
		recoverySilenced, recSilErr := repo.IsCheckSilenced(context.Background(), checkDef.UUID, checkDef.Project)
		if recSilErr != nil {
			logrus.WithError(recSilErr).Warn("processCheckResult: failed to check silence for recovery app alerters")
		}
		if recoverySilenced {
			// Check is silenced — skip recovery app alerters
		} else {
			recoveryChannels := getEffectiveAlertChannels(checkDef)
			if len(recoveryChannels) == 0 {
				// No channels configured and no default — skip app alerters
			} else {
				recoveryTypes := resolveSelectedChannelTypes(repo, checkDef)
				for _, aa := range appAlerters {
					if shouldAppAlerterFire(aa, recoveryTypes) {
						aa.HandleRecovery(context.Background(), checkDef)
					}
				}
			}
		}

		// Clear escalation notifications on recovery (reset for next incident)
		clearEscalationNotifications(repo, checkDef.UUID)
	}
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

// configChangedSinceLastAlert returns true when the check definition was
// modified after the last alert was sent.  This catches the case where a user
// adds new alert channels while a check is already in a failing state —
// without this, the new channels would never fire until the check recovers
// and fails again.
func configChangedSinceLastAlert(checkDef models.CheckDefinition, status models.CheckStatus) bool {
	if status.LastAlertSent.IsZero() {
		return false // no previous alert — handled by shouldSendAlert
	}
	return !checkDef.UpdatedAt.IsZero() && checkDef.UpdatedAt.After(status.LastAlertSent)
}

// defaultAlertChannels holds the checker-wide default alert channels, loaded at scheduler startup.
var defaultAlertChannels []string

// defaultReAlertInterval is the fallback re-alert interval used when neither
// the check nor check defaults specify one. Prevents alert spam (only first
// alert fires) while ensuring re-alerts for ongoing failures. Default: 1h.
var defaultReAlertInterval = time.Hour

// getEffectiveAlertChannels returns the list of alert channels to dispatch to.
// Falls back to checker-wide default alert channels when the check has none configured.
func getEffectiveAlertChannels(checkDef models.CheckDefinition) []string {
	if len(checkDef.AlertChannels) > 0 {
		return checkDef.AlertChannels
	}
	if len(defaultAlertChannels) > 0 {
		logrus.Debugf("Check %q (%s) has no alert_channels, falling back to system defaults: %v",
			checkDef.Name, checkDef.UUID, defaultAlertChannels)
	} else {
		logrus.Debugf("Check %q (%s) has no alert_channels and no system defaults configured",
			checkDef.Name, checkDef.UUID)
	}
	return defaultAlertChannels
}

// resolveSelectedChannelTypes returns the set of channel types selected for a check.
func resolveSelectedChannelTypes(repo db.Repository, checkDef models.CheckDefinition) map[string]bool {
	channels := getEffectiveAlertChannels(checkDef)
	types := make(map[string]bool)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, name := range channels {
		ch, err := repo.GetAlertChannelByName(ctx, name)
		if err != nil {
			continue
		}
		types[ch.Type] = true
	}
	return types
}

// shouldAppAlerterFire returns true if any selected channel type matches the AppAlerter's owned types.
func shouldAppAlerterFire(aa AppAlerter, selectedTypes map[string]bool) bool {
	for _, t := range aa.OwnedTypes() {
		if selectedTypes[t] {
			return true
		}
	}
	return false
}

// getEffectiveSeverity returns the severity for the check, defaulting to "critical".
func getEffectiveSeverity(checkDef models.CheckDefinition) string {
	if checkDef.Severity != "" {
		return checkDef.Severity
	}
	return "critical"
}

// AlerterResolver is an optional interface that db.Repository implementations can
// satisfy to override how alerters are created. This allows adapters (e.g. the
// cloud multi-tenant adapter) to wrap alerters with tracking or other middleware
// without remapping channel types in the data layer.
type AlerterResolver interface {
	ResolveAlerter(ctx context.Context, channelName string) (alerts.Alerter, error)
}

// resolveAlerter resolves an Alerter for the given channel name from the DB alert_channels table.
// If the repo implements AlerterResolver, it delegates to that for custom alerter creation.
func resolveAlerter(repo db.Repository, channelName string) (alerts.Alerter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Allow the repo to override alerter creation (e.g. to wrap with tracking).
	if resolver, ok := repo.(AlerterResolver); ok {
		return resolver.ResolveAlerter(ctx, channelName)
	}

	ch, err := repo.GetAlertChannelByName(ctx, channelName)
	if err != nil {
		return nil, fmt.Errorf("alert channel %q not found: %w", channelName, err)
	}
	return alerts.NewAlerter(ch.Type, ch.Config)
}

// sendAlerts dispatches alerts based on the check definition using the Alerter registry.
// Alerters whose Type() matches any type owned by an active AppAlerter are skipped
// to prevent duplicate alerts.
func sendAlerts(repo db.Repository, status models.CheckStatus, checkDef models.CheckDefinition, appAlerters []AppAlerter) {
	channels := getEffectiveAlertChannels(checkDef)
	if len(channels) == 0 {
		return
	}

	ownedTypes := buildOwnedTypeSet(appAlerters)
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
		// Check per-channel silence
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		silenced, err := repo.IsChannelSilenced(ctx, status.UUID, status.Project, channel)
		cancel()
		if err != nil {
			logrus.Errorf("Failed to check silence for channel %q check %s: %v", channel, status.UUID, err)
			// Continue — don't suppress alerts on error
		}
		if silenced {
			logrus.Infof("Channel %q silenced for check %s, skipping", channel, status.UUID)
			continue
		}

		alerter, err := resolveAlerter(repo, channel)
		if err != nil {
			logrus.Errorf("Failed to resolve alerter for channel %q check %s: %v", channel, status.UUID, err)
			continue
		}
		// Skip channels whose type is owned by an active AppAlerter
		if ownedTypes[alerter.Type()] {
			continue
		}
		logrus.Infof("Sending %s notification (severity=%s) for check %s (%s/%s)",
			alerter.Type(), severity, status.UUID, status.Project, status.CheckName)
		if err := alerter.SendAlert(payload); err != nil {
			logrus.Errorf("Failed to send %s alert for check %s: %v", alerter.Type(), status.UUID, err)
			checkersentry.CaptureError(err, map[string]string{
				"check.uuid": status.UUID, "alert.channel": channel, "alert.type": alerter.Type(), "op": "send_alert",
			})
		}
	}
}

// sendRecoveryAlerts dispatches recovery notifications using the Alerter registry.
// Alerters whose Type() matches any type owned by an active AppAlerter are skipped
// to prevent duplicate recovery notifications.
func sendRecoveryAlerts(repo db.Repository, status models.CheckStatus, checkDef models.CheckDefinition, appAlerters []AppAlerter) {
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

	ownedTypes := buildOwnedTypeSet(appAlerters)

	for _, channel := range channels {
		// Check per-channel silence
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		silenced, err := repo.IsChannelSilenced(ctx, status.UUID, status.Project, channel)
		cancel()
		if err != nil {
			logrus.Errorf("Failed to check silence for channel %q check %s: %v", channel, status.UUID, err)
			// Continue — don't suppress recovery alerts on error
		}
		if silenced {
			logrus.Infof("Channel %q silenced for check %s, skipping recovery", channel, status.UUID)
			continue
		}

		alerter, err := resolveAlerter(repo, channel)
		if err != nil {
			logrus.Errorf("Failed to resolve alerter for channel %q check %s: %v", channel, status.UUID, err)
			continue
		}
		// Skip channels whose type is owned by an active AppAlerter
		if ownedTypes[alerter.Type()] {
			continue
		}
		logrus.Infof("Sending %s recovery notification for check %s (%s/%s)",
			alerter.Type(), status.UUID, status.Project, status.CheckName)
		if err := alerter.SendRecovery(payload); err != nil {
			logrus.Errorf("Failed to send %s recovery for check %s: %v", alerter.Type(), status.UUID, err)
		}
	}
}
