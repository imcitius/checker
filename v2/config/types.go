package config

import (
	checks "my/checker/models/checks"
	alerts "my/checker/models/alerts"
	"time"
)

type TConfig struct {
	Defaults  TDefaults                  `yaml:"defaults"`
	DB        DBConfig            `yaml:"db"`
	Alerts    map[string]alerts.TAlert   `yaml:"alerts"`
	Projects  map[string]TProject `yaml:"projects"`
	StartTime time.Time
}

type TDefaults struct {
	TimerStep              string                  `yaml:"timer_step" env-default:"1s"`
	Duration               string                  `yaml:"duration" env-default:"60s"`
	MaintenanceDuration    string                  `yaml:"maintenance_duration" env-default:"60s"`
	HTTPPort               string                  `yaml:"http_port" env-default:"80"`
	HTTPEnabled            string                  `yaml:"http_enabled" env-default:"true"`
	TokenEncryptionKey     []byte                  `yaml:"token_encryption_key"`
	BotsEnabled            bool                    `yaml:"bots_enabled" env-default:"true"`
	BotGreetingEnabled     bool                    `yaml:"bots_greeting_enabled" env-default:"false"`
	DebugLevel             string                  `yaml:"debug_level" env-default:"info"`
	AlertsChannel          string                  `yaml:"alerts_channel" env-default:"log"`
	DefaultCheckParameters checks.TCheckParameters `yaml:"default_check_parameters"`
}

type TProject struct {
	Name         string                  `yaml:"name"`
	Healthchecks map[string]checks.THealthcheck `yaml:"healthchecks"`
	Parameters   checks.TCheckParameters `yaml:"parameters"`
}

type THealthcheck struct {
	Name       string                         `yaml:"name"`
	Parameters checks.TCheckParameters        `yaml:"parameters"`
	Checks     map[string]checks.TCheckConfig `yaml:"checks"`
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


type TProjectsConfig struct {
	Projects *map[string]TProject `mapstructure:"projects"`
}
