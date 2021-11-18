package cmd

import (
	"fmt"
	"my/checker/alerts"
	"my/checker/config"
)

func fireActiveBot() {
	if botsEnabled {
		config.Log.Infof("Active bot is enabled")
		fireBot()
	} else {
		firePassiveBot()
	}

}

func firePassiveBot() {
	if !botsEnabled {
		config.Log.Infof("Active bot is disabled, alerts only")
		message := fmt.Sprintf("Bot at your service (%s, %s, %s)\nActive bot is disabled, alerts only", config.Version, config.VersionSHA, config.VersionBuild)
		// Metrics structures is not initialized yet, so we prevent panic with "noMetrics"
		alerts.SendChatOps(message, "noMetrics")
	} else {
		fireActiveBot()
	}
}

func fireBot() {
	config.Log.Debugf("Fire bot")
	config.Wg.Add(1)
	commandChannel, err := alerts.GetCommandChannel()
	if err != nil {
		config.Log.Infof("root GetCommandChannel error: %s", err)
	} else {
		a := commandChannel.GetAlertProto()
		if a == nil {
			config.Log.Fatal("root commandChannel not found, bot not initialized")
		} else {
			a.InitBot(config.BotsSignalCh, &config.Wg)
		}
	}
}

func testConfig() {
	_, err := config.TestConfig()
	if err != nil {
		config.Log.Infof("Config loading error: %+v", err)
	} else {
		config.Log.Infof("Config load ok (err: %+v)", err)
	}
}

func signalWait() {
	config.Log.Info("Start waiting signals")
	select {
	case <-config.SignalINT:
		config.Log.Infof("Got SIGINT")
		config.InternalStatus = "stop"
		interrupt = true
		close(config.SchedulerSignalCh)
		if config.Config.Defaults.BotsEnabled {
			config.BotsSignalCh <- true
		}
		config.WebSignalCh <- true
		return
	case <-config.SignalHUP:
		config.Log.Infof("Got SIGHUP")
		config.ChangeSig <- true
		return
	case <-config.ChangeSig:
		config.Log.Infof("Config file reload")
		config.InternalStatus = "reload"
		close(config.SchedulerSignalCh)
		//config.WebSignalCh <- true
		if config.Config.Defaults.BotsEnabled && botsEnabled {
			config.BotsSignalCh <- true
		}
		return
	}
}
