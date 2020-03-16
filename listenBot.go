package main

import (
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func runListenBot(token string) {

	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
	}

	b.Handle("/ttt", func(m *tb.Message) {
		answer := "yes"
		b.Send(m.Sender, answer)
	})

	// b.Handle("/test", func(m *tb.Message) {
	// 	jsonM, _ := json.Marshal(m.Sender)
	// 	fmt.Println(string(jsonM))
	// 	answer := "printed"
	// 	b.Send(m.Sender, answer)
	// })

	b.Start()
}
