package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"checker/internal/config"
	"checker/internal/db"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logrus.SetLevel(logrus.InfoLevel)

	// Load config
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	logrus.Infof("Loading configuration from %s", configPath)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	logrus.Info("Connecting to PostgreSQL")
	repo, err := db.NewPostgresDB(cfg)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	// Migrate checks from config to database
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logrus.Info("Importing checks from config to database...")
	if err := repo.ConvertConfigToCheckDefinitions(ctx, cfg); err != nil {
		logrus.Fatalf("Failed to import checks: %v", err)
	}

	logrus.Info("✅ Successfully imported checks from config.yaml to database")

	// Show imported checks
	checks, err := repo.GetAllCheckDefinitions(ctx)
	if err != nil {
		logrus.Errorf("Failed to retrieve checks: %v", err)
	} else {
		logrus.Infof("Total checks in database: %d", len(checks))
		for _, check := range checks {
			logrus.Infof("  - %s (%s/%s) - Type: %s, Duration: %s, Enabled: %v",
				check.Name, check.Project, check.GroupName, check.Type, check.Duration, check.Enabled)
		}
	}

	fmt.Println("\n✅ Migration completed successfully!")
	fmt.Println("You can now start the checker application and it will run these checks.")
}
