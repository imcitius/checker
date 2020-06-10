package metrics

import (
	"my/checker/config"
)

func AddAlertMetricChatOpsRequest(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "ChatOps_Request").Inc()
	AlertsHistory.WithLabelValues(a.Name, "ChatOps_Request").Observe(1)
}

func AddAlertMetricNonCritical(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "NonCritical").Inc()
	AlertsHistory.WithLabelValues(a.Name, "NonCritical").Observe(1)
}

func AddAlertMetricCritical(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "Critical").Inc()
	AlertsHistory.WithLabelValues(a.Name, "Critical").Observe(1)
}

func AddAlertMetricReport(a *config.AlertConfigs) {
	AlertsCount.WithLabelValues(a.Name, "Report").Inc()
	AlertsHistory.WithLabelValues(a.Name, "Report").Observe(1)
}

//
//func AddAlertMetricOther(a *config.AlertConfigs) {
//	AlertsCount.WithLabelValues(a.Name, "Other").Inc()
//}
//
//
//func AddAlertMetricChatOpsAnswer(a *config.AlertConfigs) {
//	AlertsCount.WithLabelValues(a.Name, "ChatOps_Answer").Inc()
//}
