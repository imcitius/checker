package alerts

import (
	"context"
	"github.com/sirupsen/logrus"
	"my/checker/config"
	"my/checker/models"
)

var (
	configurer models.Configurer
	logger     *logrus.Logger
	//cache      *memoize.Memoizer
	alerters *TAlertersCollection
	wg       = config.GetWG()
)

func InitAlerts(ctx context.Context, cfg models.Configurer) {
	configurer = cfg
	logger = config.GetLog()
	//cache = memoize.NewMemoizer(24*time.Hour, 24*time.Hour)

	err := initAlerters(ctx)
	if err != nil {
		logger.Fatalf("cannot init alerters: %s", err)
	}

	if len(configurer.GetAlerts()) == 0 {
		logger.Info("no alerts found in config, use log only")
	}
}

func StopAlerters() {
	for _, a := range alerters.Alerters {
		a.Stop(wg)
	}
}
