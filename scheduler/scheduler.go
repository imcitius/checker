package scheduler

import "C"
import (
	"my/checker/config"
	"sync"
)

var Config = &config.Config

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	config.Log.Info("Scheduler started")
	config.Log.Debugf("Timeouts: %+v", config.Timeouts.Periods)

	config.Log.Debugf("Tickers %+v", config.TickersCollection)
	if len(config.TickersCollection) == 0 {
		config.Log.Fatal("No tickers")
	} else {
		for i, t := range config.TickersCollection {
			config.Log.Debugf("Run ticker: %s\n\n", i)
			config.Wg.Add(1)
			//config.Log.Infof("I: %d", i)
			//config.Log.Infof("T: %d", t)
			go runProjectTicker(t, i, wg, signalCh)
		}
	}
	go runReportsTicker(config.ReportsTicker, config.Config.Defaults.Parameters.ReportPeriod, wg, signalCh)
}
