package config

import (
	"fmt"
	"github.com/kofalt/go-memoize"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	logger = logrus.New()
	cache  *memoize.Memoizer
	wg     sync.WaitGroup
)

func InitConfig(cfgFile string) {
	initConfig(cfgFile)
	cache = memoize.NewMemoizer(24*time.Hour, 24*time.Hour)
	config.Defaults.DefaultCheckParameters.Duration = config.Defaults.Duration
	config.refineProjects()
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

func GetProjectByName(name string) (TProject, error) {
	result, err, _ := cache.Memoize(fmt.Sprintf("projectByName-%s", name), func() (interface{}, error) {
		return findProjectByName(name)
	})
	return result.(TProject), err
}

func findProjectByName(name string) (TProject, error) {
	for _, p := range config.Projects {
		if p.Name == name {
			return p, nil
		}
	}
	return TProject{}, fmt.Errorf("project not found")
}

func GetWG() *sync.WaitGroup {
	return &wg
}
