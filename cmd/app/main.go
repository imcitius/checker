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

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/scheduler"
	"checker/internal/web"
)

func main() {
	// Set up logging
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set log level to debug by default
	logrus.SetLevel(logrus.DebugLevel)

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

			// 2. Initialize MongoDB
			logrus.Info("Connecting to MongoDB")
			mongoDB, err := db.NewMongoDB(cfg)
			if err != nil {
				return fmt.Errorf("failed to connect to MongoDB: %w", err)
			}
			logrus.Info("MongoDB connection established")

			// Ensure MongoDB is closed on exit
			defer func() {
				logrus.Info("Closing MongoDB connection")
				if err := mongoDB.Close(ctx); err != nil {
					logrus.Errorf("Error closing MongoDB connection: %v", err)
				}
			}()

			// 3. Start Scheduler in background
			logrus.Info("Starting scheduler")
			schedulerCtx, schedulerCancel := context.WithCancel(ctx)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := scheduler.RunScheduler(schedulerCtx, cfg, mongoDB); err != nil {
					logrus.Errorf("Scheduler error: %v", err)
				}
			}()

			// 4. Start Web Server (in a goroutine so we can handle graceful shutdown)
			logrus.Info("Starting web server")
			serverCtx, serverCancel := context.WithCancel(ctx)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := web.RunServer(serverCtx, cfg, mongoDB); err != nil {
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
