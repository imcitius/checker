package alerts

import (
	"fmt"
	"my/checker/config"
	"my/checker/metrics"
)

type AlertConfigs struct {
	config.AlertConfigs
}

func (a *AlertConfigs) GetAlertProto() Alerter {
	if value, ok := AlerterCollections[a.Type]; ok {
		return value
	}
	return nil
}

func GetCommandChannel() (*AlertConfigs, error) {
	for _, a := range config.Config.Alerts {
		if a.Name == config.Config.Defaults.Parameters.CommandChannel {
			config.Log.Debugf("Found command channel: %v", a.ProjectChannel)
			return &AlertConfigs{a}, nil
		}
	}
	return nil, fmt.Errorf("ChatOpsChannel not found: %s", config.Config.Defaults.Parameters.CommandChannel)
}

func SendChatOps(text string, args ...interface{}) {
	//metrics.AddProjectMetricChatOpsAnswer(&projects.Project{
	//	Name: "ChatOps"})
	commandChannel, err := GetCommandChannel()
	if err != nil {
		config.Log.Infof("GetCommandChannel error: %s", err)
	}

	err = commandChannel.Alert(text, "chatops", args)
	if err != nil {
		config.Log.Infof("SendTgChatOpsMessage error: %s", err)
	}
}

func (a *AlertConfigs) Alert(text, messageType string, args ...interface{}) error {
	allowMetrics := true

	if args != nil {
		for _, arg := range args {
			val, _ := arg.(string)
			if val == "noMetrics" {
				allowMetrics = false
			}
		}
	}

	if allowMetrics == true {
		a.AddAlertMetricNonCritical()
	}

	err := a.GetAlertProto().Send(a, text, messageType)
	if err != nil {
		config.Log.Infof("Send error")
	}
	return err
}

func (a *AlertConfigs) AddAlertMetricChatOpsRequest() {
	var alertname string

	if a != nil {
		alertname = "log"
	} else {
		alertname = a.Name
	}
	metrics.AlertsCount.WithLabelValues(alertname, "ChatOps_Request").Inc()
}

func (a *AlertConfigs) AddAlertMetricNonCritical() {
	var alertname string

	if a != nil {
		alertname = "log"
	} else {
		alertname = a.Name
	}
	metrics.AlertsCount.WithLabelValues(alertname, "NonCritical").Inc()
}

func (a *AlertConfigs) AddAlertMetricCritical() {
	var alertname string

	if a != nil {
		alertname = "log"
	} else {
		alertname = a.Name
	}
	metrics.AlertsCount.WithLabelValues(alertname, "Critical").Inc()

}
