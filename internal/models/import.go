package models

// CheckImportPayload represents the YAML/JSON payload for bulk check import.
//
// Supports two usage modes:
//
// Mode 1 - Simple list (dashboard bulk import):
//
//	checks:
//	  - name: "API Health"
//	    project: "my-service"
//	    type: http
//	    url: https://api.example.com/healthz
//	    duration: 30s
//
// Mode 2 - Service config (CI-oriented, scoped by project + environment):
//
//	project: my-service
//	environment: prod
//	defaults:
//	  duration: 30s
//	  alert_type: slack
//	  alert_destination: "#alerts"
//	prune: true
//	checks:
//	  - name: "API Health"
//	    type: http
//	    url: https://api.example.com/healthz
type CheckImportPayload struct {
	// Project scope — when set, all checks inherit this project unless overridden
	Project string `json:"project" yaml:"project"`

	// Environment — becomes the group_name for all checks (e.g. "prod", "staging")
	Environment string `json:"environment" yaml:"environment"`

	// Source tracks where the import came from (e.g. "ci", "dashboard", "api")
	Source string `json:"source" yaml:"source"`

	// Prune — if true, checks in this project+environment scope that are NOT
	// in the payload will be deleted
	Prune bool `json:"prune" yaml:"prune"`

	// Defaults applied to all checks that don't override them
	Defaults CheckImportDefaults `json:"defaults" yaml:"defaults"`

	// The checks to create/update
	Checks []CheckImportItem `json:"checks" yaml:"checks"`
}

