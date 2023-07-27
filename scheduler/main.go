package scheduler

import (
	"my/checker/config"
)

var (
	wg         = config.GetWG()
	logger     = config.GetLog()
	configurer = config.GetConfig()
	//cache      *memoize.Memoizer
)

func RunScheduler() {

	//alerters, _ := config.GetAlerters()
	//alerters.Alerters[config.Defaults.DefaultAlertsChannel].Send("default", "Scheduler started alert")

	tickers, _ := GetTickers()
	//spew.Dump(tickers)

	logger.Infof("Schedulers started: ")

	counter := 1
	for _, ticker := range tickers.Tickers {
		wg.Add(1)
		//logger.Infof("%s", desc)
		if counter == len(tickers.Tickers) {
			runProjectTicker(ticker, wg)
		} else {
			go runProjectTicker(ticker, wg)
			counter++
		}
	}
	//go runReportsTicker(config.ReportsTicker, config.config.Defaults.CheckParameters.ReportPeriod, wg, signalCh)
}
