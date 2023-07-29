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

func (c *TConfig) GetCheckByUUid(uuid string) (TCheckConfig, error) {
	result, err, _ := cache.Memoize(fmt.Sprintf("checkByUuid-%s", uuid), func() (interface{}, error) {
		return findCheckByUuid(uuid)
	})
	return result.(TCheckConfig), err
}

func findCheckByUuid(uuid string) (TCheckConfig, error) {
	for _, p := range config.Projects {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				if c.UUid == uuid {
					return c, nil
				}
			}
		}
	}
	return TCheckConfig{}, fmt.Errorf("check not found")
}

func (c *TConfig) ListChecks() (interface{}, error) {
	result, err, _ := cache.Memoize(fmt.Sprintf("ListChecks"), func() (interface{}, error) {
		return listChecks()
	})
	return result, err
}

func listChecks() (interface{}, error) {
	var list map[string]map[string]map[string]string

	list = make(map[string]map[string]map[string]string)
	for _, p := range config.Projects {
		list[p.Name] = make(map[string]map[string]string)
		for _, h := range p.Healthchecks {
			list[p.Name][h.Name] = make(map[string]string)
			for _, c := range h.Checks {
				list[p.Name][h.Name][c.Name] = c.UUid
			}
		}
	}
	return list, nil
}
