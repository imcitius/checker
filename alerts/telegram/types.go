package telegram

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/sirupsen/logrus"
)

type TTelegramAlerter struct {
	Token   string
	bot     *bot.Bot
	context *context.Context
	opts    *[]bot.Option
	Log     *logrus.Logger
}
