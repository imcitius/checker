package go_telegram

import (
	"context"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

type TTelegramAlerter struct {
	Token    string
	bot      *tele.Bot
	context  *context.Context
	settings tele.Settings
	Log      *logrus.Logger

	channelID         int64
	criticalChannelID int64
}
