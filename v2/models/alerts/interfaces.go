
package alerts

import (
	"context"
	"sync"
)

type ICommonAlerter interface {
	Init(ctx context.Context)
	Start(ctx context.Context, wg *sync.WaitGroup)
	Alert(ctx context.Context, alertDetails TAlertDetails)
	AlertCritical(ctx context.Context, alertDetails TAlertDetails)
	Stop(wg *sync.WaitGroup)
	IsBot() bool
}
