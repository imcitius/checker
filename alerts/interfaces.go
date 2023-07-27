package alerts

import "sync"

type ICommonAlerter interface {
	Init()
	Start(wg *sync.WaitGroup)
	Send(message string)
	SendCritical(message string)
	Stop(wg *sync.WaitGroup)
	IsBot() bool
}
