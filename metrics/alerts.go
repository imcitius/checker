package metrics

import "my/checker/config"

func AddAlertCounter(a *config.AlertConfigs, alerttype string) {
	config.Log.Debugf("increase alert counter")

	for _, counter := range config.Config.Alerts {
		config.Log.Debugf("%s .... %s", a.Name, counter.Name)
		if a.Name == counter.Name {
			config.Log.Debugf("Increase total %s counter %s", alerttype, counter.Name)
			switch alerttype {
			case "crit":
				counter.Critical++
			case "noncrit":
				counter.NonCritical++
			case "report":
				counter.NonCritical++
			default:
				config.Log.Errorf("Undefined alert type")
			}
			counter.AlertCount++
		}
	}
}

