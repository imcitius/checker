package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
		Protocol    string `yaml:"protocol"`
		Host        string `yaml:"host"`
		Username    string `yaml:"username"`
		Database    string `yaml:"database"`
		Password    string `yaml:"password,omitempty"`
		DatabaseURL string `yaml:"database_url,omitempty"`
	} `yaml:"db"`

	Alerts map[string]struct {
		Type               string   `yaml:"type"`
		BotToken           string   `yaml:"bot_token,omitempty"`
		CriticalChannel    string   `yaml:"critical_channel,omitempty"`
		NoncriticalChannel string   `yaml:"noncritical_channel,omitempty"`
		WebhookURL         string   `yaml:"webhook_url,omitempty"`
		RoutingKey         string   `yaml:"routing_key,omitempty"`
		// Opsgenie configs
		APIKey string `yaml:"api_key,omitempty"`
		Region string `yaml:"region,omitempty"` // "us" or "eu"
		// Email (SMTP) configs
		SMTPHost     string   `yaml:"smtp_host,omitempty"`
		SMTPPort     int      `yaml:"smtp_port,omitempty"`
		SMTPUser     string   `yaml:"smtp_user,omitempty"`
		SMTPPassword string   `yaml:"smtp_password,omitempty"`
		From         string   `yaml:"from,omitempty"`
		To           []string `yaml:"to,omitempty"`
		UseTLS       bool     `yaml:"use_tls,omitempty"`
	} `yaml:"alerts"`

	Tickers map[string]TickerWithDuration

	SlackApp struct {
		BotToken       string `yaml:"bot_token"`
		SigningSecret  string `yaml:"signing_secret"`
		DefaultChannel string `yaml:"default_channel"`
	} `yaml:"slack_app"`

	Auth struct {
		OIDC struct {
			IssuerURL    string `yaml:"issuer_url"`
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			RedirectURL  string `yaml:"redirect_url"`
		} `yaml:"oidc"`
		Password  string   `yaml:"password"`
		APIKeys   []string `yaml:"api_keys"`
		JWTSecret string   `yaml:"jwt_secret"`
	} `yaml:"auth"`

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
	Type          string
	UUID          string
	Name          string
	Description   string
	URL           string
	Host          string
	Timeout       string
	Port          int
	Hash          string
	Size          int64
	Attempts      int
	Mode          string
	Count         int
	AllowFails    int
	Code          []int
	Answer        string
	AnswerPresent bool
	Headers       []map[string]string
	Auth          struct {
		User     string
		Password string
	}
	SkipCheckSSL        bool
	StopFollowRedirects bool
	SSLExpirationPeriod string
	Cookies             []http.Cookie
	DebugLevel          string
	Parameters          struct {
		Mode                string
		Duration            time.Duration
		Timeout             string
		SSLExpirationPeriod string
		MinHealth           int
		AllowFails          int
		AlerterName         string
		AlertChannel        string
		CritAlertChannel    string
		CommandChannel      string
		Mentions            []string
	} `yaml:"parameters,omitempty"`

	ActorType string
	AlertType string
	Logger    *logrus.Entry
}

// TickerWithDuration associates a ticker with its duration string representation.
type TickerWithDuration struct {
	Ticker   *time.Ticker
	Duration string
}

type ActorConfig struct {
	Type    string `yaml:"type"`
	Message string
}

// LoadConfig reads the YAML file specified by filename and returns a pointer
// to a Config struct, or an error if reading or unmarshalling fails.
func LoadConfig(filename string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Warnf("Config file %s not found, using defaults + env overrides", filename)
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("error unmarshalling config file: %w", err)
		}
	}

	cfg.applyEnvOverrides()
	cfg.setDefaults()
	cfg.setTickers()

	return &cfg, nil
}

