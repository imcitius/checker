// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/imcitius/checker/pkg/alerts"
	"github.com/imcitius/checker/pkg/checks"
	"github.com/imcitius/checker/pkg/config"
	"github.com/imcitius/checker/pkg/models"
)

// runTestRun loads checks from the config file, runs each one once, dispatches
// alerts for failures/recoveries, and returns an exit code (0 = all pass, 1 = any fail).
func runTestRun(cfg *config.Config, configPath string) int {
	// 1. Load checks from the seed/config YAML file
	checkDefs, err := loadChecksFromFile(configPath)
	if err != nil {
		logrus.Errorf("Failed to load checks: %v", err)
		return 1
	}

	if len(checkDefs) == 0 {
		logrus.Warn("No checks found in config file")
		return 0
	}

	// 2. Build alerters from config's Alerts map + any alert_channels on checks
	alerterMap := buildAlertersFromConfig(cfg)

	// 3. Run each enabled check
	var passed, failed int
	for _, def := range checkDefs {
		if !def.Enabled {
			logrus.Debugf("Skipping disabled check: %s", def.Name)
			continue
		}

		logger := logrus.WithFields(logrus.Fields{
			"check": def.Name,
			"type":  def.Type,
		})

		checker := checks.CheckerFactory(def, logger)
		if checker == nil {
			logger.Warnf("Could not create checker for type %q, skipping", def.Type)
			failed++
			fmt.Printf("  SKIP  %s (%s) - unsupported check type\n", def.Name, def.Type)
			continue
		}

		duration, runErr := checker.Run()
		if runErr != nil {
			failed++
			fmt.Printf("  FAIL  %s (%s) - %s [%s]\n", def.Name, def.Type, runErr.Error(), duration)
			logger.Warnf("Check failed: %v (took %s)", runErr, duration)
			dispatchTestAlerts(alerterMap, def, runErr.Error())
		} else {
			passed++
			fmt.Printf("  PASS  %s (%s) [%s]\n", def.Name, def.Type, duration)
			logger.Infof("Check passed (took %s)", duration)
			dispatchTestRecoveries(alerterMap, def)
		}
	}

	// 4. Print summary
	total := passed + failed
	fmt.Printf("\n--- Test Run Summary ---\n")
	fmt.Printf("Total: %d | Passed: %d | Failed: %d\n", total, passed, failed)

	if failed > 0 {
		return 1
	}
	return 0
}

// loadChecksFromFile reads the YAML config/seed file and returns CheckDefinitions.
func loadChecksFromFile(filePath string) ([]models.CheckDefinition, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", filePath, err)
	}

	var payload models.CheckImportPayload
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if len(payload.Checks) == 0 {
		return nil, nil
	}

	defs := make([]models.CheckDefinition, 0, len(payload.Checks))
	for _, item := range payload.Checks {
		// Apply payload-level defaults
		if item.Project == "" && payload.Project != "" {
			item.Project = payload.Project
		}
		if item.GroupName == "" && payload.Environment != "" {
			item.GroupName = payload.Environment
		}

		def := seedItemToCheckDefinition(item)
		def.UUID = uuid.New().String()
		def.CreatedAt = time.Now()
		def.UpdatedAt = time.Now()
		defs = append(defs, def)
	}

	return defs, nil
}

// buildAlertersFromConfig creates alerter instances from the config's Alerts map.
// Only channels with registered types are included (e.g. test_report, email, etc.).
func buildAlertersFromConfig(cfg *config.Config) map[string]alerts.Alerter {
	result := make(map[string]alerts.Alerter)

	for name, alertCfg := range cfg.Alerts {
		channelType := alertCfg.Type
		if channelType == "" {
			channelType = name
		}

		if !alerts.IsRegisteredType(channelType) {
			logrus.Debugf("Alert channel %q (type %q) not registered, skipping", name, channelType)
			continue
		}

		// Convert the alert config struct to JSON for the factory
		raw, err := json.Marshal(alertCfg)
		if err != nil {
			logrus.Warnf("Failed to marshal alert config %q: %v", name, err)
			continue
		}

		alerter, err := alerts.NewAlerter(channelType, raw)
		if err != nil {
			logrus.Debugf("Failed to create alerter %q (type %q): %v", name, channelType, err)
			continue
		}

		logrus.Infof("Initialized alert channel: %s (type: %s)", name, channelType)
		result[name] = alerter
	}

	return result
}

// dispatchTestAlerts sends failure alerts to all configured alerters.
func dispatchTestAlerts(alerterMap map[string]alerts.Alerter, def models.CheckDefinition, message string) {
	payload := alerts.AlertPayload{
		CheckName:  def.Name,
		CheckUUID:  def.UUID,
		Project:    def.Project,
		CheckGroup: def.GroupName,
		CheckType:  def.Type,
		Message:    message,
		Severity:   effectiveSeverity(def),
		Timestamp:  time.Now(),
	}

	for name, alerter := range alerterMap {
		if err := alerter.SendAlert(payload); err != nil {
			logrus.Warnf("Failed to send alert via %q: %v", name, err)
		}
	}
}

// dispatchTestRecoveries sends recovery notifications to all configured alerters.
func dispatchTestRecoveries(alerterMap map[string]alerts.Alerter, def models.CheckDefinition) {
	payload := alerts.RecoveryPayload{
		CheckName:  def.Name,
		CheckUUID:  def.UUID,
		Project:    def.Project,
		CheckGroup: def.GroupName,
		CheckType:  def.Type,
		Timestamp:  time.Now(),
	}

	for name, alerter := range alerterMap {
		if err := alerter.SendRecovery(payload); err != nil {
			logrus.Warnf("Failed to send recovery via %q: %v", name, err)
		}
	}
}

// effectiveSeverity returns the check's severity or "critical" as default.
func effectiveSeverity(def models.CheckDefinition) string {
	if def.Severity != "" {
		return def.Severity
	}
	return "critical"
}
