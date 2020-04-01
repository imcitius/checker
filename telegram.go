package main

import (
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"regexp"
	"time"
)

type TgMessage struct {
	*tb.Message
}

func (m TgMessage) GetProject() string {
	var projectName string

	fmt.Printf("message: %v\n", m)
	pattern := regexp.MustCompile("roject: (.*)\n")
	result := pattern.FindStringSubmatch(m.ReplyTo.Text)
	if result == nil {
		fmt.Printf("Project extraction error.")
	} else {
		fmt.Printf("Project extracted: %v\n", result[1])
		projectName = result[1]
	}

	return projectName
}

func (m TgMessage) GetUUID() string {
	var uuid string

	fmt.Printf("message: %v\n", m)
	pattern := regexp.MustCompile("UUID: (.*)")
	result := pattern.FindStringSubmatch(m.ReplyTo.Text)
	if result == nil {
		fmt.Printf("UUID extraction error.")
	} else {
		fmt.Printf("UUID extracted: %v\n", result[1])
		uuid = result[1]
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
		var tgMessage ChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/pu")

		if m.IsReply() {
			uuID := tgMessage.GetUUID()
			log.Printf("Pause req for UUID: %+v\n", uuID)
			for _, project := range Config.Projects {
				for _, check := range project.Checks {
					if uuID == check.uuID {
						_ = check.CeaseAlerts()
					}
				}
			}
			if err == nil {
				answer := fmt.Sprintf("Messages ceased for UUID %v", uuID)
				bot.Send(m.Sender, answer)
			}
		}

	})

	bot.Handle("/uu", func(m *tb.Message) {
		var tgMessage ChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/uu")

		if m.IsReply() {
			uuID := tgMessage.GetUUID()
			log.Printf("Resume req for UUID: %+v\n", uuID)
			for _, project := range Config.Projects {
				for _, check := range project.Checks {
					if uuID == check.uuID {
						_ = check.EnableAlerts()
					}
				}
			}
			if err == nil {
				answer := fmt.Sprintf("Messages resumed for UUID %v", uuID)
				bot.Send(m.Sender, answer)
			}
		}

	})

	bot.Handle("/pp", func(m *tb.Message) {
		var tgMessage ChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/pp")

		if m.IsReply() {
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
		}

	})

	bot.Handle("/up", func(m *tb.Message) {
		var tgMessage ChatMessage
		tgMessage = TgMessage{m}

		log.Printf("/up")

		if m.IsReply() {
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

func sendTgMessage(a Alerts, e error) error {
	//log.Printf("Alert send: %s (alert details %+v)", e, a)
	bot, err := tb.NewBot(tb.Settings{
		Token:  a.BotToken,
		Poller: &tb.LongPoller{Timeout: 5 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}
	user := tb.Chat{ID: a.ProjectChannel}
	//log.Printf("Alert to user: %+v with token %s, error: %+v", user, a.BotToken, e)

	_, err = bot.Send(&user, e.Error())
	if err != nil {
		log.Fatal(err)
	}
	return err
}
