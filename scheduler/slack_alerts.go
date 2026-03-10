package scheduler

import (
	"context"
	"my/checker/config"
	"my/checker/internal/slack"
	projects "my/checker/projects"
)

// SendSlackAppAlert handles sending a Slack App alert for a failing check.
// It checks silence status, posts a rich alert via the SlackClient, and tracks the thread.
func SendSlackAppAlert(project *projects.Project, healthcheck *config.Healthcheck, check *config.Check, errMessage string) {
	if slackClient == nil {
		config.Log.Errorf("Slack App client not initialized for check %s", check.UUid)
		return
	}

	ctx := context.Background()

	// Check if silenced
	if repo != nil {
		silenced, err := repo.IsCheckSilenced(ctx, check.UUid, project.Name)
		if err != nil {
			config.Log.Errorf("Failed to check silence status for %s: %v", check.UUid, err)
		}
		if silenced {
			config.Log.Debugf("Check %s is silenced, skipping slack_app alert", check.UUid)
			return
		}
	}

	// Determine channel: use the project's alert channel config or the default channel
	channelID := config.Config.SlackApp.DefaultChannel
	if channelID == "" {
		config.Log.Errorf("No Slack App channel configured for check %s", check.UUid)
		return
	}

	alertInfo := slack.CheckAlertInfo{
		UUID:      check.UUid,
		Name:      check.Name,
		Project:   project.Name,
		Group:     healthcheck.Name,
		CheckType: check.Type,
		Message:   errMessage,
		IsHealthy: false,
	}

	// Post the alert
	messageTs, err := slackClient.PostAlert(ctx, channelID, alertInfo)
	if err != nil {
		config.Log.Errorf("Failed to send Slack App alert for check %s: %v", check.UUid, err)
		return
	}

	config.Log.Infof("Slack App alert sent for check %s (ts: %s)", check.UUid, messageTs)

	// Track the thread in the database
	if repo != nil {
		if err := repo.CreateSlackThread(ctx, check.UUid, channelID, messageTs, messageTs); err != nil {
			config.Log.Errorf("Failed to track Slack thread for check %s: %v", check.UUid, err)
		}
	}
}

// HandleSlackAppRecovery checks for unresolved Slack threads when a check recovers
// (transitions from unhealthy to healthy). If found, it posts a resolution reply,
// updates the parent message color to green, and marks the thread as resolved.
func HandleSlackAppRecovery(project *projects.Project, healthcheck *config.Healthcheck, check *config.Check) {
	if slackClient == nil || repo == nil {
		return
	}

	ctx := context.Background()

	thread, err := repo.GetUnresolvedThread(ctx, check.UUid)
	if err != nil {
		// No unresolved thread found — this is normal for checks that didn't alert via slack_app
		config.Log.Debugf("No unresolved Slack thread for check %s: %v", check.UUid, err)
		return
	}

	if thread.ThreadTs == "" {
		return
	}

	config.Log.Infof("Resolving Slack thread for check %s (channel: %s, thread: %s)", check.UUid, thread.ChannelID, thread.ThreadTs)

	alertInfo := slack.CheckAlertInfo{
		UUID:      check.UUid,
		Name:      check.Name,
		Project:   project.Name,
		Group:     healthcheck.Name,
		CheckType: check.Type,
		Message:   "Check is healthy again",
		IsHealthy: true,
	}

	// Post resolution reply in thread
	if err := slackClient.PostResolution(ctx, thread.ChannelID, thread.ThreadTs, alertInfo); err != nil {
		config.Log.Errorf("Failed to post Slack resolution for check %s: %v", check.UUid, err)
	}

	// Update parent message color to green
	if err := slackClient.UpdateMessageColor(ctx, thread.ChannelID, thread.ParentTs, "good", alertInfo); err != nil {
		config.Log.Errorf("Failed to update Slack message color for check %s: %v", check.UUid, err)
	}

	// Mark thread as resolved in database
	if err := repo.ResolveThread(ctx, check.UUid); err != nil {
		config.Log.Errorf("Failed to resolve Slack thread for check %s: %v", check.UUid, err)
	}
}
