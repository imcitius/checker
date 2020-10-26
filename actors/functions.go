package actors

import "my/checker/config"

func GetActorByName(name string) *Actor {

	if name == "" {
		config.Log.Warn("Cannot get actor with empty name")
	}

	for _, a := range config.Config.Actors {
		//config.Log.Infof("'%s' '%s'", a.Name, name)
		if a.Name == name {
			return &Actor{a}
		}
	}
	return nil
}
