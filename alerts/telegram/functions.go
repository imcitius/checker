package telegram

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"os"
	"os/signal"
)

var (
	err error
)

func (a *TTelegramAlerter) Init() {
	//a.Log.Info("TTelegramAlerter Init")

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	//a.cancel = &cancel
	a.context = &ctx

	a.opts = &[]bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   update.Message.Text,
			})
		}),
		func() bot.Option {
			if a.Log.GetLevel() == 5 {
				return bot.WithDebug()
			}
			return bot.Option(func(b *bot.Bot) {})
		}(),
	}

	a.bot, err = bot.New(a.Token, *a.opts...)
	if err != nil {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}
}

func (a *TTelegramAlerter) Start() {
	//a.Log.Infof("TTelegramAlerter Start, %+v", a.Bot)
	if a.bot == nil {
		a.Init()
	}
	a.bot.Start(*a.context)
}

func (a *TTelegramAlerter) Stop() {
	a.bot.Close(*a.context)
}

func (a *TTelegramAlerter) Send(channel any, message string) {
	//a.Log.Info("TTelegramAlerter Send")
	if a.bot == nil {
		a.Init()
	}

	a.bot.SendMessage(*a.context, &bot.SendMessageParams{
		ChatID: channel,
		Text:   message,
	})
}
