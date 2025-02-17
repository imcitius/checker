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
	"my/checker/store/mongodb"
	"my/checker/web"
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

			cancel()
			return nil
		},
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func check(ctx context.Context) {
	config.InitLog(logLevel)
	config.InitConfig(cfgFile)
	checks.InitChecks()
	if config.GetConfig().DB.Protocol != "" {
		// causes panic, need to pass context
		//defer store.Store.Disconnect()
		store, err := store.InitDB(store.DBConfig{
			Protocol: config.GetConfig().DB.Protocol,
			Host:     config.GetConfig().DB.Host,
			Port:     config.GetConfig().DB.Port,
			Username: config.GetConfig().DB.Username,
			Password: config.GetConfig().DB.Password,
		})
		if err != nil {
			logrus.Fatalf("DB connect error: %s", err.Error())
		}
		config.GetConfig().SetDBConnected(store)
		
		loadChecks(store)
		loadAlerts(store)

	} else {
		logrus.Infof("DB is not configured")
	}
	go web.Listen()

	for {
		select {
		case <-ctx.Done():
			// Stop the background process gracefully
			fmt.Println("Background process stopping...")
			alerts.StopAlerters()
			return
		default:
			alerts.InitAlerts(ctx)
			scheduler.RunScheduler(ctx)
		}
	}
}

func loadChecks(store store.IStore) {
	checksFromDb, err := config.GetConfig().GetAllChecks()
	if err != nil {
		logrus.Fatalf("Cannot get checks from DB: %s", err)
	}
	for _, o := range checksFromDb {
		check, err := config.GetConfig().GetCheckByUUid(o.UUID)
		if err != nil {
			logrus.Debugf("Cannot get check from db: %s", err.Error())
			continue
		}
		check.LastExec = o.LastExec
		check.LastPing = o.LastPing
		check.LastResult = o.LastResult
		check.Enabled = o.Enabled

		err = config.GetConfig().UpdateCheckByUUID(check)
		if err != nil {
			logrus.Errorf("Cannot update check: %s", err.Error())
		}
	}
}

func loadAlerts(store store.IStore) {
	//alertsFromDb, err := Store.GetAllAlerts()

	var err error = nil
	if err != nil {
		logrus.Fatalf("Cannot get checks from DB: %s", err)
	}
	//for _, o := range alertsFromDb.data {
	//
	//}
}
