package main

import "net/http"

var (
	ScheduleLoop int
	Config       ConfigInt
)

type Parameters struct {
	// Messages mode quiet/loud
	Mode string `json:"mode"`
	// Checks should be run every RunEvery seconds
	RunEvery       string `json:"run_every"`
	PeriodicReport string `json:"periodic_report_time"`
	// minimum passed checks to consider project healthy
	MinHealth int `json:"min_health"`
	// how much consecutive critical checks may fail to consider not healthy
	AllowFails int `json:"allow_fails"`
	// alert name
	Alert               string `json:"noncrit_alert"`
	CritAlert           string `json:"crit_alert"`
	CommandChannel      string `json:"command_channel"`
	SSLExpirationPeriod string `json:"ssl_expiration_period"`
}

type ConfigInt struct {
	Defaults struct {
		Parameters Parameters
	}
	Projects []Project
	Alerts   []AlertConfigs
}

type ConfigFile struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		TimerStep  string      `json:"timer_step"`
		Parameters *Parameters `json:"parameters"`
		// HTTP port web interface listen
		HTTPPort string `json:"http_port"`
		// If not empty HTPP server not enabled
		HTTPEnabled string `json:"http_enabled"`
	}
	Alerts   []*AlertConfigs `json:"alerts"`
	Projects []*Project      `json:"projects"`
}

type AlertConfigs struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Tg token for bot
	BotToken string `json:"bot_token"`
	// Messages mode quiet/loud
	CriticalChannel int64 `json:"critical_channel"`
	// Empty by default, alerts will not be sent unless critical
	ProjectChannel int64 `json:"noncritical_channel"`
}

type TimeoutsCollection struct {
	periods []string
}

type Project struct {
	Name        string         `json:"name"`
	Healtchecks []*Healtchecks `json:"healthchecks"`
	Parameters  Parameters     `json:"parameters"`

	// Runtime data
	ErrorsCount int
	FailsCount  int
	Timeouts    TimeoutsCollection
}

type Healtchecks struct {
	Name   string
	Checks []*Check

	// check level parameters
	Parameters Parameters `json:"parameters"`

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
	AnswerPresent string `json:"answer_present"`
	Headers       []map[string]string
	Auth          struct {
		User     string
		Password string
	}
	SkipCheckSSL        bool `json:"skip_check_ssl"`
	StopFollowRedirects bool `json:"stop_follow_redirects"`
	Cookies             []*http.Cookie

	// Check SQL query parameters
	SqlQueryConfig struct {
		DBName, UserName, Password, Query, Response, Difference string
	} `json:"sql_query_config"`

	// Check SQL replication parameters
	SqlReplicationConfig struct {
		DBName, UserName, Password, TableName string
		ServerList                            []string
	} `json:"sql_repl_config"`

	PubSub struct {
		Password string
		Channels []string
	} `json:"pubsub_config"`

	// Runtime data
	uuID       string
	LastResult bool
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
