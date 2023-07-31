package config

import (
	"net/http"
	"time"
)

type TConfig struct {
	Defaults TDefaults `yaml:"defaults"`
	DB       DBConfig  `yaml:"db"`

	Alerts    map[string]TAlert   `yaml:"alerts"`
	Projects  map[string]TProject `yaml:"projects"`
	StartTime time.Time
}

func (c *TConfig) SetDBConnected() {
	c.DB.Connected = true
}

type DBConfig struct {
	Protocol string `yaml:"protocol" env-default:""`
	Host     string `yaml:"host" env-default:""`
	Port     string `yaml:"port" env-default:""`
	Username string `yaml:"username" env-default:""`
	Password string `yaml:"password" env-default:""`
	Database string `yaml:"database" env-default:""`

	Connected bool
}

type TDefaults struct {
	// Main timer evaluates every TimerStep seconds
	// should always be 1s, to avoid time drift bugs in scheduler
	TimerStep string `yaml:"timer_step" env-default:"1s"`
	// How often to run check by default
	Duration            string `yaml:"duration" env-default:"60s"`
	MaintenanceDuration string `yaml:"maintenance_duration" env-default:"60s"`

	// HTTP port web interface listen
	HTTPPort string `yaml:"http_port" env-default:"80"`
	// If empty HTTP server is not enabled
	HTTPEnabled        string `yaml:"http_enabled" env-default:"true"`
	TokenEncryptionKey []byte `yaml:"token_encryption_key"`

	BotsEnabled        bool `yaml:"bots_enabled" env-default:"true"`
	BotGreetingEnabled bool `yaml:"bots_greeting_enabled" env-default:"false"`

	DebugLevel    string `yaml:"debug_level" env-default:"info"`
	AlertsChannel string `yaml:"alerts_channel" env-default:"log"`

	DefaultCheckParameters TCheckParameters `yaml:"default_check_parameters"`
}

type TCheckConfig struct {
	Name string
	Type string

	Project     string
	Healthcheck string

	// Url for http and getfile check
	Url string
	// Host for other checks types
	Host     string
	Timeout  string `yaml:"timeout" env-default:"3s"`
	Port     int
	Severity string

	// hash and size for getfile check
	Hash string
	Size int64
	// retries
	Attempts int

	// alert mode
	Mode string

	// for ping check - pings count
	Count int `yaml:"count" env-default:"3"`

	// allowed seq fails number
	AllowFails int `yaml:"allow_fails"`

	// http checks optional parameters
	Code          []int
	Answer        string
	AnswerPresent string `yaml:"answer_present"`
	Headers       []map[string]string
	Auth          struct {
		User     string
		Password string
	} `yaml:"auth"`
	SkipCheckSSL        bool   `yaml:"skip_check_ssl"`
	StopFollowRedirects bool   `yaml:"stop_follow_redirects"`
	SSLExpirationPeriod string `yaml:"ssl_expiration_period"`
	Cookies             []http.Cookie

	// Check SQL query parameters
	SqlQueryConfig struct {
		DBName, UserName, Password, Query, Response, Difference, SSLMode string
	} `yaml:"sql_query_config"`

	// Check SQL replication parameters
	SqlReplicationConfig struct {
		DBName, UserName, Password, TableName, SSLMode string
		ServerList                                     []string
		// allowed replication lag
		Lag              string
		AnalyticReplicas []string `yaml:"analytic_replicas"`
	} `yaml:"sql_repl_config"`

	PubSub struct {
		Password string
		Channels []string
		SSLMode  bool
	} `yaml:"pubsub_config"`

	Actors struct {
		Up   string
		Down string
	} `yaml:"actors"`

	// Runtime data
	UUid       string
	LastResult bool
	LastExec   time.Time
	LastPing   time.Time

	DebugLevel string
	Parameters TCheckParameters `yaml:"parameters"`
}

type TCheckParameters struct {
	// Messages mode quiet/loud
	Mode string `yaml:"mode" env-default:"loud"`

	// Checks should be run every Duration seconds
	Duration string `yaml:"duration" env-default:"60s"`

	// timeout of the check's job, for example http request timeout
	Timeout             string `yaml:"timeout" env-default:"3s"`
	SSLExpirationPeriod string `yaml:"ssl_expiration_period" env-default:"360h"` // 15 days

	// minimum passed checks to consider project healthy
	MinHealth int `yaml:"min_health" env-default:"1"`

	// how much consecutive critical checks may fail to consider not healthy
	AllowFails int `yaml:"allow_fails" env-default:"0"`

	// alert name
	AlerterName      string `yaml:"alerter"`
	AlertChannel     string `yaml:"noncrit_channel"`
	CritAlertChannel string `yaml:"crit_channel"`
	CommandChannel   string `yaml:"command_channel"`

	Mentions []string `yaml:"mentions"`
}

type TProject struct {
	Name         string                  `yaml:"name"`
	Healthchecks map[string]THealthcheck `yaml:"healthchecks"`
	Parameters   TCheckParameters        `yaml:"parameters"`

	// Runtime data
	//Timeouts TimeoutsCollection
}

type THealthcheck struct {
	Name       string                  `yaml:"name"`
	Parameters TCheckParameters        `yaml:"parameters"`
	Checks     map[string]TCheckConfig `yaml:"checks"`
}

type TAlertsConfig struct {
	Alerts *map[string]TAlert
}

type TAlert struct {
	Type string `yaml:"type"`
	// token for bot
	BotToken string `yaml:"bot_token"`
	// critical channel name
	CriticalChannel string `yaml:"critical_channel"`
	// non critical and chatops channel name
	ProjectChannel string `yaml:"noncritical_channel"`

	MMWebHookURL string `yaml:"mattermost_webhook_url"`
}
