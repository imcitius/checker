package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// to count check errors and make alerting decisions
	ProjectErrorStatus map[string]map[string]int

	SchedulerLoops = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sched_loops",
			Help: "Scheduler loops count",
		})

	AlertsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_by_event_type",
			Help: "How many messages of different type sent for alert channel.",
		},
		[]string{"alert_name", "event_type"},
	)

	ProjectAlerts = prometheus.NewCounterVec(
		prometheus.CounterOpts(prometheus.GaugeOpts(prometheus.CounterOpts{
			Name: "events_by_project",
			Help: "How many events of different type occurend for specific project.",
		})),
		[]string{"project_name", "event_type"},
	)

	CheckMetrics = prometheus.NewCounterVec(
		prometheus.CounterOpts(prometheus.GaugeOpts(prometheus.CounterOpts{
			Name: "events_by_check",
			Help: "How many events of different occurend sent for specific check.",
		})),
		[]string{"project_name", "healthcheck_name", "check_uuid", "event_type"},
	)
)

func init() {

	ProjectErrorStatus = make(map[string]map[string]int)

	prometheus.MustRegister(SchedulerLoops)
	prometheus.MustRegister(ProjectAlerts)
	prometheus.MustRegister(AlertsCount)
	prometheus.MustRegister(CheckMetrics)
}
