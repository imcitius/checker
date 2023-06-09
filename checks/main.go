package checks

import (
	"github.com/sirupsen/logrus"
	"my/checker/config"
)

var (
	configurer *config.TConfig
	logger     *logrus.Logger
)

func InitChecks() {
	//logger.Fatal(config.AllSettings())
	// check if projects are in config

	configurer = config.GetConfig()
	logger = config.GetLog()

	if len(configurer.Projects) == 0 {
		logger.Fatalf("Checks' init(): no projects found in config")
	}

	refineProjects()
	//projectsConfig.refineChecks()
}
