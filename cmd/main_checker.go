package cmd

import (
	"database/sql"
	"my/checker/catalog"
	"my/checker/config"
	"my/checker/internal/db"
	"my/checker/internal/slack"
	"my/checker/scheduler"
	"my/checker/status"
	"my/checker/web"
	"os"

	_ "github.com/lib/pq"
)

func mainChecker() {
	for {
		config.Log.Info("Start main loop")
		go signalWait()
		interrupt = false

		if err := config.LoadConfig(); err != nil {
			config.Log.Infof("Config load error: %s", err)
		}

		// Initialize Slack App integration if configured
		if config.Config.SlackApp.BotToken != "" {
			config.Log.Info("Slack App integration is enabled")
			slackClient := slack.NewSlackClient(
				config.Config.SlackApp.BotToken,
				config.Config.SlackApp.SigningSecret,
				config.Config.SlackApp.DefaultChannel,
			)
			scheduler.SetSlackClient(slackClient)
			web.SetSlackClient(slackClient)
			config.Log.Info("Slack App client initialized and passed to scheduler and web")

			// Initialize database repository for Slack thread tracking and silence checks
			if config.Config.SlackApp.DatabaseURL != "" {
				sqlDB, err := sql.Open("postgres", config.Config.SlackApp.DatabaseURL)
				if err != nil {
					config.Log.Errorf("Failed to open database for Slack integration: %v", err)
				} else {
					if err := sqlDB.Ping(); err != nil {
						config.Log.Errorf("Failed to connect to database for Slack integration: %v", err)
					} else {
						repository := db.NewPostgresDB(sqlDB)
						scheduler.SetRepository(repository)
						web.SetRepository(repository)
						config.Log.Info("Database repository initialized for Slack thread tracking and silences")
					}
				}
			} else {
				config.Log.Warn("Slack App database_url not configured; thread tracking and silence features are disabled")
			}
		}

		if watchConfig {
			config.Log.Info("Start config watch")
			go config.WatchConfig()
		} else {
			config.Log.Info("Config watch disabled")
		}

		if err := config.StartTickers(); err != nil {
			config.Log.Fatalf("Error starting tickers: %s", err.Error())
		}

		config.Wg.Add(1)
		config.Log.Debugf("Fire scheduler")
		go scheduler.RunScheduler(config.SchedulerSignalCh, &config.Wg)

		if config.Config.ConsulCatalog.Enabled {
			catalog.WatchServices()
		}

		if err := status.InitStatuses(); err != nil {
			config.Log.Infof("Status init error: %s", err)
		}

		if config.Sem.TryAcquire(1) {
			config.Log.Debugf("Fire webserver")
			go web.Serve(config.WebSignalCh, config.Sem)
		} else {
			config.Log.Debugf("Webserver already running")
		}

		config.Log.Debugf("config botsEnabled is %v", config.Config.Defaults.BotsEnabled)

		switch config.Config.Defaults.BotsEnabled {
		case true:
			fireActiveBot()
		case false:
			firePassiveBot()
		}

		//config.InternalStatus = "started"

		config.Wg.Wait()

		if !interrupt {
			config.Log.Debug("Checker init complete")
		} else {
			config.Log.Debug("Checker stopped")
			os.Exit(1)
		}
	}
}
