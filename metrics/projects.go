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

//
//func AddProjectMetricChatOpsMessage(p *config.Project) {
//	AlertsCount.WithLabelValues(p.Name, "ChatOps_Message").Inc()
//}
//
//func AddProjectMetricRun(p *config.Project) error {
//	ProjectAlerts.WithLabelValues(p.Name, "Run").Inc()
//
//	return nil
//}
//
//func AddProjectMetricError(p *config.Project) error {
//	ProjectAlerts.WithLabelValues(p.Name, "Error").Inc()
//
//	ProjectErrorStatus[p.Name]["Error"]++
//
//	return nil
//}
//
//func AddProjectMetricFail(p *config.Project) error {
//	ProjectAlerts.WithLabelValues(p.Name, "Fail").Inc()
//
//	ProjectErrorStatus[p.Name]["Fail"]++
//
//	return nil
//}
//
//func AddProjectMetricReportAlert(p *config.Project) {
//	AlertsCount.WithLabelValues(p.Name, "Report").Inc()
//}
//
//func AddProjectMetricOtherAlert(p *config.Project) {
//	AlertsCount.WithLabelValues(p.Name, "Other").Inc()
//}
//
//func AddProjectMetricChatOpsRequest(p *config.Project) {
//	AlertsCount.WithLabelValues(p.Name, "ChatOps_Request").Inc()
//}
