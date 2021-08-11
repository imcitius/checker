package config

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/knadh/koanf"
)

const (
	PeriodicReport = "30m"
)

var (
	Version      string
	VersionSHA   string
	VersionBuild string

	StartTime      = time.Now()
	InternalStatus = "starting"

	ScheduleLoop int
	Config       ConfigFile
	Log          = *logrus.New()

	Timeouts TimeoutsCollection
	Sem      = semaphore.NewWeighted(int64(1))

	SignalINT, SignalHUP                                          chan os.Signal
	ConfigChangeSig, SchedulerSignalCh, BotsSignalCh, WebSignalCh chan bool
	Wg                                                            sync.WaitGroup

	Koanf = koanf.New(".")

	Secrets map[string]CachedSecret

	TokenEncryptionKey []byte
)

type CachedSecret struct {
	Secret    string
	TimeStamp time.Time
}

type ConfigFile struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		TimerStep  string     `koanf:"timer_step"`
		Parameters Parameters `koanf:"parameters"`

		// HTTP port web interface listen
		HTTPPort string `koanf:"http_port"`

		// If not empty HTPP server not enabled
		HTTPEnabled        string `koanf:"http_enabled"`
		TokenEncryptionKey []byte `koanf:"token_encryption_key"`
	}
	Alerts   []AlertConfigs
	Actors   []ActorConfigs
	Projects []Project

	ConsulCatalog ConsulCatalog `koanf:"consul_catalog"`
}

type Parameters struct {
	// Messages mode quiet/loud
	Mode string

	// Checks should be run every RunEvery seconds
	RunEvery       string `koanf:"run_every"`
	PeriodicReport string `koanf:"periodic_report_time"`

	// minimum passed checks to consider project healthy
	MinHealth int `koanf:"min_health"`

	// how much consecutive critical checks may fail to consider not healthy
	AllowFails int `koanf:"allow_fails"`

	// alert name
	AlertChannel        string `koanf:"noncrit_alert"`
	CritAlertChannel    string `koanf:"crit_alert"`
	CommandChannel      string `koanf:"command_channel"`
	SSLExpirationPeriod string `koanf:"ssl_expiration_period"`

	Mentions []string
}

type ConsulCatalog struct {
	Address string
	ACL     string
	Enabled bool
}

type AlertConfigs struct {
	Name string
	Type string
	// token for bot
	BotToken string `koanf:"bot_token"`
	// critical channel name
	CriticalChannel int64 `koanf:"critical_channel"`
	// non critical and chatops channel name
	ProjectChannel int64 `koanf:"noncritical_channel"`

	MMWebHookURL string `koanf:"mattermost_webhook_url"`
}

type ActorConfigs struct {
	Name string
	Type string
}

type TimeoutsCollection struct {
	Periods []string
}

type Project struct {
	Name         string
	Healthchecks []Healthcheck `koanf:"healthchecks"`
	Parameters   Parameters    `koanf:"parameters"`

	// Runtime data
	Timeouts TimeoutsCollection
}

type Healthcheck struct {
	Name   string
	Checks []Check `koanf:"checks"`

	// check level parameters
	Parameters Parameters `koanf:"parameters"`
}

type Check struct {
	Name string

	// Parameters related to check execution
	Type    string
	Host    string
	Timeout string
	Port    int

	// hash for fileget check
	Hash string
	// retries
	Attempts int

	// alert mode
	Mode string

	// for ping check - pings count
	Count int

	// allowed seq fails number
	AllowFails int `koanf:"allow_fails"`

	// http checks optional parameters
	Code          []int
	Answer        string
	AnswerPresent string              `koanf:"answer_present"`
	Headers       []map[string]string `koanf:"headers"`
	Auth          struct {
		User     string
		Password string
	} `koanf:"auth"`
	SkipCheckSSL        bool `koanf:"skip_check_ssl"`
	StopFollowRedirects bool `koanf:"stop_follow_redirects"`
	Cookies             []http.Cookie

	// Check SQL query parameters
	SqlQueryConfig struct {
		DBName, UserName, Password, Query, Response, Difference, SSLMode string
	} `koanf:"sql_query_config"`

	// Check SQL replication parameters
	SqlReplicationConfig struct {
		DBName, UserName, Password, TableName, SSLMode string
		ServerList                                     []string
	} `koanf:"sql_repl_config"`

	PubSub struct {
		Password string
		Channels []string
		SSLMode  bool
	} `koanf:"pubsub_config"`

	Actors struct {
		Up   string
		Down string
	} `koanf:"actors"`

	// Runtime data
	UUid       string
	LastResult bool
}

type TimeoutCollection struct {
	periods []string
}
