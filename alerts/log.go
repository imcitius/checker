package alerts

import (
	"my/checker/config"
	"sync"
)

type LogAlert struct {
	Alerter
}

func (l *LogAlert) Send(a *AlertConfigs, message, messageType string) error {
	config.Log.Debugf("Alert send: %s (alert details %+v)", message, a)

	config.Log.Infof("Log alert: %s", message)

	return nil
}

func (l *LogAlert) InitBot(ch chan bool, wg *sync.WaitGroup) {
	config.Log.Info("Log bot not implemented yet")
	defer wg.Done()

	<-ch
}
