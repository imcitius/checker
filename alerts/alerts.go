package alerts

import (
	"my/checker/config"
)

var (
	botsSignalCh chan bool
)

func GetAlertName(a *config.AlertConfigs) string {
	return a.Name
}

func GetAlertType(a *config.AlertConfigs) string {
	return a.Type
}

func GetAlertCreds(a *config.AlertConfigs) string {
	return a.BotToken
}