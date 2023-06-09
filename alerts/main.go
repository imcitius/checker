package alerts

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
)

var (
	Config *config.TConfig
	Log    *logrus.Logger
)

func InitAlerts() {
	Config = config.GetConfig()
	Log = config.GetLog()

	if len(Config.Alerts) == 0 {
		Log.Info("no alerts found in config, use log only")
	}
}
