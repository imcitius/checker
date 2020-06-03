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

func GetAlertProto(a *config.AlertConfigs) Alerter {
	switch a.Type {
	case "mattermost":
		return new(Mattermost)
	case "telegram":
		return new(Telegram)
	}
	return nil
}

func GetCommandChannel() *config.AlertConfigs{
	for _, a := range config.Config.Alerts {
		if a.Name == config.Config.Defaults.Parameters.CommandChannel {
			return &a
		}
	}
	return nil
}

func GetProjectChannel(p *config.Project) *config.AlertConfigs{
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.AlertChannel {
			return &a
		}
	}
	return nil
}

func GetCritChannel(p *config.Project) *config.AlertConfigs{
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.CritAlertChannel {
			return &a
		}
	}
	return nil
}

func Send(p *config.Project, text string) {
	metrics.Metrics.Alerts[GetProjectChannel(p).Name].CommandAns++
	err := Alert(GetProjectChannel(p), text)
	if err != nil {
		config.Log.Infof("SendTgMessage error: %s", err)
	}
}

func SendCrit(p *config.Project, text string) {
	metrics.Metrics.Alerts[GetCritChannel(p).Name].CommandAns++
	err := Alert(GetCritChannel(p), text)
	if err != nil {
		config.Log.Infof("SendTgMessage error: %s", err)
	}
}

func SendChatOps(text string) {
	metrics.Metrics.Alerts[GetCommandChannel().Name].CommandAns++
	err := Alert(GetCommandChannel(), text)
	if err != nil {
		config.Log.Infof("SendTgMessage error: %s", err)
	}
}

//func InitBots(ch chan bool, wg *sync.WaitGroup) {
//
//		GetAlertProto(GetCommandChannel()).InitBot(ch, wg)
//}

func Alert(a *config.AlertConfigs, text string) error {
	err := GetAlertProto(a).Send(a, text)
	if err != nil {
		config.Log.Infof("Send error")
	}
	return err
}