// CheckImportDefaults are default values applied to all imported checks
type CheckImportDefaults struct {
	Duration         string   `json:"duration" yaml:"duration"`
	Timeout          string   `json:"timeout" yaml:"timeout"`
	Enabled          *bool    `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	AlertType        string   `json:"alert_type" yaml:"alert_type"`
	AlertDestination string   `json:"alert_destination" yaml:"alert_destination"`
	ActorType        string   `json:"actor_type" yaml:"actor_type"`
	Severity         string   `json:"severity,omitempty" yaml:"severity,omitempty"`
	AlertChannels    []string `json:"alert_channels,omitempty" yaml:"alert_channels,omitempty"`
	ReAlertInterval  string   `json:"re_alert_interval,omitempty" yaml:"re_alert_interval,omitempty"`
	RetryCount       int      `json:"retry_count,omitempty" yaml:"retry_count,omitempty"`
	RetryInterval    string   `json:"retry_interval,omitempty" yaml:"retry_interval,omitempty"`
}

// AuthImportConfig holds HTTP Basic Auth credentials for import
type AuthImportConfig struct {
	User     string `json:"user,omitempty" yaml:"user,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

// CheckImportItem represents a single check in the import payload.
// It's a flat structure matching the CheckDefinitionViewModel shape.
type CheckImportItem struct {
	// Core fields
	Name        string `json:"name" yaml:"name"`
	Project     string `json:"project" yaml:"project"`
	GroupName   string `json:"group_name" yaml:"group_name"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description" yaml:"description"`
	Enabled     *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Duration    string `json:"duration" yaml:"duration"`

	// HTTP config
	URL                 string              `json:"url,omitempty" yaml:"url,omitempty"`
	Timeout             string              `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Answer              string              `json:"answer,omitempty" yaml:"answer,omitempty"`
	AnswerPresent       *bool               `json:"answer_present,omitempty" yaml:"answer_present,omitempty"`
	Code                []int               `json:"code,omitempty" yaml:"code,omitempty"`
	Headers             []map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Cookies             []map[string]string `json:"cookies,omitempty" yaml:"cookies,omitempty"`
	SkipCheckSSL        *bool               `json:"skip_check_ssl,omitempty" yaml:"skip_check_ssl,omitempty"`
	SSLExpirationPeriod string              `json:"ssl_expiration_period,omitempty" yaml:"ssl_expiration_period,omitempty"`
	StopFollowRedirects *bool               `json:"stop_follow_redirects,omitempty" yaml:"stop_follow_redirects,omitempty"`
	Auth                *AuthImportConfig   `json:"auth,omitempty" yaml:"auth,omitempty"`

	// TCP/ICMP config
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	Port int    `json:"port,omitempty" yaml:"port,omitempty"`

	// DNS config
	RecordType string `json:"record_type,omitempty" yaml:"record_type,omitempty"`
	Expected   string `json:"expected,omitempty" yaml:"expected,omitempty"`

	// Domain expiry config
	Domain            string `json:"domain,omitempty" yaml:"domain,omitempty"`
	ExpiryWarningDays int    `json:"expiry_warning_days,omitempty" yaml:"expiry_warning_days,omitempty"`

	// SSL cert config
	ValidateChain bool `json:"validate_chain,omitempty" yaml:"validate_chain,omitempty"`

	// Database config
	PgSQL *DBImportConfig `json:"pgsql,omitempty" yaml:"pgsql,omitempty"`
	MySQL *DBImportConfig `json:"mysql,omitempty" yaml:"mysql,omitempty"`

	// SSH config
	ExpectBanner string `json:"expect_banner,omitempty" yaml:"expect_banner,omitempty"`

	// Redis config
	RedisPassword string `json:"redis_password,omitempty" yaml:"redis_password,omitempty"`
	RedisDB       int    `json:"redis_db,omitempty" yaml:"redis_db,omitempty"`

	// SMTP config
	StartTLS     bool   `json:"starttls,omitempty" yaml:"starttls,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty" yaml:"smtp_username,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty" yaml:"smtp_password,omitempty"`

	// gRPC config
	UseTLS bool `json:"use_tls,omitempty" yaml:"use_tls,omitempty"`

	// MongoDB config
	MongoDBURI string `json:"mongodb_uri,omitempty" yaml:"mongodb_uri,omitempty"`

	// Alert config
	ActorType        string   `json:"actor_type,omitempty" yaml:"actor_type,omitempty"`
	AlertType        string   `json:"alert_type,omitempty" yaml:"alert_type,omitempty"`
	AlertDestination string   `json:"alert_destination,omitempty" yaml:"alert_destination,omitempty"`
	Severity         string   `json:"severity,omitempty" yaml:"severity,omitempty"`
	AlertChannels    []string `json:"alert_channels,omitempty" yaml:"alert_channels,omitempty"`
	ReAlertInterval  string   `json:"re_alert_interval,omitempty" yaml:"re_alert_interval,omitempty"`
	RetryCount       int      `json:"retry_count,omitempty" yaml:"retry_count,omitempty"`
	RetryInterval    string   `json:"retry_interval,omitempty" yaml:"retry_interval,omitempty"`
}

// DBImportConfig holds database-specific import fields
type DBImportConfig struct {
	UserName   string   `json:"username,omitempty" yaml:"username,omitempty"`
	DBName     string   `json:"dbname,omitempty" yaml:"dbname,omitempty"`
	Query      string   `json:"query,omitempty" yaml:"query,omitempty"`
	ServerList []string `json:"server_list,omitempty" yaml:"server_list,omitempty"`
}

// CheckImportResult is returned by the import endpoint
type CheckImportResult struct {
	Created []CheckImportResultItem `json:"created"`
	Updated []CheckImportResultItem `json:"updated"`
	Deleted []CheckImportResultItem `json:"deleted"`
	Errors  []CheckImportError      `json:"errors"`
	Summary CheckImportSummary      `json:"summary"`
}

// CheckImportResultItem describes a single check that was processed
type CheckImportResultItem struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Project string `json:"project"`
}

// CheckImportError describes a check that failed to import
type CheckImportError struct {
	Name    string `json:"name"`
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// CheckImportSummary is a quick count of what happened
type CheckImportSummary struct {
	Total   int `json:"total"`
	Created int `json:"created"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
	Errors  int `json:"errors"`
}
