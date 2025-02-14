package scheduler

import (
	"context"
	"my/checker/config"
)

var (
	wg         = config.GetWG()
	logger     = config.GetLog()
	configurer = config.GetConfig()
	//cache      *memoize.Memoizer
)

func RunScheduler(ctx context.Context) {

	//alerters, _ := config.GetAlerters()
	//alerters.Alerters[config.Defaults.DefaultAlertsChannel].Alert("default", "Scheduler started alert")

	tickers, _ := GetTickers()
	maintenances := GetMaintenanceTickers()

	logger.Infof("Schedulers started: ")

	counter := 1
	for _, ticker := range maintenances.Tickers {
		wg.Add(1)
		//logger.Infof("%s", desc)
		go runMaintenanceTicker(ticker, wg)
		counter++
	}

	for _, ticker := range tickers.Tickers {
		wg.Add(1)
		//logger.Infof("%s", desc)
		if counter == len(tickers.Tickers)+len(maintenances.Tickers) {
			runProjectTicker(ticker, ctx, wg)
		} else {
			go runProjectTicker(ticker, ctx, wg)
			counter++
		}
	}

	//go runReportsTicker(config.ReportsTicker, config.config.Defaults.CheckParameters.ReportPeriod, wg, signalCh)
}
