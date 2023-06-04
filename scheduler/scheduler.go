package scheduler

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
)

func RunScheduler(Log *logrus.Logger, Config *config.Config) {
	Log.Info("Scheduler started")
	Log.Info(Config.Test)

	////Log.Debugf("Timeouts: %+v", config.Timeouts.Periods)
	//
	//Log.Debugf("Tickers %+v", config.TickersCollection)
	//if len(config.TickersCollection) == 0 {
	//	config.Log.Fatal("No tickers")
	//} else {
	//	for i, t := range config.TickersCollection {
	//		config.Log.Debugf("Run ticker: %s\n\n", i)
	//		config.Wg.Add(1)
	//		//config.Log.Infof("I: %d", i)
	//		//config.Log.Infof("T: %d", t)
	//		go runProjectTicker(t, i, wg, signalCh)
	//	}
	//}
	//go runReportsTicker(config.ReportsTicker, config.Config.Defaults.Parameters.ReportPeriod, wg, signalCh)
}
