package main

import (
	"encoding/json"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"regexp"
	"time"
)

type TgMessage struct {
	*tb.Message
}

func (m TgMessage) GetProject() string {
	var (
		result      []string
		projectName string
	)

	conf, _ := json.Marshal(m)
	log.Printf("Message: %+v\n\n", string(conf))

	if m.IsReply() {
		// try to get from reply
		pattern := regexp.MustCompile("roject: (.*)\n")
		result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		projectName = result[1]
	} else {
		projectName = m.Payload
	}

	if result == nil {
		fmt.Printf("Project extraction error.")
	} else {
		fmt.Printf("Project extracted: %v\n", projectName)
	}

	return projectName
}

func (m TgMessage) GetUUID() string {
	var (
		result []string
		uuid   string
	)
	fmt.Printf("message: %v\n", m.Text)

	if m.IsReply() {
		// try to get uuid from reply
		pattern := regexp.MustCompile("UUID: (.*)")
		result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		uuid = result[1]
	} else {
		uuid = m.Payload
	}

	if result == nil {
		fmt.Printf("UUID extraction error.")
	} else {
		fmt.Printf("UUID extracted: %v\n", uuid)
	}

	return uuid

	// WIP test and write error handling
}

func runListenTgBot(token string) {

	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 5 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
	}

	bot.Handle("/pa", func(m *tb.Message) {
		Config.Defaults.Parameters.Mode = "quiet"
		answer := "All messages ceased"
		bot.Send(m.Sender, answer)
	})

	bot.Handle("/ua", func(m *tb.Message) {
		Config.Defaults.Parameters.Mode = "loud"
		answer := "All messages enabled"
		bot.Send(m.Sender, answer)
	})

	bot.Handle("/pu", func(m *tb.Message) {
		var tgMessage IncomingChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/pu")

		uuID := tgMessage.GetUUID()
		log.Printf("Pause req for UUID: %+v\n", uuID)
		for _, project := range Config.Projects {
			for _, healthcheck := range project.Healtchecks {
				for _, check := range healthcheck.Checks {
					if uuID == check.uuID {
						_ = check.CeaseAlerts()
					}
				}
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages ceased for UUID %v", uuID)
			bot.Send(m.Sender, answer)
		}
	})

	bot.Handle("/uu", func(m *tb.Message) {
		var tgMessage IncomingChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/uu")

		uuID := tgMessage.GetUUID()
		log.Printf("Resume req for UUID: %+v\n", uuID)
		for _, project := range Config.Projects {
			for _, healthcheck := range project.Healtchecks {
				for _, check := range healthcheck.Checks {
					if uuID == check.uuID {
						_ = check.EnableAlerts()
					}
				}
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages resumed for UUID %v", uuID)
			bot.Send(m.Sender, answer)
		}

	})

	bot.Handle("/pp", func(m *tb.Message) {
		var tgMessage IncomingChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/pp")

		projectName := tgMessage.GetProject()
		log.Printf("Pause req for project: %s\n", projectName)
		for _, project := range Config.Projects {
			if projectName == project.Name {
				_ = project.CeaseAlerts()
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages ceased for project %s", projectName)
			bot.Send(m.Sender, answer)
		}

	})

	bot.Handle("/up", func(m *tb.Message) {
		var tgMessage IncomingChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/up")

		projectName := tgMessage.GetProject()
		log.Printf("Resume req for project: %s\n", projectName)
		for _, project := range Config.Projects {
			if projectName == project.Name {
				_ = project.EnableAlerts()
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages resumed for project %s", projectName)
			bot.Send(m.Sender, answer)
		}

	})

	bot.Start()
}

func initBots() {
	var alert ChatAlert

	for _, alert = range Config.Alerts {
		if alert.GetName() == Config.Defaults.Parameters.CommandChannel {
			switch alert.GetType() {
			case "telegram":
				go runListenTgBot(alert.GetCreds())
			default:
				log.Panic("Command channel type not supported")
			}

		}
	}
}

func sendTgMessage(alerttype string, a *AlertConfigs, e error) error {
	log.Debugf("Alert send: %s (alert details %+v)", e, a)
	bot, err := tb.NewBot(tb.Settings{
		Token:  a.BotToken,
		Poller: &tb.LongPoller{Timeout: 5 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}
	user := tb.Chat{ID: a.ProjectChannel}
	log.Debugf("Alert to user: %+v with token %s, error: %+v", user, a.BotToken, e)

	_, err = bot.Send(&user, e.Error())
	if err != nil {
		log.Warnf("sendTgMessage error: %v", err)
	} else {
		log.Debugf("sendTgMessage success")
		addAlertCounter(alerttype, a)
	}
	return err
}
