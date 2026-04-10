// SPDX-License-Identifier: BUSL-1.1

package scheduler

import (
	"testing"

	"github.com/imcitius/checker/pkg/models"
)

func TestGetEffectiveAlertChannels_UsesAlertChannels(t *testing.T) {
	def := models.CheckDefinition{
		AlertChannels: []string{"telegram", "slack"},
	}
	channels := getEffectiveAlertChannels(def)
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	if channels[0] != "telegram" || channels[1] != "slack" {
		t.Fatalf("unexpected channels: %v", channels)
	}
}

func TestGetEffectiveAlertChannels_NoChannels(t *testing.T) {
	// Save and restore the package-level default
	saved := defaultAlertChannels
	defaultAlertChannels = nil
	defer func() { defaultAlertChannels = saved }()

	def := models.CheckDefinition{}
	channels := getEffectiveAlertChannels(def)
	if len(channels) != 0 {
		t.Fatalf("expected empty channels, got %v", channels)
	}
}

func TestGetEffectiveAlertChannels_FallsBackToSystemDefaults(t *testing.T) {
	// Save and restore the package-level default
	saved := defaultAlertChannels
	defaultAlertChannels = []string{"slack-ops", "telegram-alerts"}
	defer func() { defaultAlertChannels = saved }()

	def := models.CheckDefinition{} // no alert channels set
	channels := getEffectiveAlertChannels(def)
	if len(channels) != 2 {
		t.Fatalf("expected 2 default channels, got %d: %v", len(channels), channels)
	}
	if channels[0] != "slack-ops" || channels[1] != "telegram-alerts" {
		t.Fatalf("expected [slack-ops telegram-alerts], got %v", channels)
	}
}

func TestGetEffectiveAlertChannels_CheckChannelsOverrideDefaults(t *testing.T) {
	// Save and restore the package-level default
	saved := defaultAlertChannels
	defaultAlertChannels = []string{"slack-ops", "telegram-alerts"}
	defer func() { defaultAlertChannels = saved }()

	def := models.CheckDefinition{
		AlertChannels: []string{"pagerduty"},
	}
	channels := getEffectiveAlertChannels(def)
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d: %v", len(channels), channels)
	}
	if channels[0] != "pagerduty" {
		t.Fatalf("expected [pagerduty], got %v", channels)
	}
}

func TestGetEffectiveSeverity_Default(t *testing.T) {
	def := models.CheckDefinition{}
	sev := getEffectiveSeverity(def)
	if sev != "critical" {
		t.Fatalf("expected 'critical', got '%s'", sev)
	}
}

func TestGetEffectiveSeverity_Custom(t *testing.T) {
	def := models.CheckDefinition{Severity: "warning"}
	sev := getEffectiveSeverity(def)
	if sev != "warning" {
		t.Fatalf("expected 'warning', got '%s'", sev)
	}
}

func TestGetEffectiveSeverity_Info(t *testing.T) {
	def := models.CheckDefinition{Severity: "info"}
	sev := getEffectiveSeverity(def)
	if sev != "info" {
		t.Fatalf("expected 'info', got '%s'", sev)
	}
}

func TestGetEffectiveAlertChannels_MultiChannel(t *testing.T) {
	// Verify that a check with alert_channels: ["telegram", "discord"] returns both
	def := models.CheckDefinition{
		AlertChannels: []string{"telegram", "discord"},
	}
	channels := getEffectiveAlertChannels(def)
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	if channels[0] != "telegram" || channels[1] != "discord" {
		t.Fatalf("unexpected channels: %v", channels)
	}
}
