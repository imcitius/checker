package main

import (
	"fmt"
	"os"

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
		},
		Action: func(c *cli.Context) error {
			// Set log level based on debug flag
			if c.Bool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debug("Debug logging enabled")
			}

			// 1. Load config
			logrus.Info("Loading configuration")
			cfg, err := config.LoadConfig("config.yaml")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			logrus.Info("Configuration loaded successfully")

			// 2. Initialize MongoDB
			logrus.Info("Connecting to MongoDB")
			mongoDB, err := db.NewMongoDB(cfg)
			if err != nil {
				return fmt.Errorf("failed to connect to MongoDB: %w", err)
			}
			logrus.Info("MongoDB connection established")

			// 3. Start Scheduler in background
			logrus.Info("Starting scheduler")
			go scheduler.RunScheduler(cfg, mongoDB)

			// 4. Start Web Server
			logrus.Info("Starting web server")
			if err := web.RunServer(cfg, mongoDB); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}
			return nil
		},
	}

	// Run the CLI app
	if err := app.Run(os.Args); err != nil {
		logrus.Fatalf("Application failed to start: %v", err)
	}
}
