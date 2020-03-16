package main

import (
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func postChannel(channelID int, token, text string) error {
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}
	user := tb.User{ID: channelID}
	b.Send(&user, text)

	return nil

}
