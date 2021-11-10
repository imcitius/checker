package alerts

import (
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"my/checker/config"
	"my/checker/metrics"
	"my/checker/misc"
	"my/checker/status"
)

func puHandler(m *tb.Message, a *AlertConfigs) {
	a.AddAlertMetricChatOpsRequest()

	tgMessage := TgMessage{m}
	uuID, err := tgMessage.GetUUID()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}

	//config.Log.Infof("Bot request /pu")
	config.Log.Printf("Pause req for UUID: %+v\n", uuID)
	status.SetCheckMode(misc.GetCheckByUUID(uuID), "quiet")

	SendChatOps(fmt.Sprintf("@%s Messages ceased for UUID %v\n", m.Sender.Username, uuID))
}

func uuHandler(m *tb.Message, a *AlertConfigs) {

	a.AddAlertMetricChatOpsRequest()

	tgMessage := TgMessage{m}
	uuID, err := tgMessage.GetUUID()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}
	//config.Log.Infof("Bot request /uu")
	config.Log.Infof("Unpause req for UUID: %+v\n", uuID)
	status.SetCheckMode(misc.GetCheckByUUID(uuID), "loud")

	SendChatOps(fmt.Sprintf("@%s Messages resumed for UUID %v", m.Sender.Username, uuID))
}

func ppHandler(m *tb.Message, a *AlertConfigs) {
	a.AddAlertMetricChatOpsRequest()

	tgMessage := TgMessage{m}

	config.Log.Infof("Bot request /pp")
	projectName, err := tgMessage.GetProject()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}

	status.Statuses.Projects[projectName].Mode = "quiet"
	config.Log.Infof("Pause req for project: %s\n", projectName)

	SendChatOps(fmt.Sprintf("@%s Messages ceased for project %s", m.Sender.Username, projectName))
}

func upHandler(m *tb.Message, a *AlertConfigs) {

	a.AddAlertMetricChatOpsRequest()

	tgMessage := TgMessage{m}

	config.Log.Infof("Bot request /up")

	projectName, err := tgMessage.GetProject()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return
	}

	config.Log.Infof("Resume req for project: %s\n", projectName)
	status.Statuses.Projects[projectName].Mode = "loud"

	SendChatOps(fmt.Sprintf("@%s Messages resumed for project %s", m.Sender.Username, projectName))
}

func paHandler(m *tb.Message) {
	config.Log.Infof("Bot request /qa")

	status.MainStatus = "quiet"
	SendChatOps(fmt.Sprintf("@%s All messages ceased", m.Sender.Username))

}

func uaHandler(m *tb.Message) {
	config.Log.Infof("Bot request /ua")

	status.MainStatus = "loud"
	SendChatOps(fmt.Sprintf("@%s All messages enabled", m.Sender.Username))
}

func statsHandler(m *tb.Message) {
	config.Log.Infof("Bot request /stats from %s", m.Sender.Username)

	SendChatOps(fmt.Sprintf("@%s\n\n%v", m.Sender.Username, metrics.GenTextRuntimeStats()))
}
