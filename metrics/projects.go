package metrics

import "my/checker/config"

func AddProjectMetricCriticalAlert(p *config.Project) {
	AlertsCount.WithLabelValues(p.Name, "Critical").Inc()
	ProjectAlerts.WithLabelValues(p.Name, "Critical").Inc()
}

func AddProjectMetricChatOpsAnswer(p *config.Project) {
	AlertsCount.WithLabelValues(p.Name, "ChatOps_Message").Inc()
	ProjectAlerts.WithLabelValues(p.Name, "ChatOps_Message").Inc()
}
