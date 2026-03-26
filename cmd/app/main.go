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
	"checker/internal/discord"
	"checker/internal/scheduler"
	"checker/internal/slack"
	"checker/internal/telegram"
	"checker/internal/web"
)

// Injected at build time via -ldflags.
var (
	Version   string // git SHA
	BuildTime string // build timestamp
)

// newRepository creates the appropriate Repository implementation based on DB driver config.
func newRepository(cfg *config.Config) (db.Repository, error) {
	switch cfg.DB.Driver {
	case "sqlite":
		return db.NewSQLiteDB(cfg.DB.DSN)
	default: // "postgres"
		return db.NewPostgresDB(cfg)
	}
}

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

			// 2. Initialize Database
			logrus.Infof("Connecting to database (driver: %s)", cfg.DB.Driver)
			repo, err := newRepository(cfg)
			if err != nil {
				return fmt.Errorf("failed to connect to Database: %w", err)
			}
			logrus.Info("Database connection established")

			// Demo mode: wipe and reseed on every startup to prevent user-created checks persisting
			if os.Getenv("DEMO_MODE") == "true" {
				logrus.Info("Demo mode: wiping all checks and reseeding from demo/seed.yaml")
				if err := wipAndReseed(ctx, repo, "demo/seed.yaml"); err != nil {
					logrus.Warnf("Failed to reseed demo data: %v", err)
					// non-fatal — app still starts
				}
			} else if cfg.DB.Driver == "sqlite" {
				// Non-demo SQLite: seed only if empty
				count, err := repo.CountCheckDefinitions(ctx)
				if err != nil {
					logrus.Warnf("Failed to count check definitions: %v", err)
				} else if count == 0 {
					logrus.Info("SQLite: seeding checks from demo/seed.yaml")
					if err := seedFromFile(repo, "demo/seed.yaml"); err != nil {
						logrus.Warnf("Failed to seed data: %v", err)
					}
				}
			}

			// Ensure Database is closed on exit
			defer func() {
				logrus.Info("Closing Database connection")
				repo.Close()
			}()

			// 3. Build AppAlerter and WebhookRegistrar slices
			var appAlerters []scheduler.AppAlerter
			var webhooks []web.WebhookRegistrar

			// Slack App
			if cfg.SlackApp.BotToken != "" {
				slackClient := slack.NewSlackClient(cfg.SlackApp.BotToken, cfg.SlackApp.SigningSecret, cfg.SlackApp.DefaultChannel)
				logrus.Info("Slack App client initialized")
				appAlerters = append(appAlerters, scheduler.NewSlackAlerter(slackClient, repo, cfg.SlackApp.DefaultChannel))
				webhooks = append(webhooks, &web.SlackWebhookRegistrar{Client: slackClient})
			} else {
				logrus.Info("Slack App not configured, skipping")
			}

			// Discord Bot App
			if cfg.DiscordApp.BotToken != "" {
				discordClient := discord.NewDiscordClient(cfg.DiscordApp.BotToken, cfg.DiscordApp.AppID, cfg.DiscordApp.DefaultChannel)
				logrus.Info("Discord Bot client initialized")
				appAlerters = append(appAlerters, scheduler.NewDiscordAppAlerter(discordClient, repo, cfg.DiscordApp.DefaultChannel))
				webhooks = append(webhooks, &web.DiscordWebhookRegistrar{
					Client:    discordClient,
					PublicKey: cfg.DiscordApp.PublicKey,
				})
			} else {
				logrus.Info("Discord Bot App not configured, skipping")
			}

			// Telegram App
			if cfg.TelegramApp.BotToken != "" {
				tgClient := telegram.NewTelegramClient(cfg.TelegramApp.BotToken, cfg.TelegramApp.SecretToken, cfg.TelegramApp.DefaultChatID)
				logrus.Info("Telegram App client initialized")
				// Set webhook on startup if URL configured
				if cfg.TelegramApp.WebhookURL != "" {
					webhookURL := cfg.TelegramApp.WebhookURL + "/api/telegram/webhook"
					if err := tgClient.SetWebhook(context.Background(), webhookURL, cfg.TelegramApp.SecretToken); err != nil {
						logrus.Errorf("Failed to set Telegram webhook: %v", err)
					} else {
						logrus.Infof("Telegram webhook set to %s", webhookURL)
					}
				}
				appAlerters = append(appAlerters, scheduler.NewTelegramAppAlerter(tgClient, repo, cfg.TelegramApp.DefaultChatID))
				webhooks = append(webhooks, &web.TelegramWebhookRegistrar{Client: tgClient})
			} else {
				logrus.Info("Telegram App not configured, skipping")
			}

			// 3c. Initialize Auth Manager
			authMgr, err := auth.NewAuthManager(ctx, cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize auth: %w", err)
			}

			// 3d. Migrate legacy alert fields to AlertChannels
			if count, err := repo.MigrateLegacyAlertFields(ctx); err != nil {
				logrus.Errorf("Failed to migrate legacy alert fields: %v", err)
			} else if count > 0 {
				logrus.Infof("Migrated %d checks from legacy alert fields to AlertChannels", count)
			}

			// 4. Start Scheduler in background
			logrus.Info("Starting scheduler")
			schedulerCtx, schedulerCancel := context.WithCancel(ctx)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := scheduler.RunScheduler(schedulerCtx, cfg, repo, appAlerters); err != nil {
					logrus.Errorf("Scheduler error: %v", err)
				}
			}()

			// 5. Start Web Server (in a goroutine so we can handle graceful shutdown)
			logrus.Info("Starting web server")
			serverCtx, serverCancel := context.WithCancel(ctx)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := web.RunServer(serverCtx, cfg, repo, webhooks, authMgr); err != nil {
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
