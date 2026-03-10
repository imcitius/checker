package scheduler

import "C"
import (
	checks "my/checker/checks"
	"my/checker/config"
	"my/checker/internal/db"
	"my/checker/internal/slack"
	"sync"
)

var Config = &config.Config

// Package-level references for Slack App integration.
var (
	slackClient *slack.SlackClient
	repo        db.Repository
)

// SetSlackClient sets the Slack client used for slack_app alerts.
func SetSlackClient(c *slack.SlackClient) {
	slackClient = c
}

// SetRepository sets the database repository used for thread tracking and silence checks.
func SetRepository(r db.Repository) {
	repo = r
}

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// Register Slack App callbacks if client and repo are available
	if slackClient != nil {
		checks.SlackAppAlertFunc = SendSlackAppAlert
		checks.SlackAppRecoveryFunc = HandleSlackAppRecovery
		config.Log.Info("Slack App alert callbacks registered with check evaluator")
	}

	config.Log.Info("Scheduler started")
	config.Log.Debugf("Timeouts: %+v", config.Timeouts.Periods)

	config.Log.Debugf("Tickers %+v", config.TickersCollection)
	if len(config.TickersCollection) == 0 {
		config.Log.Fatal("No tickers")
	} else {
		for i, t := range config.TickersCollection {
			config.Log.Debugf("Run ticker: %s\n\n", i)
			config.Wg.Add(1)
			//config.Log.Infof("I: %d", i)
			//config.Log.Infof("T: %d", t)
			go runProjectTicker(t, i, wg, signalCh)
		}
	}
	go runReportsTicker(config.ReportsTicker, config.Config.Defaults.Parameters.ReportPeriod, wg, signalCh)
}
