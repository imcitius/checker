package cmd

import (
	"my/checker/catalog"
	"my/checker/config"
	"my/checker/scheduler"
	"my/checker/status"
	"my/checker/web"
	"os"
	"time"
)

func mainChecker() {
	for {
		config.Log.Info("Start main loop")
		go signalWait()
		interrupt = false

		err := config.LoadConfig()
		if err != nil {
			config.Log.Infof("Config load error: %s", err)
		}

		if watchConfig {
			config.Log.Info("Start config watch")
			go config.WatchConfig()
		} else {
			config.Log.Info("Config watch disabled")
		}

		if len(config.Timeouts.Periods) == 0 {
			config.Log.Fatal("No periods found")
		} else {
			// adding all possible healthchecks periods
			for _, ticker := range config.Timeouts.Periods {
				tickerDuration, err := time.ParseDuration(ticker)
				config.Log.Infof("Create ticker: %s", ticker)
				if err != nil {
					config.Log.Fatal(err)
				}
				config.TickersCollection[ticker] = config.Ticker{Ticker: *time.NewTicker(tickerDuration), Description: ticker}
			}
			config.Log.Debugf("Tickers generated: %+v", config.TickersCollection)
		}

		config.Wg.Add(1)
		config.Log.Debugf("Fire scheduler")
		go scheduler.RunScheduler(config.SchedulerSignalCh, &config.Wg)

		if config.Config.ConsulCatalog.Enabled {
			catalog.WatchServices()
		}

		err = status.InitStatuses()
		if err != nil {
			config.Log.Infof("Status init error: %s", err)
		}

		if config.Sem.TryAcquire(1) {
			config.Log.Debugf("Fire webserver")
			go web.Serve(config.WebSignalCh, config.Sem)
		} else {
			config.Log.Debugf("Webserver already running")
		}

		config.Log.Debugf("config botsEnabled is %v", config.Config.Defaults.BotsEnabled)

		switch config.Config.Defaults.BotsEnabled {
		case true:
			fireActiveBot()
		case false:
			firePassiveBot()
		}

		//config.InternalStatus = "started"

		config.Wg.Wait()

		if !interrupt {
			config.Log.Debug("Checker init complete")
		} else {
			config.Log.Debug("Checker stopped")
			os.Exit(1)
		}
	}
}
