package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"checker/internal/auth"
	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/scheduler"
	"checker/internal/slack"
	"checker/internal/web"
)

// Injected at build time via -ldflags.
var (
	Version   string // git SHA
	BuildTime string // build timestamp
)

func main() {
	// Pass build-time version info to the web package.
	web.AppVersion = Version
	web.BuildTime = BuildTime
	// Set up logging
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set log level to debug by default
	logrus.SetLevel(logrus.InfoLevel)

	// CLI App
	app := &cli.App{
		Name:  "checker",
		Usage: "A health-check application that runs scheduled checks and sends alerts",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enable debug logging",
				Value: false,
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "path to configuration file",
				Value:   "config.yaml",
			},
		},
		Action: func(c *cli.Context) error {
			// Set log level based on debug flag
			if c.Bool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debug("Debug logging enabled")
			}

			// Create a base context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			var wg sync.WaitGroup

			// 1. Load config
			logrus.Info("Loading configuration")
			configPath := c.String("config")
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			logrus.Infof("Configuration loaded successfully from %s", configPath)

			// 2. Initialize Database (PostgreSQL)
			logrus.Info("Connecting to Database")
			// We now use PostgresDB as the implementation of Repository
			repo, err := db.NewPostgresDB(cfg)
			if err != nil {
				return fmt.Errorf("failed to connect to Database: %w", err)
			}
			logrus.Info("Database connection established")

			// Ensure Database is closed on exit
			defer func() {
				logrus.Info("Closing Database connection")
				repo.Close()
			}()

			// 3. Initialize Slack App client
			var slackClient *slack.SlackClient
			if cfg.SlackApp.BotToken != "" {
				slackClient = slack.NewSlackClient(cfg.SlackApp.BotToken, cfg.SlackApp.SigningSecret, cfg.SlackApp.DefaultChannel)
				logrus.Info("Slack App client initialized")
			} else {
				logrus.Info("Slack App not configured, skipping")
			}

			// 3b. Initialize Auth Manager
			authMgr, err := auth.NewAuthManager(ctx, cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize auth: %w", err)
			}

			var slackAlerter *scheduler.SlackAlerter
			if slackClient != nil {
				slackAlerter = scheduler.NewSlackAlerter(slackClient, repo, cfg.SlackApp.DefaultChannel)
			}

			// 4. Start Scheduler in background
			logrus.Info("Starting scheduler")
			schedulerCtx, schedulerCancel := context.WithCancel(ctx)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := scheduler.RunScheduler(schedulerCtx, cfg, repo, slackAlerter); err != nil {
					logrus.Errorf("Scheduler error: %v", err)
				}
			}()

			// 5. Start Web Server (in a goroutine so we can handle graceful shutdown)
			logrus.Info("Starting web server")
			serverCtx, serverCancel := context.WithCancel(ctx)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := web.RunServer(serverCtx, cfg, repo, slackClient, authMgr); err != nil {
					logrus.Errorf("Web server error: %v", err)
					// Trigger app shutdown if web server fails
					cancel()
				}
			}()

			// Wait for termination signal
			sig := <-sigCh
			logrus.Infof("Received signal: %v. Initiating graceful shutdown...", sig)

			// Cancel all contexts to signal shutdown
			cancel()
			serverCancel()
			schedulerCancel()

			// Wait with timeout for all goroutines to finish
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			// Use a channel to signal completion of goroutines
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				logrus.Info("All components shut down gracefully")
			case <-shutdownCtx.Done():
				logrus.Warn("Shutdown timed out, some components may not have terminated properly")
			}

			return nil
		},
	}

	// Run the CLI app
	if err := app.Run(os.Args); err != nil {
		logrus.Fatalf("Application failed to start: %v", err)
	}
}
