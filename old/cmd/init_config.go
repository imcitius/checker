package cmd

import (
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/sirupsen/logrus"
	"my/checker/config"
	"strings"
)

func initConfig() {

	err := config.Koanf.Load(confmap.Provider(map[string]interface{}{
		"defaults.http.port":    "80",
		"defaults.http.enabled": true,
		// should always be 1s, to avoid time drift bugs in scheduler
		//"defaults.timer_step": "1s",
		"debug.level":         debugLevel,
		"log.format":          logFormat,
		"bots.enabled":        botsEnabled,
		"config.file":         configFile,
		"config.source":       configSource,
		"config.watchtimeout": configWatchTimeout,
		"config.format":       configFormat,
	}, "."), nil)
	if err != nil {
		logrus.Panicf("Cannot fill default config: %s", err.Error())
	}

	err = config.Koanf.Load(env.Provider("PORT", ".", func(s string) string {
		return "defaults.http.port"
	}), nil)
	if err != nil {
		logrus.Infof("PORT env not defined: %s", err.Error())
	}

	err = config.Koanf.Load(env.Provider("DEBUG_LEVEL", ".", func(s string) string {
		return "debug.level"
	}), nil)
	if err != nil {
		logrus.Infof("DEBUG_LEVEL env not defined: %s", err.Error())
	}

	for _, i := range []string{"CONSUL_", "VAULT_", "AWS_", "CHECKER_"} {
		err = config.Koanf.Load(env.Provider(i, ".", func(s string) string {
			return strings.Replace(strings.ToLower(
				s), "_", ".", -1)
		}), nil)
		if err != nil {
			logrus.Infof("%s env not defined: %s", i, err.Error())
		}
	}
}
