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
	for _, p := range c.Projects {
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
	type listObject struct {
		UUID       string
		LastStatus bool
		LastExec   time.Time
		LastPing   time.Time
	}

	var list map[string]map[string]map[string]listObject

	list = make(map[string]map[string]map[string]listObject)
	for _, p := range c.Projects {
		list[p.Name] = make(map[string]map[string]listObject)
		for _, h := range p.Healthchecks {
			list[p.Name][h.Name] = make(map[string]listObject)
			for _, c := range h.Checks {
				list[p.Name][h.Name][c.Name] = listObject{
					UUID:       c.UUid,
					LastStatus: c.LastResult,
					LastExec:   c.LastExec,
					LastPing:   c.LastPing,
				}
			}
		}
	}
	return list, nil

}

func (c *TConfig) GetAllChecks() ([]TCheckConfig, error) {

	list := make([]TCheckConfig, 0)

	for _, p := range c.Projects {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				list = append(list, c)
			}
		}
	}
	return list, nil
}

func (c *TConfig) Ping(uuid string) (TCheckConfig, error) {

	check, _ := c.GetCheckByUUid(uuid)
	check.LastPing = time.Now()
	p := c.Projects
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = check
	return check, nil
}

func (c *TConfig) SetStatus(uuid string, status bool) {

	check, _ := c.GetCheckByUUid(uuid)
	check.LastExec = time.Now()
	check.LastResult = status
	p := c.Projects
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = check
}

func (c *TConfig) GetDBConnectionString() string {
	if c.DB.Protocol == "mongodb" {
		return fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority",
			c.DB.Username, c.DB.Password, c.DB.Host)
	}

	return ""
}

func (c *TConfig) SetDBPassword(password string) {
	c.DB.Password = password
}

func (c *TConfig) GetCheckDetails(uuid string) TCheckDetails {
	check, _ := c.GetCheckByUUid(uuid)
	return TCheckDetails{
		Project:     check.Project,
		Healthcheck: check.Healthcheck,
		Name:        check.Name,
		UUid:        check.UUid,
		LastExec:    check.LastExec,
		LastResult:  check.LastResult,
		LastPing:    check.LastPing,
	}
}

func (c *TConfig) UpdateCheckByUUID(check TCheckConfig) error {
	p := c.Projects
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = check

	//for _, h := range p[check.Project].Healthchecks {
	//	for _, _c := range h.Checks {
	//		if _c.UUid == uuid {
	//			return nil
	//		}
	//	}
	//}
	return nil //fmt.Errorf("check %s not in config", uuid)
}
