package actors

import (
	"my/checker/config"
)

type LogActor struct {
	Actor
}

func (l *LogActor) Do(a *Actor) error {
	config.Log.Debugf("Action: %+v (action details)", a)

	config.Log.Infof("Log action: %s", a.Name)

	return nil
}
