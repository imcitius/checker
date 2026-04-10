// SPDX-License-Identifier: BUSL-1.1

package web

import (
	"testing"

	"github.com/imcitius/checker/pkg/models"

	"gopkg.in/yaml.v3"
)

func TestResolveChecks_InheritsRootProject(t *testing.T) {
	payload := &models.CheckImportPayload{
		Project:     "my-service",
		Environment: "prod",
		Defaults: models.CheckImportDefaults{
			Duration: "30s",
			Timeout:  "10s",
		},
		Checks: []models.CheckImportItem{
			{Name: "API Health", Type: "http", URL: "https://example.com/healthz"},
			{Name: "DB Port", Type: "tcp", Host: "db.example.com", Port: 5432},
			{Name: "Cron Monitor", Type: "passive"},
		},
	}

	resolved := resolveChecks(payload)

	if len(resolved) != 3 {
		t.Fatalf("expected 3 resolved checks, got %d", len(resolved))
	}

	for i, check := range resolved {
		if check.Project != "my-service" {
			t.Errorf("check %d (%s): expected project 'my-service', got %q", i, check.Name, check.Project)
		}
		if check.GroupName != "prod" {
			t.Errorf("check %d (%s): expected group_name 'prod', got %q", i, check.Name, check.GroupName)
		}
		if check.Duration != "30s" {
			t.Errorf("check %d (%s): expected duration '30s', got %q", i, check.Name, check.Duration)
		}
		if check.Timeout != "10s" {
			t.Errorf("check %d (%s): expected timeout '10s', got %q", i, check.Name, check.Timeout)
		}
	}
}

func TestResolveChecks_CheckOverridesDefaults(t *testing.T) {
	payload := &models.CheckImportPayload{
		Project:     "default-project",
		Environment: "prod",
		Defaults: models.CheckImportDefaults{
			Duration: "30s",
			Timeout:  "5s",
		},
		Checks: []models.CheckImportItem{
			{
				Name:     "Custom Check",
				Type:     "http",
				Project:  "custom-project",
				Duration: "5m",
				Timeout:  "30s",
			},
		},
	}

	resolved := resolveChecks(payload)

	if resolved[0].Project != "custom-project" {
		t.Errorf("expected project 'custom-project', got %q", resolved[0].Project)
	}
	if resolved[0].Duration != "5m" {
		t.Errorf("expected duration '5m', got %q", resolved[0].Duration)
	}
	if resolved[0].Timeout != "30s" {
		t.Errorf("expected timeout '30s', got %q", resolved[0].Timeout)
	}
}

func TestValidateChecks_NoErrorsWhenProjectInherited(t *testing.T) {
	payload := &models.CheckImportPayload{
		Project:     "my-service",
		Environment: "prod",
		Defaults: models.CheckImportDefaults{
			Duration: "1m",
			Timeout:  "10s",
		},
		Checks: []models.CheckImportItem{
			{Name: "Website", Type: "http", URL: "https://example.com"},
			{Name: "API", Type: "http", URL: "https://api.example.com/health"},
			{Name: "TCP Check", Type: "tcp", Host: "db.example.com"},
			{Name: "Cron Monitor", Type: "passive"},
		},
	}

	resolved := resolveChecks(payload)
	errors := validateChecks(resolved)

	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got %d:", len(errors))
		for _, e := range errors {
			t.Errorf("  Check #%d (%s): %s", e.Index+1, e.Name, e.Message)
		}
	}
}

func TestValidateChecks_ErrorWhenProjectMissing(t *testing.T) {
	// No root project, no per-check project
	payload := &models.CheckImportPayload{
		Checks: []models.CheckImportItem{
			{Name: "No Project", Type: "http", URL: "https://example.com"},
		},
	}

	resolved := resolveChecks(payload)
	errors := validateChecks(resolved)

	found := false
	for _, e := range errors {
		if e.Message == "project is required" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'project is required' validation error")
	}
}

