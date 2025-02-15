package checks

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
)

var (
	configurer *config.TConfig
	logger     *logrus.Logger
	//cache      *memoize.Memoizer
)

func InitChecks() {
	configurer = config.GetConfig()
	logger = config.GetLog()
	//cache = memoize.NewMemoizer(24*time.Hour, 24*time.Hour)

	if len(configurer.Projects) == 0 {
		logger.Fatalf("No projects found in config")
	}
}
