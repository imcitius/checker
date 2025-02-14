package alerts

import (
	"context"
	"my/checker/config"
	"sync"
)

type ICommonAlerter interface {
	Init(ctx context.Context)
	Start(ctx context.Context, wg *sync.WaitGroup)
	Alert(ctx context.Context, alertDetails config.TAlertDetails)
	AlertCritical(ctx context.Context, alertDetails config.TAlertDetails)
	Stop(wg *sync.WaitGroup)
	IsBot() bool
}
