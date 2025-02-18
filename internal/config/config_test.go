package config

import (
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
  slack:
    type: "slack"
projects:
  project1:
    parameters:
      duration: 2s
    healthchecks:
      parameters:
        duration: 3s
      checks:
        check1:
          type: "http"
          name: "TestCheck"
          url: "https://example.com"
`

func TestLoadConfig(t *testing.T) {
	// Write test config to a temporary file.
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.yaml"
	err := os.WriteFile(configPath, []byte(testConfigYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary config file: %v", err)
	}

	// Call the exported LoadConfig function.
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Expected non-nil config, got error: %v", err)
	}

	// Check that defaults were properly set
	if cfg.Defaults.Duration != 5*time.Second {
		t.Errorf("Expected default duration 5s, got %v", cfg.Defaults.Duration)
	}
	if cfg.Defaults.MaintenanceDuration != 1*time.Minute {
		t.Errorf("Expected maintenance duration 1m, got %v", cfg.Defaults.MaintenanceDuration)
	}
	if cfg.DB.Username != "tester" {
		t.Errorf("Expected DB username 'tester', got %s", cfg.DB.Username)
	}

	// Check project configuration
	project, ok := cfg.Projects["project1"]
	if !ok {
		t.Fatal("Expected project 'project1' in config")
	}
	if project.Parameters.Duration != 2*time.Second {
		t.Errorf("Expected project duration 2s, got %v", project.Parameters.Duration)
	}
	hc, ok := project.HealthChecks["default"]
	if !ok {
		t.Fatal("Expected healthcheck group 'default' in project")
	}
	if hc.Parameters.Duration != 3*time.Second {
		t.Errorf("Expected healthcheck duration 3s, got %v", hc.Parameters.Duration)
	}
	checkConfig, ok := hc.Checks["check1"]
	if !ok {
		t.Fatal("Expected check 'check1' in project1 healthchecks")
	}
	if checkConfig.Type != "http" {
		t.Errorf("Expected check type 'http', got %s", checkConfig.Type)
	}
}

// TestLoadConfig_FileNotFound verifies that LoadConfig returns an error when the file is missing.
func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}
