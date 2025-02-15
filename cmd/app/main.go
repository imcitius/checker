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
    // CLI App
    app := &cli.App{
        Name:  "checker",
        Usage: "A health-check application that runs scheduled checks and sends alerts",
        Action: func(c *cli.Context) error {
            // 1. Load config
            cfg, err := config.LoadConfig("../../config.yaml")
            if err != nil {
                return fmt.Errorf("failed to load config: %w", err)
            }

            // 2. Initialize MongoDB
            mongoDB, err := db.NewMongoDB(cfg)
            if err != nil {
                return fmt.Errorf("failed to connect to MongoDB: %w", err)
            }

            // 3. Start Scheduler in background
            go scheduler.RunScheduler(cfg, mongoDB)

            // 4. Start Web Server
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