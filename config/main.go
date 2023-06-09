package config

import (
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
