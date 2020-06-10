package metrics

import (
	"my/checker/config"
)

func AddAlertMetricChatOpsRequest(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "ChatOps_Request").Inc()
}

func AddAlertMetricNonCritical(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "NonCritical").Inc()
}

func AddAlertMetricCritical(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "Critical").Inc()
}

func AddAlertMetricReport(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "Report").Inc()
}
