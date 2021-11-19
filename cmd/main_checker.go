package cmd

import (
	"my/checker/catalog"
	"my/checker/config"
	"my/checker/scheduler"
	"my/checker/status"
	"my/checker/web"
	"os"
)

func mainChecker() {
	for {
		config.Log.Info("Start main loop")
		go signalWait()
		interrupt = false

		if err := config.LoadConfig(); err != nil {
			config.Log.Infof("Config load error: %s", err)
		}

		if watchConfig {
			config.Log.Info("Start config watch")
			go config.WatchConfig()
		} else {
			config.Log.Info("Config watch disabled")
		}

		if err := config.StartTickers(); err != nil {
			config.Log.Fatalf("Error starting tickers: %s", err.Error())
		}

		config.Wg.Add(1)
		config.Log.Debugf("Fire scheduler")
		go scheduler.RunScheduler(config.SchedulerSignalCh, &config.Wg)

		if config.Config.ConsulCatalog.Enabled {
			catalog.WatchServices()
		}

		if err := status.InitStatuses(); err != nil {
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
