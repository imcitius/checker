package alerts

import (
	"context"
	"my/checker/config"
	"my/checker/internal/slack"
	"sync"
)

// SlackApp implements the Alerter interface for the Slack App integration.
// It uses the Slack Bot Token API for richer messaging (threads, reactions, etc.)
// as opposed to the legacy Mattermost/Slack webhook approach.
//
// Note: The primary slack_app alert flow (with thread tracking, silence checks,
// and recovery detection) is handled by the scheduler's SendSlackAppAlert and
// HandleSlackAppRecovery functions. This Alerter implementation serves as a
// fallback for the standard alert pipeline (e.g. critical project-level alerts).
type SlackApp struct {
	Alerter
}

func (s *SlackApp) Send(a *AlertConfigs, message, messageType string) error {
	if config.Config.SlackApp.BotToken == "" {
		config.Log.Warn("SlackApp alerter: bot token not configured, falling back to log")
		return nil
	}

	channelID := config.Config.SlackApp.DefaultChannel
	if channelID == "" {
		config.Log.Warn("SlackApp alerter: no default channel configured, falling back to log")
		return nil
	}

	client := slack.NewSlackClient(config.Config.SlackApp.BotToken)
	ctx := context.Background()

	alertInfo := slack.CheckAlertInfo{
		Name:    a.Name,
		Message: message,
	}

	_, err := client.PostAlert(ctx, channelID, alertInfo)
	if err != nil {
		config.Log.Errorf("SlackApp alert send error: %v", err)
		return err
	}

	config.Log.Debugf("SlackApp alert sent: %s (type: %s)", message, messageType)
	return nil
}

func (s *SlackApp) InitBot(_ chan bool, _ *sync.WaitGroup) {
	config.Log.Debug("SlackApp bot initialization: no dedicated bot loop needed")
}
