package config

import (
	"github.com/sirupsen/logrus"
)

var (
	logger = logrus.New()
)

func InitConfig(cfgFile string) {
	initCleanenv(cfgFile)
}

func InitLog(logLevel string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(level)
	if logger.GetLevel() == 5 {
		logger.SetReportCaller(true)
	}
}

func GetConfig() *TConfig {
	return &config
}

func GetLog() *logrus.Logger {
	return logger
}
