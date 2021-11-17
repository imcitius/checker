package alerts

import (
	"fmt"
	tb "gopkg.in/tucnak/telebot.v3"
	"my/checker/config"
	"my/checker/metrics"
	"my/checker/status"
)

func quHandler(c tb.Context) error {
	//a.AddAlertMetricChatOpsRequest()

	tgMessage := TgMessage{c.Message()}
	uuID, err := tgMessage.GetUUID()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return err
	}

	//config.Log.Infof("Bot request /pu")
	config.Log.Printf("Pause req for UUID: %+v\n", uuID)
	err = status.SetCheckMode(config.GetCheckByUUID(uuID), "quiet")
	if err != nil {
		config.Log.Errorf("Error change check's status: %s", err)
	}

	SendChatOps(fmt.Sprintf("@%s Messages ceased for UUID %v\n", c.Sender().Username, uuID))

	return nil
}

func luHandler(c tb.Context) error {
	//a.AddAlertMetricChatOpsRequest()
	tgMessage := TgMessage{c.Message()}
	uuID, err := tgMessage.GetUUID()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return err
	}
	//config.Log.Infof("Bot request /uu")
	config.Log.Infof("Unpause req for UUID: %+v\n", uuID)
	err = status.SetCheckMode(config.GetCheckByUUID(uuID), "loud")
	if err != nil {
		config.Log.Errorf("Error change check's status: %s", err)
	}

	SendChatOps(fmt.Sprintf("@%s Messages resumed for UUID %v", c.Sender().Username, uuID))

	return nil
}

func qpHandler(c tb.Context) error {
	//a.AddAlertMetricChatOpsRequest()
	tgMessage := TgMessage{c.Message()}
	config.Log.Infof("Bot request /pp")
	projectName, err := tgMessage.GetProject()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return err
	}
	status.Statuses.Projects[projectName].Mode = "quiet"
	config.Log.Infof("Pause req for project: %s\n", projectName)
	SendChatOps(fmt.Sprintf("@%s Messages ceased for project %s", c.Sender().Username, projectName))

	return nil
}

func lpHandler(c tb.Context) error {
	//a.AddAlertMetricChatOpsRequest()
	tgMessage := TgMessage{c.Message()}
	config.Log.Infof("Bot request /up")
	projectName, err := tgMessage.GetProject()
	if err != nil {
		SendChatOps(fmt.Sprintf("%s", err))
		return err
	}
	config.Log.Infof("Resume req for project: %s\n", projectName)
	status.Statuses.Projects[projectName].Mode = "loud"
	SendChatOps(fmt.Sprintf("@%s Messages resumed for project %s", c.Sender().Username, projectName))

	return nil
}

func qaHandler(c tb.Context) error {
	config.Log.Infof("Bot request /qa")
	status.MainStatus = "quiet"
	SendChatOps(fmt.Sprintf("@%s All messages ceased", c.Sender().Username))

	return nil
}

func laHandler(c tb.Context) error {
	config.Log.Infof("Bot request /ua")
	status.MainStatus = "loud"
	SendChatOps(fmt.Sprintf("@%s All messages enabled", c.Sender().Username))

	return nil
}

func statsHandler(c tb.Context) error {
	config.Log.Infof("Bot request /stats from %s", c.Sender().Username)
	SendChatOps(fmt.Sprintf("@%s\n\n%v", c.Sender().Username, metrics.GenTextRuntimeStats()))

	return nil
}

func versionHandler(c tb.Context) error {
	config.Log.Infof("Bot request /version from %s", c.Sender().Username)
	SendChatOps(fmt.Sprintf("@%s\n%s, %s, %s\n", c.Sender().Username, config.Version, config.VersionSHA, config.VersionBuild))

	return nil
}
