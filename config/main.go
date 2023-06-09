package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

var (
	StartTime = time.Now()
	config    = TConfig{
		Defaults: TDefaults{
			TimerStep: "1s",
			// How often to run check by default
			Duration: "60s",

			// HTTP port web interface listen
			HTTPPort: "80",
			// If empty HTTP server is not enabled
			HTTPEnabled:        "true",
			TokenEncryptionKey: []byte{},

			BotsEnabled:        true,
			BotGreetingEnabled: false,

			DebugLevel:    "info",
			AlertsChannel: "log",
		},
		Alerts:   map[string]TAlert{},
		Projects: map[string]TProject{},
	}
)

func initConfig(cfgFile string) {
	if cfgFile != "" {
		err := cleanenv.ReadConfig(cfgFile, &config)
		if err != nil {
			panic(fmt.Errorf("Error: uncaught error! %s", err))
		} else {
			logger.Infof("Using c file %s\n", cfgFile)
		}
	} else {
		panic(fmt.Errorf("config file not found at path %s\n", cfgFile))
	}
}
