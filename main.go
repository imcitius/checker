package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"my/checker/alerts"
	"my/checker/checks"
	"my/checker/config"
	"my/checker/scheduler"
	"os"
	"os/signal"
	"syscall"
)

var (
	// Viper config location
	cfgFile  string
	logLevel string
)

func main() {
	app := &cli.App{
		Name:  "check",
		Usage: "To check",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Value:       "testconfigs.config.yaml",
				Usage:       "config file",
				Destination: &cfgFile,
			},
			&cli.StringFlag{
				Name:        "logLevel",
				Value:       "info",
				Usage:       "log level",
				Destination: &logLevel,
			},
		},

		Action: func(*cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())

			go check(ctx)

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			select {
			case <-ctx.Done():
				// Background process finished
				fmt.Println("Background process finished.")
			case <-sigChan:
				// Interrupt signal received, cancel the context
				cancel()
				fmt.Println("Interrupt signal received. Stopping background process...")
			}

			return nil
		},
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func check(ctx context.Context) error {
	config.InitLog(logLevel)
	config.InitConfig(cfgFile)
	checks.InitChecks()

	for {
		select {
		case <-ctx.Done():
			// Stop the background process gracefully
			fmt.Println("Background process stopping...")
			alerts.StopAlerters()
			return nil
		default:
			alerts.InitAlerts()
			scheduler.RunScheduler()
		}
	}
}
