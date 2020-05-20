package cmd

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"sync"
)

var (
	ScheduleLoop              int
	Config                    ConfigFile
	log                       *logrus.Logger = logrus.New()
	signalINT, signalHUP      chan os.Signal
	doneCh, schedulerSignalCh chan bool
	wg                        sync.WaitGroup
	interrupt                 bool = false
)

type Parameters struct {
	// Messages mode quiet/loud
	Mode string
	// Checks should be run every RunEvery seconds
	RunEvery       string `mapstructure:"run_every"`
	PeriodicReport string `mapstructure:"periodic_report_time"`
	// minimum passed checks to consider project healthy
	MinHealth int `mapstructure:"min_health"`
	// how much consecutive critical checks may fail to consider not healthy
	AllowFails int `mapstructure:"allow_fails"`
	// alert name
	Alert               string `mapstructure:"noncrit_alert"`
	CritAlert           string `mapstructure:"crit_alert"`
	CommandChannel      string `mapstructure:"command_channel"`
	SSLExpirationPeriod string `mapstructure:"ssl_expiration_period"`
}

type ConfigFile struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		TimerStep  string     `mapstructure:"timer_step"`
		Parameters Parameters `mapstructure:"parameters"`
		// HTTP port web interface listen
		HTTPPort string `mapstructure:"http_port"`
		// If not empty HTPP server not enabled
		HTTPEnabled string `mapstructure:"http_enabled"`
	}
	Alerts   []AlertConfigs
	Projects []Project
}

type AlertConfigs struct {
	Name string
	Type string
	// Tg token for bot
	BotToken string `mapstructure:"bot_token"`
	// Messages mode quiet/loud
	CriticalChannel int64 `mapstructure:"critical_channel"`
	// Empty by default, alerts will not be sent unless critical
	ProjectChannel int64 `mapstructure:"noncritical_channel"`
}

type TimeoutsCollection struct {
	periods []string
}

type Project struct {
	Name        string
	Healtchecks []Healtchecks `mapstructure:"healthchecks"`
	Parameters  Parameters    `mapstructure:"parameters"`

	// Runtime data
	ErrorsCount int
	FailsCount  int
	Timeouts    TimeoutsCollection
}

type Healtchecks struct {
	Name   string
	Checks []Check `mapstructure:"checks"`

	// check level parameters
	Parameters Parameters `mapstructure:"parameters"`

	RunCount    int
	ErrorsCount int
	FailsCount  int
}

type Check struct {
	// Parameters related to healthcheck execution
	Type     string
	Host     string
	Timeout  string
	Port     int
	Attempts int
	Mode     string
	Count    int

	// http checks optional parameters
	Code          int
	Answer        string
	AnswerPresent string              `mapstructure:"answer_present"`
	Headers       []map[string]string `mapstructure:"headers"`
	Auth          struct {
		User     string
		Password string
	} `mapstructure:"auth"`
	SkipCheckSSL        bool `mapstructure:"skip_check_ssl"`
	StopFollowRedirects bool `mapstructure:"stop_follow_redirects"`
	Cookies             []http.Cookie

	// Check SQL query parameters
	SqlQueryConfig struct {
		DBName, UserName, Password, Query, Response, Difference string
	} `mapstructure:"sql_query_config"`

	// Check SQL replication parameters
	SqlReplicationConfig struct {
		DBName, UserName, Password, TableName string
		ServerList                            []string
	} `mapstructure:"sql_repl_config"`

	PubSub struct {
		Password string
		Channels []string
	} `mapstructure:"pubsub_config"`

	// Runtime data
	uuID       string
	LastResult bool
}

type TimeoutCollection struct {
	periods []string
}

func (c *ConfigFile) loadConfig() error {

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Panicf("Fatal error config file: %s \n", err)
	}

	dl, err := logrus.ParseLevel(viper.GetString("debugLevel"))
	if err != nil {
		log.Panicf("Cannot parse debug level: %v", err)
	} else {
		log.SetLevel(dl)
	}

	viper.Unmarshal(c)

	return nil
}

func (p *TimeoutsCollection) Add(period string) {
	var found bool
	if period != "" {
		for _, item := range p.periods {
			if item == period {
				found = true
			}
		}
		if !found {
			p.periods = append(p.periods, period)
		}
	} else {
		log.Debug("Empty timeout not adding")
	}
}

var testCfg = &cobra.Command{
	Use:   "testcfg",
	Short: "unmarshal config file into config structure",
	Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {

		log.Infof("Config :\n%+v\n\n\n", Config)
	},
}

func (c *ConfigFile) fillDefaults() error {

	//log.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, project := range c.Projects {
		if project.Parameters.RunEvery == "" {
			project.Parameters.RunEvery = c.Defaults.Parameters.RunEvery
		}
		if project.Parameters.Mode == "" {
			project.Parameters.Mode = c.Defaults.Parameters.Mode
		}
		if project.Parameters.AllowFails == 0 {
			project.Parameters.AllowFails = c.Defaults.Parameters.AllowFails
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = c.Defaults.Parameters.MinHealth
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = c.Defaults.Parameters.MinHealth
		}
		if project.Parameters.Alert == "" {
			project.Parameters.Alert = c.Defaults.Parameters.Alert
		}
		if project.Parameters.CritAlert == "" {
			project.Parameters.CritAlert = c.Defaults.Parameters.Alert
		}
		if project.Parameters.PeriodicReport == "" {
			project.Parameters.PeriodicReport = c.Defaults.Parameters.PeriodicReport
		}
		if project.Parameters.SSLExpirationPeriod == "" {
			project.Parameters.SSLExpirationPeriod = c.Defaults.Parameters.SSLExpirationPeriod
		}
		Config.Projects[i] = project
	}

	return nil
}

func (c *ConfigFile) fillUUIDs() error {
	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	for i := range c.Projects {
		for j := range c.Projects[i].Healtchecks {
			for k := range c.Projects[i].Healtchecks[j].Checks {
				u2 := uuid.NewSHA1(ns, []byte(Config.Projects[i].Healtchecks[j].Checks[k].Host))
				c.Projects[i].Healtchecks[j].Checks[k].uuID = u2.String()
			}
		}
	}
	return err
}

func (p *TimeoutCollection) Add(period string) {
	var found bool
	if period != "" {
		for _, item := range p.periods {
			if item == period {
				found = true
			}
		}
		if !found {
			p.periods = append(p.periods, period)
		}
	} else {
		log.Debug("Empty timeout not adding")
	}
}
