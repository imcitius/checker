package scheduler

import (
    "context"
    "time"

    "github.com/sirupsen/logrus"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "checker/internal/alerts"
    "checker/internal/checks"
    "checker/internal/config"
    "checker/internal/db"
    "checker/internal/models"
)

func RunScheduler(cfg *config.Config, mongoDB *db.MongoDB) {
    // Basic ticker approach
    ticker := time.NewTicker(cfg.Defaults.Duration)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            performAllChecks(cfg, mongoDB)
        }
    }
}

// performAllChecks iterates over all projects & checks, runs them, and stores results
func performAllChecks(cfg *config.Config, mongoDB *db.MongoDB) {
    for projectName, project := range cfg.Projects {
        // Use project.Parameters.Duration if present, else default
        checkInterval := project.Parameters.Duration
        if checkInterval == 0 {
            checkInterval = cfg.Defaults.Duration
        }

        for checkGroup, groupDetails := range project.HealthChecks {
            for checkName, checkData := range groupDetails.Checks {
                // Retrieve current status from DB to see if it's enabled
                currentStatus, err := findCheckStatus(mongoDB, projectName, checkGroup, checkName)
                if err != nil && err != mongo.ErrNoDocuments {
                    logrus.Errorf("Failed to fetch check status: %v", err)
                    continue
                }

                // If new, default isEnabled to true
                isEnabled := true
                if currentStatus != nil {
                    isEnabled = currentStatus.IsEnabled
                }
                if !isEnabled {
                    logrus.Infof("Skipping disabled check %s/%s/%s", projectName, checkGroup, checkName)
                    continue
                }

                // Run the appropriate check
                isHealthy, message := runCheck(checkData.Type, checkData.URL, checkData.AnswerPresent)
                now := time.Now()

                // Prepare record
                checkStatus := models.CheckStatus{
                    Project:    projectName,
                    CheckGroup: checkGroup,
                    CheckName:  checkName,
                    CheckType:  checkData.Type,
                    LastRun:    now,
                    IsHealthy:  isHealthy,
                    Message:    message,
                    IsEnabled:  isEnabled,
                }
                if currentStatus != nil {
                    checkStatus.ID = currentStatus.ID
                    checkStatus.LastAlertSent = currentStatus.LastAlertSent
                }

                // Update DB
                upsertCheckStatus(mongoDB, &checkStatus)

                // Send alerts if needed
                if !isHealthy {
                    shouldAlert := shouldSendAlert(cfg, checkStatus)
                    if shouldAlert {
                        sendAlerts(cfg, checkStatus)
                        checkStatus.LastAlertSent = now
                        upsertCheckStatus(mongoDB, &checkStatus)
                    }
                }
            }
        }
    }
}

func runCheck(checkType, url string, answerPresent bool) (bool, string) {
    switch checkType {
    case "http":
        return checks.HTTPCheck(url, answerPresent)
    case "tcp":
        return checks.TCPCheck(url)
    case "ping":
        return checks.PingCheck(url)
    default:
        return false, "Unknown check type"
    }
}

// upsertCheckStatus either inserts or updates a CheckStatus in MongoDB
func upsertCheckStatus(mongoDB *db.MongoDB, status *models.CheckStatus) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    filter := bson.M{
        "project":    status.Project,
        "check_group": status.CheckGroup,
        "check_name":  status.CheckName,
    }
    update := bson.M{"$set": status}
    _, err := mongoDB.Database.Collection("check_statuses").UpdateOne(ctx, filter, update, 
        options.Update().SetUpsert(true),
    )
    if err != nil {
        logrus.Errorf("Failed to upsert status for %s: %v", status.CheckName, err)
    }
}

func findCheckStatus(mongoDB *db.MongoDB, project, group, name string) (*models.CheckStatus, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    filter := bson.M{
        "project":     project,
        "check_group": group,
        "check_name":  name,
    }

    var status models.CheckStatus
    err := mongoDB.Database.Collection("check_statuses").FindOne(ctx, filter).Decode(&status)
    if err != nil {
        return nil, err
    }
    return &status, nil
}

func sendAlerts(cfg *config.Config, status models.CheckStatus) {
    alertChannel := cfg.Defaults.AlertsChannel
    // If needed, you can override at project level, etc.

    if alert, ok := cfg.Alerts[alertChannel]; ok {
        msg := "ALERT: " + status.Project + " | " + status.CheckGroup + " | " + status.CheckName + " is DOWN. " + status.Message
        switch alert.Type {
        case "telegram":
            // Decide whether to send to critical_channel or noncritical_channel
            // In real usage, you might check severity or other conditions
            alerts.SendTelegramAlert(alert.BotToken, alert.CriticalChannel, msg)
        case "slack":
            // Slack alert
            // alerts.SendSlackAlert(alert.WebhookURL, msg)
        }
    } else {
        logrus.Warnf("No alert configuration found for channel: %s", alertChannel)
    }
}

// Decide whether to send an alert. This could factor in throttling, maintenance windows, etc.
func shouldSendAlert(cfg *config.Config, status models.CheckStatus) bool {
    // Example: only send alert if last alert was more than X minutes ago
    if time.Since(status.LastAlertSent) < 5*time.Minute {
        return false
    }
    return true
}