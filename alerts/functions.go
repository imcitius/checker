package alerts

import (
	"fmt"
	"my/checker/config"
	"my/checker/metrics"
)

func GetAlertProto(a *config.AlertConfigs) Alerter {
	if value, ok := AlerterCollections[a.Type]; ok {
		return value
	}
	return nil
}

func GetCommandChannel() (*config.AlertConfigs, error) {
	for _, a := range config.Config.Alerts {
		if a.Name == config.Config.Defaults.Parameters.CommandChannel {
			config.Log.Debugf("Found command channel: %v", a.ProjectChannel)
			return &a, nil
		}
	}
	return nil, fmt.Errorf("ChatOpsChannel not found: %s", config.Config.Defaults.Parameters.CommandChannel)
}

func SendChatOps(text string) {
	//metrics.AddProjectMetricChatOpsAnswer(&projects.Project{
	//	Name: "ChatOps"})
	commandChannel, err := GetCommandChannel()
	if err != nil {
		config.Log.Infof("GetCommandChannel error: %s", err)
	}

	err = Alert(commandChannel, text, "chatops")
	if err != nil {
		config.Log.Infof("SendTgChatOpsMessage error: %s", err)
	}
}

func Alert(a *config.AlertConfigs, text, messageType string) error {
	metrics.AddAlertMetricNonCritical(a)

	err := GetAlertProto(a).Send(a, text, messageType)
	if err != nil {
		config.Log.Infof("Send error")
	}
	return err
}
