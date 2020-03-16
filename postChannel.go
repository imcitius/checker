package main

import (
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func postChannel(channelID int, token, text string) error {
	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}
	user := tb.User{ID: channelID}
	if config.Mode == "quiet" {
		log.Println("Loud mode: ", text)
		bot.Send(&user, text)
	} else {
		log.Println("Quiet mode: ", text)
	}

	return nil

}
