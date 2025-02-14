package telegram

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
	"strconv"
)

var (
	configurer *config.TConfig
	logger     *logrus.Logger
)

func init() {
	configurer = config.GetConfig()
	logger = config.GetLog()
}

func NewAlerter(alertConfig config.TAlert) *TTelegramAlerter {
	channelID, _ := strconv.ParseInt(alertConfig.ProjectChannel, 10, 64)
	criticalChannelID, _ := strconv.ParseInt(alertConfig.CriticalChannel, 10, 64)

	return &TTelegramAlerter{
		Token:             alertConfig.BotToken,
		Log:               logger,
		channelID:         channelID,
		criticalChannelID: criticalChannelID,
	}
}
