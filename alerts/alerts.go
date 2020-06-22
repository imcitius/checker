package alerts

import (
	"my/checker/config"
	"sync"
)

var (
	botsSignalCh chan bool
)

type Alerter interface {
	Send(a *config.AlertConfigs, message string) error
	InitBot(botsSignalCh chan bool, wg *sync.WaitGroup)
}

var AlerterCollection map[string]Alerter