func (cfg *Config) applyEnvOverrides() {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DB.DatabaseURL = v
	}
	if v := os.Getenv("PGHOST"); v != "" {
		cfg.DB.Host = v
	}
	if v := os.Getenv("PGUSER"); v != "" {
		cfg.DB.Username = v
	}
	if v := os.Getenv("PGPASSWORD"); v != "" {
		cfg.DB.Password = v
	}
	if v := os.Getenv("PGDATABASE"); v != "" {
		cfg.DB.Database = v
	}
	if v := os.Getenv("SLACK_BOT_TOKEN"); v != "" {
		cfg.SlackApp.BotToken = v
	}
	if v := os.Getenv("SLACK_SIGNING_SECRET"); v != "" {
		cfg.SlackApp.SigningSecret = v
	}
	if v := os.Getenv("SLACK_DEFAULT_CHANNEL"); v != "" {
		cfg.SlackApp.DefaultChannel = v
	}

	// Auth overrides
	if v := os.Getenv("AUTH_OIDC_ISSUER_URL"); v != "" {
		cfg.Auth.OIDC.IssuerURL = v
	}
	if v := os.Getenv("AUTH_OIDC_CLIENT_ID"); v != "" {
		cfg.Auth.OIDC.ClientID = v
	}
	if v := os.Getenv("AUTH_OIDC_CLIENT_SECRET"); v != "" {
		cfg.Auth.OIDC.ClientSecret = v
	}
	if v := os.Getenv("AUTH_OIDC_REDIRECT_URL"); v != "" {
		cfg.Auth.OIDC.RedirectURL = v
	}
	if v := os.Getenv("AUTH_PASSWORD"); v != "" {
		cfg.Auth.Password = v
	}
	if v := os.Getenv("AUTH_API_KEYS"); v != "" {
		cfg.Auth.APIKeys = strings.Split(v, ",")
	}
	if v := os.Getenv("AUTH_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
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
			Type               string   `yaml:"type"`
			BotToken           string   `yaml:"bot_token,omitempty"`
			CriticalChannel    string   `yaml:"critical_channel,omitempty"`
			NoncriticalChannel string   `yaml:"noncritical_channel,omitempty"`
			WebhookURL         string   `yaml:"webhook_url,omitempty"`
			RoutingKey         string   `yaml:"routing_key,omitempty"`
			APIKey             string   `yaml:"api_key,omitempty"`
			Region             string   `yaml:"region,omitempty"`
			SMTPHost           string   `yaml:"smtp_host,omitempty"`
			SMTPPort           int      `yaml:"smtp_port,omitempty"`
			SMTPUser           string   `yaml:"smtp_user,omitempty"`
			SMTPPassword       string   `yaml:"smtp_password,omitempty"`
			From               string   `yaml:"from,omitempty"`
			To                 []string `yaml:"to,omitempty"`
			UseTLS             bool     `yaml:"use_tls,omitempty"`
		})
	}
	if _, exists := cfg.Alerts["telegram"]; !exists {
		cfg.Alerts["telegram"] = struct {
			Type               string   `yaml:"type"`
			BotToken           string   `yaml:"bot_token,omitempty"`
			CriticalChannel    string   `yaml:"critical_channel,omitempty"`
			NoncriticalChannel string   `yaml:"noncritical_channel,omitempty"`
			WebhookURL         string   `yaml:"webhook_url,omitempty"`
			RoutingKey         string   `yaml:"routing_key,omitempty"`
			APIKey             string   `yaml:"api_key,omitempty"`
			Region             string   `yaml:"region,omitempty"`
			SMTPHost           string   `yaml:"smtp_host,omitempty"`
			SMTPPort           int      `yaml:"smtp_port,omitempty"`
			SMTPUser           string   `yaml:"smtp_user,omitempty"`
			SMTPPassword       string   `yaml:"smtp_password,omitempty"`
			From               string   `yaml:"from,omitempty"`
			To                 []string `yaml:"to,omitempty"`
			UseTLS             bool     `yaml:"use_tls,omitempty"`
		}{
			Type: "telegram",
		}
	}
	if _, exists := cfg.Alerts["slack"]; !exists {
		cfg.Alerts["slack"] = struct {
			Type               string   `yaml:"type"`
			BotToken           string   `yaml:"bot_token,omitempty"`
			CriticalChannel    string   `yaml:"critical_channel,omitempty"`
			NoncriticalChannel string   `yaml:"noncritical_channel,omitempty"`
			WebhookURL         string   `yaml:"webhook_url,omitempty"`
			RoutingKey         string   `yaml:"routing_key,omitempty"`
			APIKey             string   `yaml:"api_key,omitempty"`
			Region             string   `yaml:"region,omitempty"`
			SMTPHost           string   `yaml:"smtp_host,omitempty"`
			SMTPPort           int      `yaml:"smtp_port,omitempty"`
			SMTPUser           string   `yaml:"smtp_user,omitempty"`
			SMTPPassword       string   `yaml:"smtp_password,omitempty"`
			From               string   `yaml:"from,omitempty"`
			To                 []string `yaml:"to,omitempty"`
			UseTLS             bool     `yaml:"use_tls,omitempty"`
		}{
			Type: "slack",
		}
	}

	// Generate random JWT secret if auth is enabled but no secret provided
	if (cfg.Auth.OIDC.IssuerURL != "" || cfg.Auth.Password != "") && cfg.Auth.JWTSecret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			logrus.Errorf("Failed to generate random JWT secret: %v", err)
		} else {
			cfg.Auth.JWTSecret = hex.EncodeToString(b)
			logrus.Warn("AUTH_JWT_SECRET not set, generated random secret — sessions will not survive restarts")
		}
	}

	// Set Projects defaults.
	if cfg.Projects == nil {
		logrus.Warn("No projects found in config, starting with empty project list")
		cfg.Projects = make(map[string]ProjectConfig)
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
					check.UUID = generateUUID(check.Name, getHostOrURL(check))
				}
				group.Checks[checkName] = check
			}
			project.HealthChecks[groupName] = group
		}
		cfg.Projects[projectName] = project
	}
}

