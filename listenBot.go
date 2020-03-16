package main

import (
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func runListenBot(token string) {

	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
	}

	bot.Handle("/pause", func(m *tb.Message) {
		config.Mode = "quiet"
		answer := "Messages ceased"
		bot.Send(m.Sender, answer)
	})

	bot.Handle("/unpause", func(m *tb.Message) {
		config.Mode = "loud"
		answer := "Messages reenabled"
		bot.Send(m.Sender, answer)
	})

	bot.Start()
}
