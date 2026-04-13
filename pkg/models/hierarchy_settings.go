// SPDX-License-Identifier: BUSL-1.1

package models

import "time"

// ProjectSettings holds project-level overrides.
// NULL fields inherit from system defaults.
type ProjectSettings struct {
	TenantID         string     `json:"-"`
	Project          string     `json:"project"`
	Enabled          *bool      `json:"enabled"`            // nil = inherit
	Duration         *string    `json:"duration"`           // nil = inherit
	ReAlertInterval  *string    `json:"re_alert_interval"`  // nil = inherit
	MaintenanceUntil *time.Time `json:"maintenance_until"`  // nil = not in maintenance
	MaintenanceReason string    `json:"maintenance_reason"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// GroupSettings holds group-level overrides.
// NULL fields inherit from project settings, then system defaults.
type GroupSettings struct {
	TenantID         string     `json:"-"`
	Project          string     `json:"project"`
	GroupName        string     `json:"group_name"`
	Enabled          *bool      `json:"enabled"`            // nil = inherit
	Duration         *string    `json:"duration"`           // nil = inherit
	ReAlertInterval  *string    `json:"re_alert_interval"`  // nil = inherit
	MaintenanceUntil *time.Time `json:"maintenance_until"`  // nil = not in maintenance
	MaintenanceReason string    `json:"maintenance_reason"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// EffectiveSettings are the resolved settings for a check after walking
// the hierarchy: check → group → project → system defaults.
type EffectiveSettings struct {
	Enabled          bool
	Duration         string
	ReAlertInterval  string
	InMaintenance    bool
	MaintenanceReason string
	MaintenanceUntil *time.Time
}

// IsInMaintenance returns true if the maintenance window is active.
func (ps *ProjectSettings) IsInMaintenance() bool {
	return ps != nil && ps.MaintenanceUntil != nil && ps.MaintenanceUntil.After(time.Now())
}

// IsInMaintenance returns true if the maintenance window is active.
func (gs *GroupSettings) IsInMaintenance() bool {
	return gs != nil && gs.MaintenanceUntil != nil && gs.MaintenanceUntil.After(time.Now())
}

// ResolveEffective walks the settings hierarchy and returns effective values.
// Resolution order: check → group → project → system defaults.
//
// For enabled: AND logic — disabled at any level means disabled.
// For maintenance: OR logic — maintenance at any level means in maintenance.
// For duration/re_alert_interval: first non-nil value wins walking up the chain.
func ResolveEffective(check CheckDefinition, groupSettings *GroupSettings, projectSettings *ProjectSettings, defaults CheckDefaults) EffectiveSettings {
	es := EffectiveSettings{
		Enabled: check.Enabled,
	}

	// Enabled: AND logic — if project or group disabled, check is disabled
	if projectSettings != nil && projectSettings.Enabled != nil && !*projectSettings.Enabled {
		es.Enabled = false
	}
	if groupSettings != nil && groupSettings.Enabled != nil && !*groupSettings.Enabled {
		es.Enabled = false
	}

	// Duration: check → group → project → defaults
	es.Duration = check.Duration
	if es.Duration == "" && groupSettings != nil && groupSettings.Duration != nil {
		es.Duration = *groupSettings.Duration
	}
	if es.Duration == "" && projectSettings != nil && projectSettings.Duration != nil {
		es.Duration = *projectSettings.Duration
	}
	if es.Duration == "" {
		es.Duration = defaults.CheckInterval
	}

	// ReAlertInterval: check → group → project → defaults
	es.ReAlertInterval = check.ReAlertInterval
	if es.ReAlertInterval == "" && groupSettings != nil && groupSettings.ReAlertInterval != nil {
		es.ReAlertInterval = *groupSettings.ReAlertInterval
	}
	if es.ReAlertInterval == "" && projectSettings != nil && projectSettings.ReAlertInterval != nil {
		es.ReAlertInterval = *projectSettings.ReAlertInterval
	}
	if es.ReAlertInterval == "" {
		es.ReAlertInterval = defaults.ReAlertInterval
	}

	// Maintenance: OR logic — any level in maintenance → in maintenance
	now := time.Now()
	if check.MaintenanceUntil != nil && check.MaintenanceUntil.After(now) {
		es.InMaintenance = true
		es.MaintenanceUntil = check.MaintenanceUntil
	}
	if groupSettings != nil && groupSettings.MaintenanceUntil != nil && groupSettings.MaintenanceUntil.After(now) {
		es.InMaintenance = true
		if es.MaintenanceUntil == nil || groupSettings.MaintenanceUntil.After(*es.MaintenanceUntil) {
			es.MaintenanceUntil = groupSettings.MaintenanceUntil
		}
		if es.MaintenanceReason == "" {
			es.MaintenanceReason = groupSettings.MaintenanceReason
		}
	}
	if projectSettings != nil && projectSettings.MaintenanceUntil != nil && projectSettings.MaintenanceUntil.After(now) {
		es.InMaintenance = true
		if es.MaintenanceUntil == nil || projectSettings.MaintenanceUntil.After(*es.MaintenanceUntil) {
			es.MaintenanceUntil = projectSettings.MaintenanceUntil
		}
		if es.MaintenanceReason == "" {
			es.MaintenanceReason = projectSettings.MaintenanceReason
		}
	}

	return es
}
