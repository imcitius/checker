package scheduler

import (
	"context"
	"fmt"
	"time"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// secondsFromDayStart returns the number of seconds elapsed since the start of the current day
func secondsFromDayStart() int64 {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return int64(now.Sub(startOfDay).Seconds())
}

// durationToSeconds converts a duration string (e.g., "1m", "1h") to seconds
func durationToSeconds(duration string) (int64, error) {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return 0, err
	}
	return int64(d.Seconds()), nil
}

// RunScheduler starts the health check scheduler.
func RunScheduler(ctx context.Context, cfg *config.Config, mongoDB *db.MongoDB) error {
	logrus.Info("Starting health check scheduler")

	// Create a cancellable context for the ticker
	tickerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a single ticker that runs every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Channel to collect errors
	errChan := make(chan error, 1)

	// Start the ticker goroutine
	go func() {
		logrus.Debug("Starting scheduler ticker")

		for {
			select {
			case <-tickerCtx.Done():
				logrus.Debug("Scheduler ticker shutting down")
				return
			case <-ticker.C:
				// Get current seconds from day start
				currentSeconds := secondsFromDayStart()

				// Get all enabled check definitions
				definitions, err := getAllEnabledChecks(mongoDB)
				if err != nil {
					logrus.Errorf("Failed to get enabled checks: %v", err)
					continue
				}

				// Process each check definition
				for _, checkDef := range definitions {
					// Convert check duration to seconds
					checkSeconds, err := durationToSeconds(checkDef.Duration)
					if err != nil {
						logrus.Warnf("Invalid duration for check %s: %s", checkDef.UUID, checkDef.Duration)
						continue
					}

					// Skip if check period is 0 (invalid)
					if checkSeconds == 0 {
						continue
					}

					if !checkDef.Enabled {
						logrus.Infof("Check %s is disabled", checkDef.UUID)
						continue
					}

					// Check if it's time to run this check
					if currentSeconds%checkSeconds == 0 {
						// Run the check in a separate goroutine
						go func(def models.CheckDefinition) {
							if err := executeCheck(mongoDB, def); err != nil {
								logrus.Errorf("Error executing check %s: %v", def.UUID, err)
							}
						}(checkDef)
					}
				}
			}
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		logrus.Info("Scheduler shutting down due to context cancellation")
		cancel() // Cancel ticker context
		return ctx.Err()
	case err := <-errChan:
		logrus.Errorf("Scheduler shutting down due to error: %v", err)
		cancel() // Cancel ticker context
		return err
	}
}

// getAllEnabledChecks retrieves all enabled check definitions from the database
func getAllEnabledChecks(mongoDB *db.MongoDB) ([]models.CheckDefinition, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"enabled": true,
	}

	var definitions []models.CheckDefinition
	cursor, err := mongoDB.Collection("check_definitions").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query check definitions: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &definitions); err != nil {
		return nil, fmt.Errorf("failed to decode check definitions: %w", err)
	}

	return definitions, nil
}

// executeCheck runs a single check and updates its status
func executeCheck(mongoDB *db.MongoDB, checkDef models.CheckDefinition) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": checkDef.Project,
		"group":   checkDef.GroupName,
		"check":   checkDef.Name,
		"type":    checkDef.Type,
		"uuid":    checkDef.UUID,
	})

	logger.Debug("Executing check")

	// Create checker instance
	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		return fmt.Errorf("failed to create checker for %s", checkDef.UUID)
	}

	// Run the check
	isHealthy := true
	errMessage := ""
	runTime := time.Now()

	checkDuration, err := checker.Run()
	if err != nil {
		isHealthy = false
		errMessage = err.Error()
	}

	logger.WithFields(logrus.Fields{
		"healthy":     isHealthy,
		"message":     errMessage,
		"duration_ms": checkDuration.Milliseconds(),
	}).Info("Check completed")

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
		Host:        checkDef.Host,
		Periodicity: checkDef.Duration,
	}

	// Update status in database
	if err := upsertCheckStatus(mongoDB, &checkStatus); err != nil {
		logger.WithError(err).Error("Failed to update check status")
		return err
	}

	// Handle alerts if check fails
	if !isHealthy {
		checkDuration, _ := time.ParseDuration(checkDef.Duration)
		if shouldSendAlert(checkDuration, checkStatus) {
			alertStartTime := time.Now()
			sendAlerts(checkStatus, checkDef)
			logger.WithField("alert_duration_ms", time.Since(alertStartTime).Milliseconds()).Info("Alert sent")

			checkStatus.LastAlertSent = runTime
			if err := upsertCheckStatus(mongoDB, &checkStatus); err != nil {
				logger.WithError(err).Error("Failed to update last alert time")
			}
		}
	}

	return nil
}

