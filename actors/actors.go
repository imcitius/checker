package actors

import "my/checker/config"

type ActorInterface interface {
	Do(a *Actor) error
}

type Actor struct {
	config.ActorConfigs
}

var ActorCollection map[string]ActorInterface

func init() {
	ActorCollection = make(map[string]ActorInterface)
	ActorCollection["log"] = new(LogActor)
}
