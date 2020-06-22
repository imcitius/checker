package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// to count check errors and make alerting decisions
	ProjectErrorStatus map[string]map[string]int

	SchedulerLoops = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sched_loops_count",
			Help: "Scheduler loops count",
		})

	SchedulerChecksDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sched_checks_duration",
			Help: "Scheduler checks loop duration",
		})

	SchedulerReportsDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sched_reports_duration",
			Help: "Scheduler reports loop duration",
		})

	SchedulerAlertsDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sched_alerts_duration",
			Help: "Scheduler alerts loop duration",
		})

	SchedulerLoopConfig = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sched_loop_config",
			Help: "Scheduler loop duration configured",
		})

	AlertsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_by_event_type",
			Help: "How many messages of different type sent for alert channel.",
		},
		[]string{"alert_name", "event_type"},
	)

	ProjectAlerts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_by_project",
			Help: "How many events of different type occured for specific project.",
		},
		[]string{"project_name", "event_type"},
	)

	CheckMetrics = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_by_check",
			Help: "How many events of different type occured sent for specific check.",
		},
		[]string{"project_name", "healthcheck_name", "check_uuid", "event_type"},
	)

	CheckDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "check_duration",
			Help: "How much time specific check executes",
		},
		[]string{"project_name", "healthcheck_name", "check_uuid", "check_type"},
	)
)

func init() {

	ProjectErrorStatus = make(map[string]map[string]int)

	prometheus.MustRegister(SchedulerLoops)
	prometheus.MustRegister(SchedulerChecksDuration)
	prometheus.MustRegister(SchedulerAlertsDuration)
	prometheus.MustRegister(SchedulerReportsDuration)
	prometheus.MustRegister(SchedulerLoopConfig)
	prometheus.MustRegister(AlertsCount)
	prometheus.MustRegister(ProjectAlerts)
	prometheus.MustRegister(CheckMetrics)
	prometheus.MustRegister(CheckDuration)
}
