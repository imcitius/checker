package status

import (
"my/checker/config"
)

var (
	MainStatus string
    Statuses *StatusCollection
)

type ProjectsStatuses struct {
	Name   string
	Status string
}
type CheckStatuses struct {
	UUID   string
	Status string
}

type StatusCollection struct {
	Projects map[string]*ProjectsStatuses
	Checks   map[string]*CheckStatuses
}

func init() {
	Statuses = new(StatusCollection)
	Statuses.Projects = make(map[string]*ProjectsStatuses)
	Statuses.Checks = make(map[string]*CheckStatuses)
}

func InitStatuses() error {

	config.Log.Debug("Init status structures")

	for _, p := range config.Config.Projects {
		config.Log.Debugf("Init project %s metrics", p.Name)

		initProjectStatus(&p)

		for _, h := range p.Healtchecks {
			for _, c := range h.Checks {
				initCheckStatus(&c)
			}
		}
	}

	//j, _ := json.Marshal(Metrics)
	//config.Log.Debugf("Metrics: %+v", string(j))

	return nil
}
