package log

func (a *TLogAlerter) Init() {}

func (a *TLogAlerter) Start() {}

func (a *TLogAlerter) Send(channel any, message string) {
	a.Log.Infof("Channel %s, message: %s", channel, message)
}

func (a *TLogAlerter) Stop() {}
