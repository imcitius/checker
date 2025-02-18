package config

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config struct representing the YAML
type Config struct {
	Defaults struct {
		Duration            time.Duration `yaml:"duration"`
		AlertsChannel       string        `yaml:"alerts_channel"`
		MaintenanceDuration time.Duration `yaml:"maintenance_duration"`
	} `yaml:"defaults"`
	ServerPort string `yaml:"server_port"`

	DB struct {
		Protocol string `yaml:"protocol"`
		Host     string `yaml:"host"`
		Username string `yaml:"username"`
		Database string `yaml:"database"`
		Password string `yaml:"password,omitempty"`
	} `yaml:"db"`

	Alerts map[string]struct {
		Type               string `yaml:"type"`
		BotToken           string `yaml:"bot_token,omitempty"`
		CriticalChannel    string `yaml:"critical_channel,omitempty"`
		NoncriticalChannel string `yaml:"noncritical_channel,omitempty"`
		// add Slack configs, etc.
	} `yaml:"alerts"`

	Tickers map[string]TTickerWithDuration

	Projects map[string]ProjectConfig `yaml:"projects"`
}

// ProjectConfig represents a project's configuration
type ProjectConfig struct {
	Parameters struct {
		Duration time.Duration `yaml:"duration"`
	} `yaml:"parameters"`
	// Changed HealthChecks from a single HealthCheckConfig to a map,
	// so that each key (for example "http", "ping", "tcp") can have its own parameters.
	HealthChecks map[string]HealthCheckConfig `yaml:"healthchecks"`
}

// HealthCheckConfig represents a group of health checks and its parameters.
type HealthCheckConfig struct {
	Parameters struct {
		Duration time.Duration `yaml:"duration"`
	} `yaml:"parameters"`
	// Checks belonging to this group
	Checks map[string]CheckConfig `yaml:"checks"`
}

// CheckConfig represents an individual check's configuration
type CheckConfig struct {
	Type          string              `yaml:"type"`
	UUID          string              `yaml:"uuid,omitempty"`
	Name          string              `yaml:"name,omitempty"`
	URL           string              `yaml:"url,omitempty"`
	Host          string              `yaml:"host,omitempty"`
	Timeout       string              `yaml:"timeout" env-default:"3s"`
	Port          int                 `yaml:"port,omitempty"`
	Hash          string              `yaml:"hash,omitempty"`
	Size          int64               `yaml:"size,omitempty"`
	Attempts      int                 `yaml:"attempts,omitempty"`
	Mode          string              `yaml:"mode,omitempty"`
	Count         int                 `yaml:"count" env-default:"3"`
	AllowFails    int                 `yaml:"allow_fails"`
	Code          []int               `yaml:"code,omitempty"`
	Answer        string              `yaml:"answer,omitempty"`
	AnswerPresent bool                `yaml:"answer_present,omitempty"`
	Headers       []map[string]string `yaml:"headers,omitempty"`
	Auth          struct {
		User     string `yaml:"user,omitempty"`
		Password string `yaml:"password,omitempty"`
	} `yaml:"auth,omitempty"`
	SkipCheckSSL        bool          `yaml:"skip_check_ssl" env-default:"false"`
	StopFollowRedirects bool          `yaml:"stop_follow_redirects" env-default:"false"`
	SSLExpirationPeriod string        `yaml:"ssl_expiration_period,omitempty"`
	Cookies             []http.Cookie `yaml:"cookies,omitempty"`
	DebugLevel          string        `yaml:"debug_level,omitempty"`
	Parameters          struct {
		Mode                string        `yaml:"mode" env-default:"loud"`
		Duration            time.Duration `yaml:"duration" env-default:"60s"`
		Timeout             string        `yaml:"timeout" env-default:"3s"`
		SSLExpirationPeriod string        `yaml:"ssl_expiration_period" env-default:"360h"`
		MinHealth           int           `yaml:"min_health" env-default:"1"`
		AllowFails          int           `yaml:"allow_fails" env-default:"0"`
		AlerterName         string        `yaml:"alerter,omitempty"`
		AlertChannel        string        `yaml:"noncrit_channel,omitempty"`
		CritAlertChannel    string        `yaml:"crit_channel,omitempty"`
		CommandChannel      string        `yaml:"command_channel,omitempty"`
		Mentions            []string      `yaml:"mentions,omitempty"`
	} `yaml:"parameters,omitempty"`
}

type TTickerWithDuration struct {
	Ticker   *time.Ticker
	Duration string
}

// LoadConfig reads the YAML file specified by filename and returns a pointer
// to a Config struct, or an error if reading or unmarshalling fails.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}

	cfg.setDefaults()
	cfg.setTickers()

	return &cfg, nil
}

