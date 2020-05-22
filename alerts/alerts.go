package alerts

import (
	"errors"
	"fmt"
	"my/checker/config"
	"my/checker/telegram"
)

func SendAlert(a *config.AlertConfigs, alerttype string, e error) error {

	switch a.Type {
	case "telegram":
		err := telegram.SendTgMessage(alerttype, a, e)
		return err
	default:
		err := errors.New(fmt.Sprintf("Not implemented bot type %s, name %s", a.Type, a.Name))
		return err
	}
}

func GetAlertName(a *config.AlertConfigs) string {
	return a.Name
}

func GetAlertType(a *config.AlertConfigs) string {
	return a.Type
}

func GetAlertCreds(a *config.AlertConfigs) string {
	return a.BotToken
}

func InitBots() {

	for _, alert := range config.Config.Alerts {
		if GetAlertName(&alert) == config.Config.Defaults.Parameters.CommandChannel {
			switch GetAlertType(&alert) {
			case "telegram":
				go telegram.RunListenTgBot(GetAlertCreds(&alert))
			default:
				config.Log.Panic("Command channel type not supported")
			}

		}
	}
}
