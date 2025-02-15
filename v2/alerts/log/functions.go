package log

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync"
	"my/checker/models"
)

func (a *TLogAlerter) Init(ctx context.Context) {}

func (a *TLogAlerter) Start(ctx context.Context, wg *sync.WaitGroup) {}

func (a *TLogAlerter) Alert(ctx context.Context, alertDetails models.TAlertDetails) {
	a.Log.Infof("Channel non-critical, message: %s", alertDetails.Message)
}

func (a *TLogAlerter) AlertCritical(ctx context.Context, alertDetails models.TAlertDetails) {
	a.Log.Infof("Channel CRITICAL, message: %s", alertDetails.Message)
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
