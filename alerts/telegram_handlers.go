package alerts

import (
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"my/checker/config"
	"my/checker/metrics"
	"my/checker/status"
)

func puHandler(m *tb.Message, a *config.AlertConfigs) {
	metrics.AddAlertMetricChatOpsRequest(a)

	var tgMessage config.IncomingChatMessage
	tgMessage = TgMessage{m}
	uuID, err := tgMessage.GetUUID()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}

	config.Log.Infof("Bot request /pu")
	config.Log.Printf("Pause req for UUID: %+v\n", uuID)
	status.SetCheckMode(getCheckByUUID(uuID), "quiet")

	SendChatOps(fmt.Sprintf("Messages ceased for UUID %v", uuID))
}

func ppHandler(m *tb.Message, a *config.AlertConfigs) {
	metrics.AddAlertMetricChatOpsRequest(a)

	var tgMessage config.IncomingChatMessage
	tgMessage = TgMessage{m}

	config.Log.Infof("Bot request /pp")
	projectName, err := tgMessage.GetProject()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}

	// TODO
	status.Statuses.Projects[projectName].Mode = "quiet"
	config.Log.Printf("Pause req for project: %s\n", projectName)

	SendChatOps(fmt.Sprintf("Messages ceased for project %s", projectName))
}

func paHandler() {
	config.Log.Infof("Bot request /pa")

	status.MainStatus = "quiet"
	SendChatOps("All messages ceased")

}

func uaHandler() {
	config.Log.Infof("Bot request /ua")

	status.MainStatus = "loud"
	SendChatOps("All messages enabled")
}

func uuHandler(m *tb.Message, a *config.AlertConfigs) {

	metrics.AddAlertMetricChatOpsRequest(a)

	var tgMessage config.IncomingChatMessage
	tgMessage = TgMessage{m}
	uuID, err := tgMessage.GetUUID()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}
	config.Log.Infof("Bot request /uu")
	config.Log.Printf("Unpause req for UUID: %+v\n", uuID)
	status.SetCheckMode(getCheckByUUID(uuID), "loud")

	SendChatOps(fmt.Sprintf("Messages resumed for UUID %v", uuID))
}

func getCheckByUUID(uuID string) *config.Check {
	for _, project := range config.Config.Projects {
		for _, healthcheck := range project.Healthchecks {
			for _, check := range healthcheck.Checks {
				if uuID == check.UUid {
					return &check
				}
			}
		}
	}
	return nil

}

func upHandler(m *tb.Message, a *config.AlertConfigs) {

	metrics.AddAlertMetricChatOpsRequest(a)

	var tgMessage config.IncomingChatMessage
	tgMessage = TgMessage{m}

	config.Log.Infof("Bot request /up")

	projectName, err := tgMessage.GetProject()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}

	config.Log.Printf("Resume req for project: %s\n", projectName)
	status.Statuses.Projects[projectName].Mode = "loud"

	SendChatOps(fmt.Sprintf("Messages resumed for project %s", projectName))
}

func statsHandler(m *tb.Message) {
	config.Log.Infof("Bot request /stats from %s", m.Sender.Username)

	SendChatOps(fmt.Sprintf("@" + m.Sender.Username + "\n\n" + metrics.GenTextRuntimeStats()))
}
