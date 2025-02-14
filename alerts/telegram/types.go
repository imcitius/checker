package telegram

import (
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

type TTelegramAlerter struct {
	Token    string
	bot      *tele.Bot
	settings tele.Settings
	options  tele.SendOptions
	Log      *logrus.Logger

	channelID         int64
	criticalChannelID int64
}
