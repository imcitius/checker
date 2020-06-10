package alerts

import (
	"my/checker/config"
	"my/checker/metrics"
	"sync"
)

type Alerter interface {
	Send(a *config.AlertConfigs, message string) error
	InitBot(botsSignalCh chan bool, wg *sync.WaitGroup)
}

var AlerterCollection map[string]Alerter

func GetAlertProto(a *config.AlertConfigs) Alerter {
	return AlerterCollection[a.Type]
}

func GetCommandChannel() *config.AlertConfigs {
	for _, a := range config.Config.Alerts {
		if a.Name == config.Config.Defaults.Parameters.CommandChannel {
			return &a
		}
	}
	return nil
}

func GetProjectChannel(p *config.Project) *config.AlertConfigs {
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.AlertChannel {
			return &a
		}
	}
	return nil
}

func GetCritChannel(p *config.Project) *config.AlertConfigs {
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.CritAlertChannel {
			return &a
		}
	}
	return nil
}

func Send(p *config.Project, text string) {
	metrics.AddProjectMetricChatOpsAnswer(p)

	err := Alert(GetProjectChannel(p), text)
	if err != nil {
		config.Log.Infof("SendTgMessage error: %s", err)
	}
}

func SendCrit(p *config.Project, text string) {
	metrics.AddProjectMetricCriticalAlert(p)
	metrics.AddAlertMetricCritical(GetCritChannel(p))

	err := Alert(GetCritChannel(p), text)
	if err != nil {
		config.Log.Infof("SendTgMessage error: %s", err)
	}
}

func SendChatOps(text string) {
	metrics.AddProjectMetricChatOpsAnswer(&config.Project{
		Name: "ChatOps"})

}

func Alert(a *config.AlertConfigs, text string) error {
	metrics.AddAlertMetricNonCritical(a)

	err := GetAlertProto(a).Send(a, text)
	if err != nil {
		config.Log.Infof("Send error")
	}
	return err
}
