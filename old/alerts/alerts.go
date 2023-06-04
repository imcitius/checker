package alerts

import (
	"sync"
)

type Alerter interface {
	Send(a *AlertConfigs, message, messageType string) error
	InitBot(botsSignalCh chan bool, wg *sync.WaitGroup)
}

var AlerterCollections map[string]Alerter

func init() {
	AlerterCollections = make(map[string]Alerter)
	AlerterCollections["log"] = new(LogAlert)
	AlerterCollections["mattermost"] = new(Mattermost)
	AlerterCollections["telegram"] = new(Telegram)
}
