// SPDX-License-Identifier: BUSL-1.1

package db

import (
	"github.com/imcitius/checker/pkg/config"
	"github.com/imcitius/checker/pkg/models"
	"context"
	"time"
)

// Repository defines the interface for database interactions
type Repository interface {
	Close()
	GetAllCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error)
	GetEnabledCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error)
	GetCheckDefinitionByUUID(ctx context.Context, uuid string) (models.CheckDefinition, error)
	CreateCheckDefinition(ctx context.Context, def models.CheckDefinition) (string, error)
	UpdateCheckDefinition(ctx context.Context, def models.CheckDefinition) error
	DeleteCheckDefinition(ctx context.Context, uuid string) error
	ToggleCheckDefinition(ctx context.Context, uuid string, enabled bool) error
	UpdateCheckStatus(ctx context.Context, status models.CheckStatus) error
	SetMaintenanceWindow(ctx context.Context, uuid string, until *time.Time) error
	BulkToggleCheckDefinitions(ctx context.Context, uuids []string, enabled bool) (int64, error)
	BulkDeleteCheckDefinitions(ctx context.Context, uuids []string) (int64, error)
	BulkUpdateAlertChannels(ctx context.Context, uuids []string, action string, channels []string) (int64, error)
	GetAllProjects(ctx context.Context) ([]string, error)
	GetAllCheckTypes(ctx context.Context) ([]string, error)
	ConvertConfigToCheckDefinitions(ctx context.Context, config *config.Config) error
	CountCheckDefinitions(ctx context.Context) (int, error)
	GetAllDefaultTimeouts() map[string]string

	// Slack thread tracking
	CreateSlackThread(ctx context.Context, checkUUID, channelID, threadTs, parentTs string) error
	GetUnresolvedThread(ctx context.Context, checkUUID string) (models.SlackAlertThread, error)
	ResolveThread(ctx context.Context, checkUUID string) error
	UpdateSlackThread(ctx context.Context, checkUUID, threadTs, channelID string) error

	// Telegram thread tracking
	CreateTelegramThread(ctx context.Context, checkUUID, chatID string, messageID int) error
	GetUnresolvedTelegramThread(ctx context.Context, checkUUID string) (models.TelegramAlertThread, error)
	GetTelegramThreadByMessage(ctx context.Context, chatID string, messageID int) (models.TelegramAlertThread, error)
	ResolveTelegramThread(ctx context.Context, checkUUID string) error

	// Discord thread tracking
	CreateDiscordThread(ctx context.Context, checkUUID, channelID, messageID, threadID string) error
	GetUnresolvedDiscordThread(ctx context.Context, checkUUID string) (models.DiscordAlertThread, error)
	ResolveDiscordThread(ctx context.Context, checkUUID string) error

	// Alert silences
	CreateSilence(ctx context.Context, silence models.AlertSilence) error
	IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error)
	IsChannelSilenced(ctx context.Context, checkUUID, project, channelName string) (bool, error)
	DeactivateSilence(ctx context.Context, scope, target string) error
	DeactivateSilenceByID(ctx context.Context, id int) error
	GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error)
	GetUnhealthyChecks(ctx context.Context) ([]models.CheckDefinition, error)

	// Alert history
	CreateAlertEvent(ctx context.Context, event models.AlertEvent) error
	ResolveAlertEvent(ctx context.Context, checkUUID string) error
	GetAlertHistory(ctx context.Context, limit, offset int, filters models.AlertHistoryFilters) ([]models.AlertEvent, int, error)

	// Escalation policies
	GetAllEscalationPolicies(ctx context.Context) ([]models.EscalationPolicy, error)
	GetEscalationPolicyByName(ctx context.Context, name string) (models.EscalationPolicy, error)
	CreateEscalationPolicy(ctx context.Context, policy models.EscalationPolicy) error
	UpdateEscalationPolicy(ctx context.Context, policy models.EscalationPolicy) error
	DeleteEscalationPolicy(ctx context.Context, name string) error

	// Escalation notifications
	GetEscalationNotifications(ctx context.Context, checkUUID, policyName string) ([]models.EscalationNotification, error)
	CreateEscalationNotification(ctx context.Context, notification models.EscalationNotification) error
	DeleteEscalationNotifications(ctx context.Context, checkUUID string) error

	// Migrations
	MigrateLegacyAlertFields(ctx context.Context) (int, error) // returns count of migrated checks

	// Alert channels
	GetAllAlertChannels(ctx context.Context) ([]models.AlertChannel, error)
	GetAlertChannelByName(ctx context.Context, name string) (models.AlertChannel, error)
	CreateAlertChannel(ctx context.Context, channel models.AlertChannel) error
	UpdateAlertChannel(ctx context.Context, channel models.AlertChannel) error
	DeleteAlertChannel(ctx context.Context, name string) error

	// Settings
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetCheckDefaults(ctx context.Context) (models.CheckDefaults, error)
	SaveCheckDefaults(ctx context.Context, defaults models.CheckDefaults) error

	// Project and group settings (hierarchical overrides)
	GetProjectSettings(ctx context.Context, project string) (*models.ProjectSettings, error)
	UpsertProjectSettings(ctx context.Context, settings models.ProjectSettings) error
	GetAllProjectSettings(ctx context.Context) ([]models.ProjectSettings, error)
	GetGroupSettings(ctx context.Context, project, groupName string) (*models.GroupSettings, error)
	UpsertGroupSettings(ctx context.Context, settings models.GroupSettings) error
	GetAllGroupSettings(ctx context.Context) ([]models.GroupSettings, error)

	// Multi-region check results
	GetLatestRegionResults(ctx context.Context, checkUUID string) ([]models.CheckResult, error)
	InsertCheckResult(ctx context.Context, result models.CheckResult) error
	GetUnevaluatedCycles(ctx context.Context, minRegions int, timeout time.Duration) ([]UnevaluatedCycle, error)
	ClaimCycleForEvaluation(ctx context.Context, checkUUID string, cycleKey time.Time) (bool, error)
	GetCycleResults(ctx context.Context, checkUUID string, cycleKey time.Time) ([]models.CheckResult, error)
	PurgeOldCheckResults(ctx context.Context, olderThan time.Duration) (int64, error)
}

// UnevaluatedCycle represents a check cycle that has enough regional results for consensus evaluation.
type UnevaluatedCycle struct {
	CheckUUID   string
	CycleKey    time.Time
	RegionCount int
}
