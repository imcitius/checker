package checks

import (
	"net/http"
	"time"
)

type TCommonCheck struct {
	Name        string
	Sid         string
	UUID        string
	Project     string
	Healthcheck string
	Type        string
	// Alerter     ICommonAlerter
	Parameters  TCheckParameters
	CheckConfig TCheckConfig
	RealCheck   ISpecificCheck
	Result      TCheckResult
	Enabled     bool
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
	UUID       string
	LastResult bool
	// datetime when check was performed last time by checker
	LastExec time.Time

	// datetime when passive style check was pinged via web
	LastPing time.Time

	DebugLevel string
	Parameters TCheckParameters `yaml:"parameters"`
	Enabled    bool             `yaml:"enabled"`
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

type TCheckDetails struct {
	Project     string
	Healthcheck string
	Name        string
	UUID        string
	LastResult  bool
	LastExec    time.Time
	LastPing    time.Time
	Enabled     bool
}

type TCheckResult struct {
	Duration time.Duration
	Error    error
}

type TChecksCollection struct {
	Checks []TCheckWithDuration
}

type TCheckWithDuration struct {
	Check    ICommonCheck
	Duration string
}

type THealthcheck struct {
	Name       string                  `yaml:"name"`
	Parameters TCheckParameters        `yaml:"parameters"`
	Checks     map[string]TCheckConfig `yaml:"checks"`
}
