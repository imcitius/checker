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
}
