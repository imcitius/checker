package alerts

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
)

var (
	configurer *config.TConfig
	logger     *logrus.Logger
)

func InitAlerts() {
	configurer = config.GetConfig()
	logger = config.GetLog()

	if len(configurer.Alerts) == 0 {
		logger.Info("no alerts found in config, use log only")
	}
}