// setTickers creates tickers for all configured durations in the configuration.
func (cfg *Config) setTickers() {

	tickers := make(map[string]TickerWithDuration)
	// Create a default ticker based on the default duration.
	defaultDuration := cfg.Defaults.Duration

	tickers[cfg.Defaults.Duration.String()] = TickerWithDuration{
		time.NewTicker(defaultDuration),
		defaultDuration.String(),
	}

	// Iterate over projects and health checks to create tickers for each duration.
	for _, project := range cfg.Projects {
		if project.Parameters.Duration > 0 {
			tickers[project.Parameters.Duration.String()] = TickerWithDuration{
				time.NewTicker(project.Parameters.Duration),
				project.Parameters.Duration.String(),
			}
		}
		for _, group := range project.HealthChecks {
			if group.Parameters.Duration > 0 {
				tickers[group.Parameters.Duration.String()] = TickerWithDuration{
					time.NewTicker(group.Parameters.Duration),
					group.Parameters.Duration.String(),
				}
			}
			for _, check := range group.Checks {
				if check.Parameters.Duration > 0 {
					tickers[check.Parameters.Duration.String()] = TickerWithDuration{
						time.NewTicker(check.Parameters.Duration),
						check.Parameters.Duration.String(),
					}
				}
			}
		}
	}
	cfg.Tickers = tickers
}

// generateUUID creates a deterministic UUID based on the given name components.
func generateUUID(name ...string) string {
	var err error

	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	if err != nil {
		return ""
	}

	u2 := uuid.NewSHA1(ns, []byte(strings.Join(name, ".")))
	return u2.String()
}

// getHostOrURL returns the URL if set, otherwise returns the host address.
func getHostOrURL(c CheckConfig) string {
	if c.URL != "" {
		return c.URL
	}
	return c.Host
}
