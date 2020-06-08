package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sync/semaphore"
	"net/http"
	"os"
	"sync"
)

var (
	ScheduleLoop               int
	Config                     ConfigFile
	Log                        = logrus.New()
	Checks                     = make(map[string]func(c *Check, p *Project) error)
	Timeouts                   TimeoutsCollection
	CfgFile, CfgSrc, CfgFormat string
	CfgWatchTimeout            string
	Sem                        = semaphore.NewWeighted(int64(1))
	Version                    string
	VersionSHA                 string
	VersionBuild               string

	SignalINT, SignalHUP                                          chan os.Signal
	ConfigChangeSig, SchedulerSignalCh, BotsSignalCh, WebSignalCh chan bool
	Wg                                                            sync.WaitGroup

	Viper = viper.New()
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
	AlertChannel        string `mapstructure:"noncrit_alert"`
	CritAlertChannel    string `mapstructure:"crit_alert"`
	CommandChannel      string `mapstructure:"command_channel"`
	SSLExpirationPeriod string `mapstructure:"ssl_expiration_period"`

	Mentions []string
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

	MMWebHookURL string `mapstructure:"mattermost_webhook_url"`
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
		DBName, UserName, Password, Query, Response, Difference, SSLMode string
	} `mapstructure:"sql_query_config"`

	// Check SQL replication parameters
	SqlReplicationConfig struct {
		DBName, UserName, Password, TableName, SSLMode string
		ServerList                                     []string
	} `mapstructure:"sql_repl_config"`

	PubSub struct {
		Password string
		Channels []string
		SSLMode  bool
	} `mapstructure:"pubsub_config"`

	// Runtime data
	UUid       string
	LastResult bool
}

type TimeoutCollection struct {
	periods []string
}
