package main

import (
	"fmt"
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func initBots() {
	go runListenBot(Config.Defaults.Parameters.BotToken)
}

func runListenBot(token string) {

	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
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
		answer := "Messages reenabled"
		bot.Send(m.Sender, answer)

	})

	bot.Handle("/pp", func(m *tb.Message) {
		if m.IsReply() {
			projectName := extractProject(m.ReplyTo.Text)
			fmt.Printf("Pause req for project: %+v\n", projectName)
			err = ceaseProject(projectName)
			if err == nil {
				answer := fmt.Sprintf("Messages ceased for project %v", projectName)
				bot.Send(m.Sender, answer)
			}
		} else {
			// WIP add return error text if not reply
			return
		}
	})

	bot.Handle("/up", func(m *tb.Message) {
		if m.IsReply() {
			projectName := extractProject(m.ReplyTo.Text)
			fmt.Printf("Unpause req for project: %+v\n", projectName)
			err = enableProject(projectName)
			if err == nil {
				answer := fmt.Sprintf("Messages enabled for project %v", projectName)
				bot.Send(m.Sender, answer)
			}
		} else {
			// WIP add return error text if not reply
			return
		}
	})

	bot.Handle("/pu", func(m *tb.Message) {
		if m.IsReply() {
			uuID := extractUUID(m.ReplyTo.Text)
			fmt.Printf("Pause req for UUID: %+v\n", uuID)
			err = ceaseUUID(uuID)
			if err == nil {
				answer := fmt.Sprintf("Messages ceased for UUID %v", uuID)
				bot.Send(m.Sender, answer)
			}
		} else {
			// WIP add return error text if not reply
			return
		}
	})

	bot.Handle("/uu", func(m *tb.Message) {
		if m.IsReply() {
			uuID := extractUUID(m.ReplyTo.Text)
			fmt.Printf("Unpause req for UUID: %+v\n", uuID)
			err = enableUUID(uuID)
			if err == nil {
				answer := fmt.Sprintf("Messages enabled for UUID %v", uuID)
				bot.Send(m.Sender, answer)
			}
		} else {
			// WIP add return error text if not reply
			return
		}
	})

	bot.Start()
}

func sendAlert(channelID int64, token, message string) error {

	// log.Printf("Sending alert: channel %d, token %s, message %s", channelID, token, message)
	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}
	user := tb.Chat{ID: channelID}

	bot.Send(&user, message)
	return nil

}
