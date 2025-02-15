package config

import (
	"fmt"
	"io/ioutil"
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

	Projects map[string]struct {
		Parameters struct {
			Duration time.Duration `yaml:"duration"`
		} `yaml:"parameters"`
		HealthChecks map[string]struct {
			Checks map[string]struct {
				Type          string `yaml:"type"`
				URL           string `yaml:"url,omitempty"`
				AnswerPresent bool   `yaml:"answer_present,omitempty"`
				// Add more config fields for TCP, ping, etc.
			} `yaml:"checks"`
		} `yaml:"healthchecks"`
	} `yaml:"projects"`
}

// LoadConfig reads the YAML file and unmarshals into Config struct
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}
