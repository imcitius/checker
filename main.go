package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"my/checker/alerts"
	"my/checker/checks"
	"my/checker/config"
	"my/checker/scheduler"
	"os"
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
				Value:       ".config.yaml",
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
			config.InitLog(logLevel)
			config.InitConfig(cfgFile)
			checks.InitChecks()
			alerts.InitAlerts()
			scheduler.RunScheduler()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

// config.InitConfig(cfgFile), config.InitLog(logLevel)

//func Run() {
//	scheduler.RunScheduler()
//}
