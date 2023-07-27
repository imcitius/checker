package go_telegram

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
)

var (
	//configurer *config.TConfig
	logger *logrus.Logger
)

func init() {
	//configurer = config.GetConfig()
	logger = config.GetLog()
}
