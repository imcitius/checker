package alerts

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
)

var (
	configurer *config.TConfig
	logger     *logrus.Logger
	//cache      *memoize.Memoizer
	alerters *TAlertersCollection
	wg       = config.GetWG()
)

func InitAlerts() {
	configurer = config.GetConfig()
	logger = config.GetLog()
	//cache = memoize.NewMemoizer(24*time.Hour, 24*time.Hour)

	err := initAlerters()
	if err != nil {
		logger.Fatalf("cannot init alerters: %s", err)
	}

	if len(configurer.Alerts) == 0 {
		logger.Info("no alerts found in config, use log only")
	}
}

func StopAlerters() {
	for _, a := range alerters.Alerters {
		a.Stop(wg)
	}
}
