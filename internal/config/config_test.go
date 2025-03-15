package config

import (
	"errors"
	"os"
	"testing"
	"time"
)

const testConfigYAML = `
defaults:
  duration: 5s
  alerts_channel: "telegram"
  maintenance_duration: 1m
server_port: "9090"
db:
  protocol: "mongodb"
  host: "localhost"
  username: "tester"
  database: "test_db"
  password: "secret"
alerts:
  telegram:
    type: "telegram"
    bot_token: "test_token"
    critical_channel: "critical"
    noncritical_channel: "noncritical"
  slack:
    type: "slack"
projects:
  project1:
    parameters:
      duration: 2s
    healthchecks:
      default:
        parameters:
          duration: 3s
`

const invalidConfigYAML = `
defaults:
  duration: invalid
server_port: "not_a_port"
db:
  protocol: "unsupported"
projects: {}
`

func TestLoadConfig(t *testing.T) {
	// Write test config to a temporary file
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.yaml"
	err := os.WriteFile(configPath, []byte(testConfigYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary config file: %v", err)
	}

	// Load the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Test defaults
	t.Run("Defaults", func(t *testing.T) {
		if cfg.Defaults.Duration != 5*time.Second {
			t.Errorf("Expected default duration 5s, got %v", cfg.Defaults.Duration)
		}
		if cfg.Defaults.MaintenanceDuration != time.Minute {
			t.Errorf("Expected maintenance duration 1m, got %v", cfg.Defaults.MaintenanceDuration)
		}
		if cfg.Defaults.AlertsChannel != "telegram" {
			t.Errorf("Expected alerts channel 'telegram', got %s", cfg.Defaults.AlertsChannel)
		}
	})

	// Test database configuration
	t.Run("Database", func(t *testing.T) {
		if cfg.DB.Protocol != "mongodb" {
			t.Errorf("Expected DB protocol 'mongodb', got %s", cfg.DB.Protocol)
		}
		if cfg.DB.Host != "localhost" {
			t.Errorf("Expected DB host 'localhost', got %s", cfg.DB.Host)
		}
		if cfg.DB.Username != "tester" {
			t.Errorf("Expected DB username 'tester', got %s", cfg.DB.Username)
		}
		if cfg.DB.Database != "test_db" {
			t.Errorf("Expected DB name 'test_db', got %s", cfg.DB.Database)
		}
		if cfg.DB.Password != "secret" {
			t.Errorf("Expected DB password 'secret', got %s", cfg.DB.Password)
		}
	})

	// Test alerts configuration
	t.Run("Alerts", func(t *testing.T) {
		if len(cfg.Alerts) != 2 {
			t.Errorf("Expected 2 alert types, got %d", len(cfg.Alerts))
		}
		if cfg.Alerts["telegram"].Type != "telegram" {
			t.Errorf("Expected telegram alert type 'telegram', got %s", cfg.Alerts["telegram"].Type)
		}
		if cfg.Alerts["slack"].Type != "slack" {
			t.Errorf("Expected slack alert type 'slack', got %s", cfg.Alerts["slack"].Type)
		}
	})

	// Test project configuration
	t.Run("Projects", func(t *testing.T) {
		project, ok := cfg.Projects["project1"]
		if !ok {
			t.Fatal("Expected project 'project1' in config")
		}

		// Test project parameters
		if project.Parameters.Duration != 2*time.Second {
			t.Errorf("Expected project duration 2s, got %v", project.Parameters.Duration)
		}

		// Test healthchecks parameters
		hc, ok := project.HealthChecks["default"]
		if !ok {
			t.Fatal("Expected healthcheck group 'default' in project")
		}
		if hc.Parameters.Duration != 3*time.Second {
			t.Errorf("Expected healthcheck duration 3s, got %v", hc.Parameters.Duration)
		}
	})
}

// TestLoadConfig_FileNotFound verifies that LoadConfig returns an error when the file is missing
func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}

// TestLoadConfig_InvalidYAML verifies that LoadConfig returns an error for invalid YAML
func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/invalid_config.yaml"
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary config file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

// TestLoadConfig_InvalidConfig verifies that LoadConfig returns an error for invalid configuration values
func TestLoadConfig_InvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/invalid_config.yaml"
	err := os.WriteFile(configPath, []byte(invalidConfigYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary config file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid configuration, got nil")
	}
}

// TestLoadConfig_EmptyConfig verifies that LoadConfig handles empty configuration files
func TestLoadConfig_EmptyConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/empty_config.yaml"
	err := os.WriteFile(configPath, []byte("projects: {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed for empty config: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil config for empty file")
	}
}
