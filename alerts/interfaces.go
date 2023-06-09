package alerts

type ICommonAlerter interface {
	Init()
	Start()
	Send(channel any, message string)
	Stop()
}
