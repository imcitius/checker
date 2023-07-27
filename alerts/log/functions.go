package log

import (
	"github.com/sirupsen/logrus"
	"sync"
)

func (a *TLogAlerter) Init() {}

func (a *TLogAlerter) Start(wg *sync.WaitGroup) {}

func (a *TLogAlerter) Send(message string) {
	a.Log.Infof("Channel %s, message: %s", message)
}

func (a *TLogAlerter) SendCritical(message string) {
	a.Log.Infof("Channel %s, CRITICAL message: %s", message)
}

func (a *TLogAlerter) Stop(wg *sync.WaitGroup) {}

func NewAlerter(logger *logrus.Logger) *TLogAlerter {
	return &TLogAlerter{
		Log: logger,
	}
}

func (a *TLogAlerter) IsBot() bool {
	return false
}
