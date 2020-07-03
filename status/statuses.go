package status

import (
	"my/checker/config"
)

var (
	MainStatus string
	Statuses   *StatusCollection
)

type ProjectsStatuses struct {
	Name           string
	Mode           string // represent current alerting mode
	Status         string // represent current checks status
	Alive          int
	SeqErrorsCount int
	FailsCount     int
}
type CheckStatuses struct {
	UUID       string
	Mode       string // represent current alerting mode
	Status     string // represent current checks status
	LastResult bool
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
		config.Log.Debugf("Init project %s statuses", p.Name)

		initProjectStatus(&p)

		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				initCheckStatus(&c)
			}
		}
	}

	//j, _ := json.Marshal(Metrics)
	//config.Log.Debugf("Metrics: %+v", string(j))

	return nil
}

func InitProject(p *config.Project) {

	config.Log.Debug("Init project status structure")

	initProjectStatus(p)

	for _, h := range p.Healthchecks {
		for _, c := range h.Checks {
			initCheckStatus(&c)
		}
	}
}
