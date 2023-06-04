package status

import (
	"my/checker/config"
	"time"
)

var (
	MainStatus string
	Statuses   *Collection
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
	Name           string
	UUID           string
	Mode           string // represent current alerting mode
	Status         string // represent current checks status
	LastResult     bool
	When           time.Time
	ExecuteCount   int
	SeqErrorsCount int
	FailsCount     int
}

type Collection struct {
	Projects map[string]*ProjectsStatuses
	Checks   map[string]*CheckStatuses
}

func init() {
	Statuses = new(Collection)
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
				InitCheckStatus(&c)
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
			InitCheckStatus(&c)
		}
	}
}
