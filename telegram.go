package main

import (
	"encoding/json"
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

	bot.Handle("/pause", func(m *tb.Message) {
		if m.IsReply() {
			jsonMessage, _ := json.Marshal(m.ReplyTo.Text)
			projectName := extractProject(jsonMessage)
			fmt.Printf("Pause req for project: %+v\n", projectName)
			err = ceaseProject(projectName)
			if err == nil {
				answer := fmt.Sprintf("Messages ceased for project %v", projectName)
				bot.Send(m.Sender, answer)
			}
		} else {
			Config.Defaults.Parameters.Mode = "quiet"
			answer := "All messages ceased"
			bot.Send(m.Sender, answer)
		}
	})

	bot.Handle("/unpause", func(m *tb.Message) {
		if m.IsReply() {
			jsonMessage, _ := json.Marshal(m.ReplyTo.Text)
			projectName := extractProject(jsonMessage)
			fmt.Printf("Unpause req for project: %+v\n", projectName)
			err = enableProject(projectName)
			if err == nil {
				answer := fmt.Sprintf("Messages enabled for project %v", projectName)
				bot.Send(m.Sender, answer)
			}
		} else {
			Config.Defaults.Parameters.Mode = "loud"
			answer := "Messages reenabled"
			bot.Send(m.Sender, answer)
		}
	})

	bot.Start()
}

func postChannel(channelID int64, token, message string) error {
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
