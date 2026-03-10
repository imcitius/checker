package db

import (
	"context"

	"my/checker/internal/models"
)

// Repository defines the interface for database operations related to
// Slack alert threading and alert silencing.
type Repository interface {
	// Slack Alert Threads

	// CreateSlackThread inserts a new Slack alert thread record.
	CreateSlackThread(ctx context.Context, checkUUID, channelID, threadTs, parentTs string) error

	// GetUnresolvedThread returns the latest unresolved thread for a check.
	GetUnresolvedThread(ctx context.Context, checkUUID string) (models.SlackAlertThread, error)

	// ResolveThread marks all unresolved threads for a check as resolved.
	ResolveThread(ctx context.Context, checkUUID string) error

	// Alert Silences

	// CreateSilence inserts a new alert silence and returns its ID.
	CreateSilence(ctx context.Context, silence models.AlertSilence) (int, error)

	// GetActiveSilences returns all active, non-expired silences.
	GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error)

	// IsCheckSilenced checks if any active silence matches the given check.
	IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error)

	// DeactivateSilence sets is_active=false for the given silence ID.
	DeactivateSilence(ctx context.Context, silenceID int) error

	// DeactivateAllSilences sets is_active=false for all silences.
	DeactivateAllSilences(ctx context.Context) error
}
