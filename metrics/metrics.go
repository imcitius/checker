package metrics

import (
	"my/checker/config"
)

type ProjectsMetrics struct {
	Name           string
	SeqErrorsCount int
	ErrorsCount    int
	FailsCount     int
	RunCount       int
	Alive          int
}
type AlertMetrics struct {
	Name        string
	AlertCount  int
	NonCritical int
	Critical    int
}
type HealtcheckMetrics struct {
	Name        string
	RunCount    int
	ErrorsCount int
	FailsCount  int
}
type CheckMetrics struct {
	UUID        string
	RunCount    int
	ErrorsCount int
	FailsCount  int
}

type MetricsCollection struct {
	Projects     map[string]*ProjectsMetrics
	Alerts       map[string]*AlertMetrics
	Healthchecks map[string]*HealtcheckMetrics
	Checks       map[string]*CheckMetrics
}

var Metrics *MetricsCollection

func init() {
	Metrics = new(MetricsCollection)
	Metrics.Projects = make(map[string]*ProjectsMetrics)
	Metrics.Alerts = make(map[string]*AlertMetrics)
	Metrics.Checks = make(map[string]*CheckMetrics)
	Metrics.Healthchecks = make(map[string]*HealtcheckMetrics)
}

func InitMetrics() error {

	config.Log.Debug("Init metrics structures")

	for _, p := range config.Config.Projects {
		config.Log.Debugf("Init project %s metrics", p.Name)

		initProjectMetric(&p)

		for _, h := range p.Healtchecks {
			initHealtheckMetric(&h)
			for _, c := range h.Checks {
				initCheckMetric(&c)
			}
		}
	}

	for _, a := range config.Config.Alerts {
		initAlertMetrics(&a)
	}

	//j, _ := json.Marshal(Metrics)
	//config.Log.Debugf("Metrics: %+v", string(j))

	return nil
}