func TestParseAndResolve_FullYAML(t *testing.T) {
	yamlContent := `project: cyclops-production
environment: prod

defaults:
  duration: 1m
  timeout: 10s
  alert_type: slack
  alert_destination: "#checker-production"

checks:
  - name: "Website"
    type: http
    url: https://cyclops.io
  - name: "API Health"
    type: http
    url: https://api.cyclops.io/v1/health
  - name: "Ingress"
    type: http
    url: https://ingress.cyclops.io
    code:
      - 404
  - name: "SDK"
    type: http
    url: https://cyclops.io/sdk/v1/sdk-latest.js
    answer: "sdk-1.0"
  - name: "Cron Monitor"
    type: passive
    timeout: 15m
`

	var payload models.CheckImportPayload
	if err := yaml.Unmarshal([]byte(yamlContent), &payload); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	if payload.Project != "cyclops-production" {
		t.Errorf("expected root project 'cyclops-production', got %q", payload.Project)
	}
	if len(payload.Checks) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(payload.Checks))
	}

	resolved := resolveChecks(&payload)
	errors := validateChecks(resolved)

	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got %d:", len(errors))
		for _, e := range errors {
			t.Errorf("  Check #%d (%s): %s", e.Index+1, e.Name, e.Message)
		}
	}

	// Verify all checks inherited project
	for i, check := range resolved {
		if check.Project != "cyclops-production" {
			t.Errorf("check %d (%s): expected project 'cyclops-production', got %q", i, check.Name, check.Project)
		}
	}

	// Verify the code field was parsed for Ingress check
	if len(resolved[2].Code) != 1 || resolved[2].Code[0] != 404 {
		t.Errorf("Ingress check: expected code [404], got %v", resolved[2].Code)
	}

	// Verify the answer field was parsed for SDK check
	if resolved[3].Answer != "sdk-1.0" {
		t.Errorf("SDK check: expected answer 'sdk-1.0', got %q", resolved[3].Answer)
	}
}

func TestValidateChecks_DuplicateCheckNamesWarning(t *testing.T) {
	payload := &models.CheckImportPayload{
		Project:     "my-service",
		Environment: "prod",
		Checks: []models.CheckImportItem{
			{Name: "Dashboard", Type: "http", URL: "https://example.com/dash1"},
			{Name: "API Health", Type: "http", URL: "https://example.com/api"},
			{Name: "Dashboard", Type: "http", URL: "https://example.com/dash2"}, // duplicate
		},
	}

	resolved := resolveChecks(payload)
	errors := validateChecks(resolved)

	// Should have a duplicate warning for the second "Dashboard"
	found := false
	for _, e := range errors {
		if e.Index == 2 && e.Name == "Dashboard" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected duplicate check name warning for 'Dashboard' at index 2")
	}
}

func TestValidateChecks_NoDuplicateWarningForDifferentScopes(t *testing.T) {
	// Same name but different project+group = not duplicates
	checks := []models.CheckImportItem{
		{Name: "Health", Type: "http", URL: "https://a.com", Project: "svc-a", GroupName: "prod"},
		{Name: "Health", Type: "http", URL: "https://b.com", Project: "svc-b", GroupName: "prod"},
	}

	errors := validateChecks(checks)

	for _, e := range errors {
		if e.Message != "" && e.Name == "Health" && e.Index == 1 {
			t.Errorf("should not flag duplicate for different project scopes, got: %s", e.Message)
		}
	}
}

func TestResolveChecks_PayloadDefaultAlertChannels(t *testing.T) {
	payload := &models.CheckImportPayload{
		Project:     "my-service",
		Environment: "prod",
		Defaults: models.CheckImportDefaults{
			Duration:      "1m",
			AlertChannels: []string{"slack-ops", "pagerduty"},
		},
		Checks: []models.CheckImportItem{
			{Name: "Check A", Type: "http", URL: "https://example.com"},
			{Name: "Check B", Type: "http", URL: "https://example.com/b", AlertChannels: []string{"telegram"}},
		},
	}

	resolved := resolveChecks(payload)

	// Check A should inherit payload defaults
	if len(resolved[0].AlertChannels) != 2 || resolved[0].AlertChannels[0] != "slack-ops" {
		t.Errorf("Check A: expected payload default alert channels [slack-ops pagerduty], got %v", resolved[0].AlertChannels)
	}
	// Check B should keep its own
	if len(resolved[1].AlertChannels) != 1 || resolved[1].AlertChannels[0] != "telegram" {
		t.Errorf("Check B: expected own alert channels [telegram], got %v", resolved[1].AlertChannels)
	}
}

