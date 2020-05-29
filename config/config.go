package config

import (
	"emperror.dev/errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"strings"
)

var (
	ScheduleLoop int
	Config       ConfigFile
	Log          *logrus.Logger                              = logrus.New()
	Checks       map[string]func(c *Check, p *Project) error = make(map[string]func(c *Check, p *Project) error)
	Timeouts     TimeoutsCollection
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
	Periods []string
}

type Project struct {
	Name        string
	Healtchecks []Healtchecks `mapstructure:"healthchecks"`
	Parameters  Parameters    `mapstructure:"parameters"`

	// Runtime data
	Timeouts TimeoutsCollection
}

type Healtchecks struct {
	Name   string
	Checks []Check `mapstructure:"checks"`

	// check level parameters
	Parameters Parameters `mapstructure:"parameters"`
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
	Code          []int
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
	UUid       string
	LastResult bool
}

type TimeoutCollection struct {
	periods []string
}

func LoadConfig() error {
	Log.Debug("Reading config...")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		Log.Panicf("Fatal error config file: %s \n", err)
	}

	dl, err := logrus.ParseLevel(viper.GetString("debugLevel"))
	if err != nil {
		Log.Panicf("Cannot parse debug level: %v", err)
	} else {
		Log.SetLevel(dl)
	}

	viper.Unmarshal(&Config)

	return nil
}

func (p *TimeoutsCollection) Add(period string) {
	var found bool
	if period != "" {
		for _, item := range p.Periods {
			if item == period {
				found = true
			}
		}
		if !found {
			p.Periods = append(p.Periods, period)
		}
	} else {
		Log.Debug("Empty timeout not adding")
	}
}

func FillDefaults() error {
	Log.Debug("Filling defaults...")

	//config.Log.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, project := range Config.Projects {
		if project.Parameters.RunEvery == "" {
			project.Parameters.RunEvery = Config.Defaults.Parameters.RunEvery
		}
		if project.Parameters.Mode == "" {
			project.Parameters.Mode = Config.Defaults.Parameters.Mode
		}
		if project.Parameters.AllowFails == 0 {
			project.Parameters.AllowFails = Config.Defaults.Parameters.AllowFails
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = Config.Defaults.Parameters.MinHealth
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = Config.Defaults.Parameters.MinHealth
		}
		if project.Parameters.Alert == "" {
			project.Parameters.Alert = Config.Defaults.Parameters.Alert
		}
		if project.Parameters.CritAlert == "" {
			project.Parameters.CritAlert = Config.Defaults.Parameters.Alert
		}
		if project.Parameters.PeriodicReport == "" {
			project.Parameters.PeriodicReport = Config.Defaults.Parameters.PeriodicReport
		}
		if project.Parameters.SSLExpirationPeriod == "" {
			project.Parameters.SSLExpirationPeriod = Config.Defaults.Parameters.SSLExpirationPeriod
		}
		Config.Projects[i] = project
	}

	return nil
}

func FillUUIDs() error {
	Log.Debug("Filling UUIDs...")
	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	for i := range Config.Projects {
		for j := range Config.Projects[i].Healtchecks {
			for k := range Config.Projects[i].Healtchecks[j].Checks {
				u2 := uuid.NewSHA1(ns, []byte(Config.Projects[i].Healtchecks[j].Checks[k].Host))
				Config.Projects[i].Healtchecks[j].Checks[k].UUid = u2.String()
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
		Log.Debug("Empty timeout not adding")
	}
}

func FillTimeouts() error {
	Log.Debug("Filling timeouts...")
	Timeouts.Add(Config.Defaults.Parameters.RunEvery)

	for _, project := range Config.Projects {

		//config.Log.Debugf("Project name: %s", project.Name)
		//config.Log.Debugf("Parameters: %+v", project.Parameters)

		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts.Add(project.Parameters.RunEvery)
		}
		for _, healthcheck := range project.Healtchecks {
			if healthcheck.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
				Timeouts.Add(healthcheck.Parameters.RunEvery)
				project.Timeouts.Add(healthcheck.Parameters.RunEvery)
			}
			Log.Debugf("Project %s timeouts found: %+v\n", project.Name, project.Timeouts)
		}
	}
	Log.Debugf("Total timeouts found: %+v\n\n", Timeouts)

	return nil
}

func FillSecrets() error {
	Log.Debug("Filling secrets...")
	for i, a := range Config.Alerts {
		if strings.HasPrefix(a.BotToken, "vault") {
			secret := strings.Split(a.BotToken, ":")
			path := secret[1]
			field := secret[2]
			token, err := GetVaultSecret(path, field)
			if err == nil {
				Config.Alerts[i].BotToken = token
				Log.Debugf("Bot %s token %s", a.Name, Config.Alerts[i].BotToken)
			} else {
				return errors.Errorf("Cannot get vault secret %s:%s, err: %v", path, field, err)
			}
		}
	}
	return nil
}
