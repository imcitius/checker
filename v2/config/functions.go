package config

import (
	"fmt"
	"sync"
	"time"

	alerts "my/checker/models/alerts"
	checks "my/checker/models/checks"
	store "my/checker/store"

	"github.com/kofalt/go-memoize"
	"github.com/sirupsen/logrus"
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

	if config.DB.Protocol != "" {
		SetConfigurer(&config)
		_, err := InitDB()
		if err != nil {
			logger.Errorf("Failed to initialize store: %v", err)
		}
	}
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

func toModelCheckConfig(c checks.TCheckConfig) checks.TCheckConfig {
	return checks.TCheckConfig{
		Name:        c.Name,
		Type:        c.Type,
		Project:     c.Project,
		Healthcheck: c.Healthcheck,
		UUID:        c.UUID,
		LastResult:  c.LastResult,
		LastExec:    c.LastExec,
		LastPing:    c.LastPing,
		Enabled:     c.Enabled,
	}
}

func fromModelCheckConfig(c checks.TCheckConfig) checks.TCheckConfig {
	return checks.TCheckConfig{
		Name:        c.Name,
		Type:        c.Type,
		Project:     c.Project,
		Healthcheck: c.Healthcheck,
		UUID:        c.UUID,
		LastResult:  c.LastResult,
		LastExec:    c.LastExec,
		LastPing:    c.LastPing,
		Enabled:     c.Enabled,
	}
}

func (c *TConfig) GetCheckByUUid(uuid string) (checks.TCheckConfig, error) {
	for _, p := range c.Projects {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				if c.UUID == uuid {
					return toModelCheckConfig(c), nil
				}
			}
		}
	}
	return checks.TCheckConfig{}, fmt.Errorf("check not found")
}

func (c *TConfig) ListChecks() (interface{}, error) {
	type listObject struct {
		UUID       string
		LastResult bool
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
					UUID:       c.UUID,
					LastResult: c.LastResult,
					LastExec:   c.LastExec,
					LastPing:   c.LastPing,
				}
			}
		}
	}
	return list, nil
}

func (c *TConfig) GetAllChecks() ([]checks.TCheckConfig, error) {
	list := make([]checks.TCheckConfig, 0)

	for _, p := range c.Projects {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				list = append(list, toModelCheckConfig(c))
			}
		}
	}
	return list, nil
}

func (c *TConfig) Ping(uuid string) (checks.TCheckConfig, error) {
	check, _ := c.GetCheckByUUid(uuid)
	check.LastPing = time.Now()
	p := c.Projects
	localCheck := fromModelCheckConfig(check)
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = localCheck
	return check, nil
}

func (c *TConfig) SetStatus(uuid string, status bool) {
	check, _ := c.GetCheckByUUid(uuid)
	check.LastExec = time.Now()
	check.LastResult = status
	p := c.Projects
	localCheck := fromModelCheckConfig(check)
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = localCheck

	if err := c.Save(); err != nil {
		logger.Errorf("Failed to save config: %v", err)
	}
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

func (c *TConfig) GetCheckDetails(uuid string) checks.TCheckDetails {
	check, _ := c.GetCheckByUUid(uuid)
	return checks.TCheckDetails{
		Project:     check.Project,
		Healthcheck: check.Healthcheck,
		Name:        check.Name,
		UUID:        check.UUID,
		LastExec:    check.LastExec,
		LastResult:  check.LastResult,
		LastPing:    check.LastPing,
		Enabled:     check.Enabled,
	}
}

func (c *TConfig) UpdateCheckByUUID(check checks.TCheckConfig) error {
	p := c.Projects
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = fromModelCheckConfig(check)

	logger.Infof("Check state: %+v", p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name])

	if err := c.Save(); err != nil {
		logger.Errorf("Failed to save config: %v", err)
		return err
	}

	return nil
}

func (c *TConfig) ToggleCheck(uuid string, enabled bool) error {
	check, _ := c.GetCheckByUUid(uuid)
	check.Enabled = enabled
	c.UpdateCheckByUUID(check)

	return nil
}

func (c *TConfig) Save() error {
	if !c.DB.Connected {
		return fmt.Errorf("database not connected")
	}
	return store.Store.UpdateChecks()
}

func (c *TConfig) GetDB() DBConfig {
	return DBConfig{
		Protocol:  c.DB.Protocol,
		Host:      c.DB.Host,
		Port:      c.DB.Port,
		Username:  c.DB.Username,
		Password:  c.DB.Password,
		Database:  c.DB.Database,
		Connected: c.DB.Connected,
	}
}

func (c *TConfig) GetAlerts() []alerts.TAlert {
	alerts := make([]alerts.TAlert, 0, len(c.Alerts))
	for name, a := range c.Alerts {
		alert := alerts.TAlert{
			Name:            name,
			Type:            a.Type,
			BotToken:        a.BotToken,
			ProjectChannel:  a.ProjectChannel,
			CriticalChannel: a.CriticalChannel,
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func (c *TConfig) SetDBConnected() {
	c.DB.Connected = true
}

func InitDB() (store.IStore, error) {
	switch config.GetDB().Protocol {
	case "mongodb":
		DB, err := store.InitMongoDB(config.GetDBConnectionString())
		if err != nil {
			return nil, err
		}
		// return store, nil
		// client := &mongoDbStore{}
		// store, err := client.Init()
		// if err != nil {
		// 	return nil, err
		// }
		// Store = store
	}
	config.SetDBConnected()

	loadChecks()
	loadAlerts()
	// return Store, nil
}

func loadChecks() {
	checksFromDb, err := Store.GetAllChecks()
	if err != nil {
		logger.Fatalf("Cannot get checks from DB: %s", err)
	}
	for _, o := range checksFromDb {
		check, err := GetCheckByUUid(o.UUID)
		if err != nil {
			logger.Debugf("Cannot get check from db: %s", err.Error())
			continue
		}
		check.LastExec = o.LastExec
		check.LastPing = o.LastPing
		check.LastResult = o.LastResult
		check.Enabled = o.Enabled

		err = UpdateCheckByUUID(check)
		if err != nil {
			logger.Errorf("Cannot update check: %s", err.Error())
		}
	}
}

func loadAlerts() {
	//alertsFromDb, err := Store.GetAllAlerts()

	var err error = nil
	if err != nil {
		logger.Fatalf("Cannot get checks from DB: %s", err)
	}
	//for _, o := range alertsFromDb.data {
	//
	//}
}

func SetConfigurer(cfg *TConfig) {
	config = *cfg
}