func TestApplySystemDefaultAlertChannels(t *testing.T) {
	checks := []models.CheckImportItem{
		{Name: "A", AlertChannels: nil},
		{Name: "B", AlertChannels: []string{"pagerduty"}},
		{Name: "C", AlertChannels: nil},
	}

	systemDefaults := []string{"slack-ops", "telegram"}
	applySystemDefaultAlertChannels(checks, systemDefaults)

	// A should get system defaults
	if len(checks[0].AlertChannels) != 2 || checks[0].AlertChannels[0] != "slack-ops" {
		t.Errorf("Check A: expected system defaults [slack-ops telegram], got %v", checks[0].AlertChannels)
	}
	// B should keep its own
	if len(checks[1].AlertChannels) != 1 || checks[1].AlertChannels[0] != "pagerduty" {
		t.Errorf("Check B: expected own [pagerduty], got %v", checks[1].AlertChannels)
	}
	// C should get system defaults
	if len(checks[2].AlertChannels) != 2 || checks[2].AlertChannels[0] != "slack-ops" {
		t.Errorf("Check C: expected system defaults [slack-ops telegram], got %v", checks[2].AlertChannels)
	}
}

func TestApplySystemDefaultAlertChannels_NoOp(t *testing.T) {
	checks := []models.CheckImportItem{
		{Name: "A", AlertChannels: []string{"discord"}},
		{Name: "B", AlertChannels: []string{"slack"}},
	}

	systemDefaults := []string{"slack-ops"}
	applySystemDefaultAlertChannels(checks, systemDefaults)

	// All checks already have channels, no changes
	if checks[0].AlertChannels[0] != "discord" {
		t.Errorf("Check A: should be unchanged, got %v", checks[0].AlertChannels)
	}
	if checks[1].AlertChannels[0] != "slack" {
		t.Errorf("Check B: should be unchanged, got %v", checks[1].AlertChannels)
	}
}

func TestImportItemToCheckDefinition_HTTPWithAllFields(t *testing.T) {
	ap := true
	sc := false
	sfr := true

	item := models.CheckImportItem{
		Name:    "Full HTTP Check",
		Project: "test",
		Type:    "http",
		URL:     "https://example.com",
		Timeout: "5s",
		Answer:  "ok",
		AnswerPresent: &ap,
		Code:    []int{200, 201},
		Headers: []map[string]string{{"Authorization": "Bearer token"}},
		Cookies: []map[string]string{{"session": "abc123"}},
		SkipCheckSSL: &sc,
		SSLExpirationPeriod: "30d",
		StopFollowRedirects: &sfr,
		Auth: &models.AuthImportConfig{
			User:     "admin",
			Password: "secret",
		},
	}

	def := importItemToCheckDefinition(item)

	httpCfg, ok := def.Config.(*models.HTTPCheckConfig)
	if !ok {
		t.Fatalf("expected HTTPCheckConfig, got %T", def.Config)
	}

	if httpCfg.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %q", httpCfg.URL)
	}
	if httpCfg.Answer != "ok" {
		t.Errorf("expected answer 'ok', got %q", httpCfg.Answer)
	}
	if !httpCfg.AnswerPresent {
		t.Error("expected answer_present true")
	}
	if len(httpCfg.Code) != 2 || httpCfg.Code[0] != 200 {
		t.Errorf("expected code [200, 201], got %v", httpCfg.Code)
	}
	if httpCfg.StopFollowRedirects != true {
		t.Error("expected stop_follow_redirects true")
	}
	if httpCfg.Auth.User != "admin" {
		t.Errorf("expected auth user 'admin', got %q", httpCfg.Auth.User)
	}
}

func TestImportItemToCheckDefinition_Passive(t *testing.T) {
	item := models.CheckImportItem{
		Name:    "Cron Monitor",
		Project: "test",
		Type:    "passive",
		Timeout: "15m",
	}

	def := importItemToCheckDefinition(item)

	passiveCfg, ok := def.Config.(*models.PassiveCheckConfig)
	if !ok {
		t.Fatalf("expected PassiveCheckConfig, got %T", def.Config)
	}
	if passiveCfg.Timeout != "15m" {
		t.Errorf("expected timeout '15m', got %q", passiveCfg.Timeout)
	}
}

func TestImportItemToCheckDefinition_ICMP(t *testing.T) {
	item := models.CheckImportItem{
		Name:    "Ping Test",
		Project: "test",
		Type:    "icmp",
		Host:    "server.example.com",
	}

	def := importItemToCheckDefinition(item)

	icmpCfg, ok := def.Config.(*models.ICMPCheckConfig)
	if !ok {
		t.Fatalf("expected ICMPCheckConfig, got %T", def.Config)
	}
	if icmpCfg.Host != "server.example.com" {
		t.Errorf("expected host 'server.example.com', got %q", icmpCfg.Host)
	}
}
