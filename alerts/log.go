package alerts

import (
	"my/checker/config"
	"sync"
)

type Log struct {
	Alerter
}

func init() {
	AlerterCollections = make(map[string]Alerter)
	AlerterCollections["log"] = new(Log)
}

func (m *Log) Send(a *config.AlertConfigs, message string) error {
	config.Log.Debugf("Alert send: %s (alert details %+v)", message, a)

	config.Log.Infof("Log alert: %s", message)

	return nil
}

func (t Log) InitBot(ch chan bool, wg *sync.WaitGroup) {
	config.Log.Info("Log bot not implemented yet")
}
