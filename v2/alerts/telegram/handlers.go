package telegram

import (
	tele "gopkg.in/telebot.v3"
)

func qaHandler(c tele.Context) error {
	logger.Infof("Bot request /qa")
	//status.MainStatus = "quiet"
	//SendChatOps(fmt.Sprintf("@%s All messages ceased", c.Sender().Username))

	return nil
}

func quHandler(c tele.Context) error {
	//a.AddAlertMetricChatOpsRequest()
	logger.Infof("Bot request /pu")
	logger.Infof("%#v", messagesContext.GetData())
	//value := c.Get("testField")
	//c.Reply(value.(string))

	//tgMessage := TgMessage{c.Message()}
	//uuID, err := tgMessage.GetUUID()
	//if err != nil {
	//	SendChatOps(fmt.Sprintf("%s", err))
	//	return err
	//}
	//
	////logger.Printf("Pause req for UUID: %+v\n", uuID)
	//err = status.SetCheckMode(config.GetCheckByUUID(uuID), "quiet")
	//if err != nil {
	//	config.Log.Errorf("Error change check's status: %s", err)
	//}
	//
	//SendChatOps(fmt.Sprintf("@%s Messages ceased for UUID %v\n", c.Sender().Username, uuID))

	return nil
}

func qpHandler(c tele.Context) error {
	//a.AddAlertMetricChatOpsRequest()
	logger.Infof("Bot request /pp")
	//value := c.Get("testField2")
	//c.Reply(value.(string))

	//tgMessage := TgMessage{c.Message()}
	//projectName, err := tgMessage.GetProject()
	//if err != nil {
	//	SendChatOps(fmt.Sprintf("%s", err))
	//	return err
	//}
	//status.Statuses.Projects[projectName].Mode = "quiet"
	//config.Log.Infof("Pause req for project: %s\n", projectName)
	//SendChatOps(fmt.Sprintf("@%s Messages ceased for project %s", c.Sender().Username, projectName))

	return nil
}
