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
		for _, ticker := range config.TickersCollection {
			runProjectTickers(&ticker, wg, signalCh)
		}
		runReportsTicker(wg, signalCh)
	}
}
