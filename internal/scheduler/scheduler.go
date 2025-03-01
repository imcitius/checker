package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RunScheduler starts the health check scheduler.
func RunScheduler(cfg *config.Config, mongoDB *db.MongoDB) {
	logrus.Info("Starting health check scheduler")
	counter := 1
	wg := sync.WaitGroup{}

	for _, ticker := range cfg.Tickers {
		wg.Add(1)
		if counter == len(cfg.Tickers) /*+len(maintenances.Tickers)*/ {
			runProjectTicker(ticker, &wg, cfg, mongoDB)
		} else {
			go runProjectTicker(ticker, &wg, cfg, mongoDB)
			counter++
		}
	}
}

func runProjectTicker(ticker config.TickerWithDuration, wg *sync.WaitGroup, cfg *config.Config, mongoDB *db.MongoDB) {
	defer ticker.Ticker.Stop()
	defer wg.Done()
	logrus.Debugf("Starting project ticker with duration %s", ticker.Duration)

	for range ticker.Ticker.C {
		performAllChecks(ticker.Duration, cfg, mongoDB)
	}
}

// performAllChecks iterates over all projects and runs each check if check duration equal to passed on func call
func performAllChecks(duration string, cfg *config.Config, mongoDB *db.MongoDB) {
	// logrus.Debugf("Performing all checks with duration: %s", duration)
	totalChecks := 0
	successfulChecks := 0

	for projectName, project := range cfg.Projects {
		// logrus.WithField("project", projectName).Debugf("Processing project %s checks", projectName)
		// Iterate over each health check group
		for groupName, group := range project.HealthChecks {
			// Iterate over each check in the group
			for checkName, checkData := range group.Checks {
				totalChecks++

				// Compute effective duration with precedence:
				//   defaults < project < group < check (if check.Parameters.Duration is set)
				effectiveDuration := cfg.Defaults.Duration
				if project.Parameters.Duration != 0 {
					effectiveDuration = project.Parameters.Duration
				}
				if group.Parameters.Duration != 0 {
					effectiveDuration = group.Parameters.Duration
				}
				if checkData.Parameters.Duration != 0 {
					effectiveDuration = checkData.Parameters.Duration
				}

				if effectiveDuration.String() == duration {
					// Retrieve current status from DB.
					logger := logrus.WithFields(logrus.Fields{
						"project": projectName,
						"group":   groupName,
						"check":   checkName,
						"type":    checkData.Type,
					})
					checkData.Logger = logger
					currentStatus, err := findCheckStatus(mongoDB, checkData.UUID)
					if err != nil && err != mongo.ErrNoDocuments {
						logger.WithError(err).Error("Failed to fetch check status")
						continue
					}

					// If the check ran recently then skip it.
					if currentStatus != nil {
						if !currentStatus.LastRun.IsZero() {
							if time.Since(currentStatus.LastRun) < effectiveDuration {
								remaining := effectiveDuration - time.Since(currentStatus.LastRun)
								logger.WithField("next_run_in", remaining).Debugf("Skipping check, not due yet")
								continue
							}
						}
						if !currentStatus.IsEnabled {
							//logger.Debug("Skipping check, it is disabled")
							continue
						}
					}

					// Create checker instance.
					checker := CheckerFactory(checkData, logger)
					if checker == nil {
						logger.Error("Failed to create checker")
						continue
					}

					logger.Debug("Starting individual check")

					// Run the check.
					isHealthy := true
					errMessage := ""
					runTime := time.Now()

					checkDuration, err := checker.Run()
					if err != nil {
						successfulChecks++
						isHealthy = false
						errMessage = err.Error()
					}

					logger.WithFields(logrus.Fields{
						"healthy":     isHealthy,
						"message":     errMessage,
						"duration_ms": checkDuration.Milliseconds(),
					}).Info("Check completed")

					checkStatus := models.CheckStatus{
						UUID:      checkData.UUID,
						Project:   projectName,
						CheckName: checkName,
						CheckType: checkData.Type,
						LastRun:   runTime,
						IsHealthy: isHealthy,
						Message:   errMessage,
						IsEnabled: true, // assuming true if it’s not skipped
					}
					// Preserve ID and LastAlertSent if the record already exists.
					if currentStatus != nil {
						checkStatus.ID = currentStatus.ID
						checkStatus.LastAlertSent = currentStatus.LastAlertSent
					}

					// Update the status in the database.
					if err := upsertCheckStatus(mongoDB, &checkStatus); err != nil {
						logger.WithError(err).Error("Failed to update check status")
					}

					// Handle alerts if the check fails.
					if !isHealthy {
						if shouldSendAlert(effectiveDuration, checkStatus) {
							alertStartTime := time.Now()
							sendAlerts(cfg, checkStatus)
							logger.WithField("alert_duration_ms", time.Since(alertStartTime).Milliseconds()).
								Info("Alert sent")
							checkStatus.LastAlertSent = runTime
							if err := upsertCheckStatus(mongoDB, &checkStatus); err != nil {
								logger.WithError(err).Error("Failed to update last alert time")
							}
						}
					}
				}
			}
		}
	}

	//logrus.WithFields(logrus.Fields{
	//	"duration": duration,
	//	"total":    totalChecks,
	//	"success":  successfulChecks,
	//	"failed":   totalChecks - successfulChecks,
	//}).Info("Check run summary")
}

// findCheckStatus looks up the check status in the database.
func findCheckStatus(mongoDB *db.MongoDB, UUID string) (*models.CheckStatus, error) {
	// Stub implementation – replace with actual MongoDB query.
	filter := bson.M{
		"UUID": UUID,
	}

	var checkStatus models.CheckStatus
	err := mongoDB.Collection("check_statuses").FindOne(context.Background(), filter).Decode(&checkStatus)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, fmt.Errorf("error finding check status: %w", err)
	}

	return &checkStatus, nil
}

// upsertCheckStatus inserts or updates a CheckStatus in the database.
func upsertCheckStatus(mongoDB *db.MongoDB, status *models.CheckStatus) error {
	logrus.Debugf("Upserting check status for UUID: %s, Project: %s, CheckName: %s", status.UUID, status.Project, status.CheckName)
	// Implement MongoDB upsert logic
	filter := bson.M{
		"UUID": status.UUID,
	}

	update := bson.M{
		"$set": bson.M{
			"UUID":            status.UUID,
			"project":         status.Project,
			"check_name":      status.CheckName,
			"is_healthy":      status.IsHealthy,
			"message":         status.Message,
			"last_run":        status.LastRun,
			"last_alert_sent": status.LastAlertSent,
			"updated_at":      time.Now(),
			"host":            status.Host,
			"periodicity":     status.Periodicity,
		},
	}

	opts := options.Update().SetUpsert(true)

	result, err := mongoDB.Collection("check_statuses").UpdateOne(
		context.Background(),
		filter,
		update,
		opts,
	)
	if err != nil {
		logrus.Errorf("Error upserting check status: %v", err)
		return fmt.Errorf("error upserting check status: %w", err)
	}
	logrus.Debugf("Upsert result - Modified: %d, Upserted ID: %v", 
		result.ModifiedCount, result.UpsertedID)
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

// sendAlerts dispatches alerts based on the configuration.
func sendAlerts(cfg *config.Config, status models.CheckStatus) {
	// Stub implementation – invoke alert functions, e.g., slack/telegram.
	logrus.Infof("Sending alert for project %s, check %s", status.Project, status.CheckName)
	// Example: alerts.SendTelegram(...), or alerts.SendSlack(...)
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