// SetDefaults assigns default values to Config fields if they are not set
func (cfg *Config) setDefaults() {
	// Set Defaults

	// Set global defaults if not provided.
	if cfg.Defaults.Duration == 0 {
		cfg.Defaults.Duration = 10 * time.Second
	}
	if cfg.Defaults.AlertsChannel == "" {
		cfg.Defaults.AlertsChannel = "telegram"
	}
	if cfg.Defaults.MaintenanceDuration == 0 {
		cfg.Defaults.MaintenanceDuration = 15 * time.Minute
	}

	// Set DB defaults.
	if cfg.DB.Protocol == "" {
		cfg.DB.Protocol = "mongodb"
	}
	if cfg.DB.Host == "" {
		cfg.DB.Host = "localhost"
	}
	if cfg.DB.Username == "" {
		cfg.DB.Username = "checker-dev"
	}
	if cfg.DB.Database == "" {
		cfg.DB.Database = "checker_dev"
	}

	// Set Alerts defaults.
	if cfg.Alerts == nil {
		cfg.Alerts = make(map[string]struct {
			Type               string `yaml:"type"`
			BotToken           string `yaml:"bot_token,omitempty"`
			CriticalChannel    string `yaml:"critical_channel,omitempty"`
			NoncriticalChannel string `yaml:"noncritical_channel,omitempty"`
		})
	}
	if _, exists := cfg.Alerts["telegram"]; !exists {
		cfg.Alerts["telegram"] = struct {
			Type               string `yaml:"type"`
			BotToken           string `yaml:"bot_token,omitempty"`
			CriticalChannel    string `yaml:"critical_channel,omitempty"`
			NoncriticalChannel string `yaml:"noncritical_channel,omitempty"`
		}{
			Type: "telegram",
		}
	}
	if _, exists := cfg.Alerts["slack"]; !exists {
		cfg.Alerts["slack"] = struct {
			Type               string `yaml:"type"`
			BotToken           string `yaml:"bot_token,omitempty"`
			CriticalChannel    string `yaml:"critical_channel,omitempty"`
			NoncriticalChannel string `yaml:"noncritical_channel,omitempty"`
		}{
			Type: "slack",
		}
	}

	// Set Projects defaults.
	if cfg.Projects == nil {
		log.Fatalf("Projects not found in config file")
	}
	for projectName, project := range cfg.Projects {
		// Set project duration default if not provided.
		if project.Parameters.Duration == 0 {
			project.Parameters.Duration = cfg.Defaults.Duration
		}

		// Initialize health check groups map if nil.
		if project.HealthChecks == nil {
			project.HealthChecks = make(map[string]HealthCheckConfig)
		}

		// Iterate over each health check group.
		for groupName, group := range project.HealthChecks {
			if group.Checks == nil {
				group.Checks = make(map[string]CheckConfig)
			}
			// Set defaults for each check inside the group.
			for checkName, check := range group.Checks {
				if check.Name == "" {
					check.Name = checkName
				}
				if check.UUID == "" {
					check.UUID = genUUID(check.Name, hostOrUrl(check))
				}
				group.Checks[checkName] = check
			}
			project.HealthChecks[groupName] = group
		}
		cfg.Projects[projectName] = project
	}
}

// create function getTickers that returns a map of all configured tickers based on the configuration.
func (cfg *Config) setTickers() {

	tickers := make(map[string]TTickerWithDuration)
	// Create a default ticker based on the default duration.
	defaultDuration := cfg.Defaults.Duration

	tickers[cfg.Defaults.Duration.String()] = TTickerWithDuration{
		time.NewTicker(defaultDuration),
		defaultDuration.String(),
	}

	// Iterate over projects and health checks to create tickers for each duration.
	for _, project := range cfg.Projects {
		if project.Parameters.Duration > 0 {
			tickers[project.Parameters.Duration.String()] = TTickerWithDuration{
				time.NewTicker(project.Parameters.Duration),
				project.Parameters.Duration.String(),
			}
		}
		for _, group := range project.HealthChecks {
			if group.Parameters.Duration > 0 {
				tickers[group.Parameters.Duration.String()] = TTickerWithDuration{
					time.NewTicker(group.Parameters.Duration),
					group.Parameters.Duration.String(),
				}
			}
			for _, check := range group.Checks {
				if check.Parameters.Duration > 0 {
					tickers[check.Parameters.Duration.String()] = TTickerWithDuration{
						time.NewTicker(check.Parameters.Duration),
						check.Parameters.Duration.String(),
					}
				}
			}
		}
	}
	cfg.Tickers = tickers
}

func genUUID(name ...string) string {
	var err error

	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	if err != nil {
		return ""
	}

	u2 := uuid.NewSHA1(ns, []byte(strings.Join(name, ".")))
	return u2.String()
}

func hostOrUrl(c CheckConfig) string {
	if c.URL != "" {
		return c.URL
	}
	return c.Host
}
