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
	DefaultPeriodicReportPeriod = "1h"
	DefaultCheckPeriod          = "1m"
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	DefaultSSLExpiration = "360h"

	DefaultHTTPCheckTimeout  = "2s"
	DefaultTCPConnectTimeout = "2s"

	DefaultDebugLevel = "warn"
)

var (
	Version      = "dev"
	VersionSHA   = "dev"
	VersionBuild = "dev"

	StartTime      = time.Now()
	InternalStatus = "starting"

	ScheduleLoop int
	Config       File
	Log          = *logrus.New()

	Timeouts TimeoutsCollection
	Sem      = semaphore.NewWeighted(int64(1))

	SignalINT         chan os.Signal
	SignalHUP         chan os.Signal
	ConfigChangeSig   chan bool
	SchedulerSignalCh chan bool
	BotsSignalCh      chan bool
	WebSignalCh       chan bool
	Wg                sync.WaitGroup

	Koanf = koanf.New(".")

	Secrets map[string]CachedSecret

	TokenEncryptionKey []byte

	TickersCollection = map[string]*time.Ticker{}
	ReportsTicker     = &time.Ticker{}
)

type CachedSecret struct {
	Secret    string
	TimeStamp time.Time
}

type File struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		// should always be 1s, to avoid time drift bugs in scheduler
		TimerStep  string     `koanf:"timer_step"`
		Parameters Parameters `koanf:"parameters"`

		// HTTP port web interface listen
		HTTPPort string `koanf:"http_port"`

		// If empty HTTP server is not enabled
		HTTPEnabled        string `koanf:"http_enabled"`
		TokenEncryptionKey []byte `koanf:"token_encryption_key"`

		BotsEnabled        bool `koanf:"bots_enabled"`
		BotGreetingEnabled bool `koanf:"bots_greeting_enabled"`

		DebugLevel string `koanf:"debug_level"`
	}
	Alerts   []AlertConfigs
	Actors   []ActorConfigs
	Projects []Project

	ConsulCatalog ConsulCatalog `koanf:"consul_catalog"`
	SlackApp      SlackAppConfig `koanf:"slack_app"`
}

type Parameters struct {
	// Messages mode quiet/loud
	Mode string

	// Checks should be run every Period seconds
	Period       string `koanf:"check_period"`
	ReportPeriod string `koanf:"report_period"`

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

// SlackAppConfig holds configuration for the Slack App integration.
// This is separate from the legacy Mattermost/Slack webhook alert type.
type SlackAppConfig struct {
	BotToken       string `koanf:"bot_token"`
	SigningSecret  string `koanf:"signing_secret"`
	DefaultChannel string `koanf:"default_channel"`
	DatabaseURL    string `koanf:"database_url"`
}

type ConsulCatalog struct {
	Address string
	ACL     string
	Enabled bool
}

type AlertConfigs struct {
	Name string
	Type string
	// token for bot (Telegram bot token or Slack Bot User OAuth Token xoxb-...)
	BotToken string `koanf:"bot_token"`
	// critical channel name
	CriticalChannel string `koanf:"critical_channel"`
	// non critical and chatops channel name
	ProjectChannel string `koanf:"noncritical_channel"`

	// Legacy Mattermost/Slack webhook URL.
	// If only this is set (with type "slack"), the legacy webhook mode is used.
	MMWebHookURL string `koanf:"mattermost_webhook_url"`

	// Slack App fields (used when type is "slack" and bot_token is set).
	// signing_secret is used for Slack request signature verification.
	SigningSecret string `koanf:"signing_secret"`
	// channel_id is the default Slack channel ID for this alert destination.
	ChannelID string `koanf:"channel_id"`
}

// IsSlackApp returns true if this alert config uses the Slack App (Bot Token API)
// rather than the legacy Mattermost webhook mode.
// When both bot_token and mattermost_webhook_url are set, bot_token takes precedence.
func (a *AlertConfigs) IsSlackApp() bool {
	return a.Type == "slack" && a.BotToken != ""
}

// IsLegacyWebhook returns true if this alert config uses the legacy webhook mode.
// This is the case when type is "slack" or "mattermost" and only webhook_url is set.
func (a *AlertConfigs) IsLegacyWebhook() bool {
	return (a.Type == "slack" || a.Type == "mattermost") && a.MMWebHookURL != "" && a.BotToken == ""
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
	Type     string
	Host     string
	Timeout  string
	Port     int
	Severity string `koanf:"severity"`

	// hash and size for fileget check
	Hash string
	Size int64
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
		// allowed replication lag
		Lag              string
		AnalyticReplicas []string `koanf:"analytic_replicas"`
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

	DebugLevel string
}

type TimeoutCollection struct {
	periods []string
}
