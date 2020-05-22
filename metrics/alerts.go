package metrics

import "my/checker/config"

func initAlertMetrics(a *config.AlertConfigs) {
	if _, ok := Metrics.Alerts[a.Name]; !ok {
		Metrics.Alerts[a.Name] = new(AlertMetrics)
		Metrics.Alerts[a.Name].Name = a.Name
	}
}

func AddAlertCounter(a *config.AlertConfigs, alerttype string) {
	config.Log.Debugf("increase alert counter")

	switch alerttype {
	case "crit":
		Metrics.Alerts[a.Name].Critical++
	case "noncrit":
		Metrics.Alerts[a.Name].NonCritical++
	case "report":
		Metrics.Alerts[a.Name].NonCritical++
	default:
		config.Log.Errorf("Undefined alert type")
	}
	Metrics.Alerts[a.Name].AlertCount++
}