// getCheckDefinitionsByDuration retrieves all enabled check definitions for a specific duration
func getCheckDefinitionsByDuration(ctx context.Context, mongoDB *db.MongoDB, duration string) ([]models.CheckDefinition, error) {
	checkDefinitionsCollection := "check_definitions"

	// Query for enabled checks with the specified duration
	filter := bson.M{
		"enabled":  true,
		"duration": duration,
	}

	var definitions []models.CheckDefinition

	cursor, err := mongoDB.Collection(checkDefinitionsCollection).Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query check definitions: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &definitions); err != nil {
		return nil, fmt.Errorf("failed to decode check definitions: %w", err)
	}

	return definitions, nil
}

// findCheckStatus looks up the check status in the database.
func findCheckStatus(mongoDB *db.MongoDB, UUID string) (*models.CheckStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"uuid": UUID,
	}

	var checkDef models.CheckDefinition
	err := mongoDB.Collection("check_definitions").FindOne(ctx, filter).Decode(&checkDef)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, fmt.Errorf("error finding check: %w", err)
	}

	// Convert CheckDefinition to CheckStatus
	return &models.CheckStatus{
		ID:            checkDef.ID,
		UUID:          checkDef.UUID,
		Project:       checkDef.Project,
		CheckGroup:    checkDef.GroupName,
		CheckName:     checkDef.Name,
		CheckType:     checkDef.Type,
		LastRun:       checkDef.LastRun,
		IsHealthy:     checkDef.IsHealthy,
		Message:       checkDef.LastMessage,
		IsEnabled:     checkDef.Enabled,
		LastAlertSent: checkDef.LastAlertSent,
		Host:          checkDef.Host,
		Periodicity:   checkDef.Duration,
	}, nil
}

// upsertCheckStatus updates the status fields in the check definition.
func upsertCheckStatus(mongoDB *db.MongoDB, status *models.CheckStatus) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"uuid": status.UUID,
	}

	update := bson.M{
		"$set": bson.M{
			"last_run":        status.LastRun,
			"is_healthy":      status.IsHealthy,
			"last_message":    status.Message,
			"last_alert_sent": status.LastAlertSent,
			"updated_at":      time.Now(),
		},
	}

	opts := options.Update().SetUpsert(false) // Don't upsert as we only update existing checks

	result, err := mongoDB.Collection("check_definitions").UpdateOne(
		ctx,
		filter,
		update,
		opts,
	)
	if err != nil {
		logrus.Errorf("Error updating check status: %v", err)
		return fmt.Errorf("error updating check status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("check with UUID %s not found", status.UUID)
	}

	logrus.Debugf("Update result - Modified: %d", result.ModifiedCount)
	return nil
}

// shouldSendAlert determines if an alert should be sent.
func shouldSendAlert(effectiveDuration time.Duration, status models.CheckStatus) bool {
	// If check is healthy, no need to alert.
	if status.IsHealthy {
		return false
	}
	// If no previous alert was sent, we should alert.
	if status.LastAlertSent.IsZero() {
		return true
	}
	// Check if enough time (as defined by the effective duration) has passed since last alert.
	return time.Since(status.LastAlertSent) > effectiveDuration
}

// sendAlerts dispatches alerts based on the check definition.
func sendAlerts(status models.CheckStatus, checkDef models.CheckDefinition) {
	// Skip if no actor type is defined
	if checkDef.ActorType == "" {
		logrus.Debugf("No actor type defined for check %s, skipping alerts", status.UUID)
		return
	}

	actor, err := ActorFactory(checkDef)
	if err != nil {
		logrus.Errorf("Failed to create actor for check %s: %v", status.UUID, err)
		return
	}

	if actor != nil {
		// Implement alerting logic here
		logrus.Infof("Sending %s alert for check %s (%s/%s)",
			checkDef.ActorType, status.UUID, status.Project, status.CheckName)
	}
}

// mergeHeaders converts a slice of header maps ([]map[string]string) into a single map[string]string.
func mergeHeaders(headersSlice []map[string]string) map[string]string {
	m := make(map[string]string)
	for _, hm := range headersSlice {
		for k, v := range hm {
			m[k] = v
		}
	}
	return m
}
