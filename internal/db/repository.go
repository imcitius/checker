package db

import (
	"checker/internal/config"
	"checker/internal/models"
	"context"
)

// Repository defines the interface for database interactions
type Repository interface {
	GetAllCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error)
	GetEnabledCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error)
	GetCheckDefinitionByUUID(ctx context.Context, uuid string) (models.CheckDefinition, error)
	CreateCheckDefinition(ctx context.Context, def models.CheckDefinition) (string, error)
	UpdateCheckDefinition(ctx context.Context, def models.CheckDefinition) error
	DeleteCheckDefinition(ctx context.Context, uuid string) error
	ToggleCheckDefinition(ctx context.Context, uuid string, enabled bool) error
	UpdateCheckStatus(ctx context.Context, status models.CheckStatus) error
	GetAllProjects(ctx context.Context) ([]string, error)
	GetAllCheckTypes(ctx context.Context) ([]string, error)
	ConvertConfigToCheckDefinitions(ctx context.Context, config *config.Config) error
	GetAllDefaultTimeouts() map[string]string

	// Slack thread tracking
	CreateSlackThread(ctx context.Context, checkUUID, channelID, threadTs, parentTs string) error
	GetUnresolvedThread(ctx context.Context, checkUUID string) (models.SlackAlertThread, error)
	ResolveThread(ctx context.Context, checkUUID string) error
	UpdateSlackThread(ctx context.Context, checkUUID, threadTs, channelID string) error

	// Alert silences
	CreateSilence(ctx context.Context, silence models.AlertSilence) error
	IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error)
	DeactivateSilence(ctx context.Context, scope, target string) error
	DeactivateSilenceByID(ctx context.Context, id int) error
	GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error)
	GetUnhealthyChecks(ctx context.Context) ([]models.CheckDefinition, error)

	// Alert history
	CreateAlertEvent(ctx context.Context, event models.AlertEvent) error
	ResolveAlertEvent(ctx context.Context, checkUUID string) error
	GetAlertHistory(ctx context.Context, limit, offset int, filters models.AlertHistoryFilters) ([]models.AlertEvent, int, error)
}
