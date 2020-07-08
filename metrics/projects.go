package metrics

import (
	project "my/checker/projects"
)

func AddProjectMetricCriticalAlert(p *project.Project) {
	AlertsCount.WithLabelValues(p.Name, "Critical").Inc()
	ProjectAlerts.WithLabelValues(p.Name, "Critical").Inc()
}

func AddProjectMetricChatOpsAnswer(p *project.Project) {
	AlertsCount.WithLabelValues(p.Name, "ChatOps_Message").Inc()
	ProjectAlerts.WithLabelValues(p.Name, "ChatOps_Message").Inc()
}
