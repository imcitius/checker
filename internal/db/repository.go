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

	// UpdateSlackThread updates the Slack thread timestamp and channel ID
	// on the check_definitions table for a given check UUID.
	UpdateSlackThread(ctx context.Context, uuid, threadTS, channelID string) error

	// GetSlackThread returns the Slack thread timestamp and channel ID
	// stored on the check_definitions table for a given check UUID.
	GetSlackThread(ctx context.Context, uuid string) (threadTS, channelID string, err error)

	// Alert Silences

	// GetActiveSilence returns the first active, non-expired silence matching
	// the given scope and target, or nil if none found.
	GetActiveSilence(ctx context.Context, scope, target string) (*models.AlertSilence, error)

	// CreateSilence inserts a new alert silence.
	CreateSilence(ctx context.Context, silence models.AlertSilence) error

	// DeactivateSilence sets active=false for the given silence ID.
	DeactivateSilence(ctx context.Context, id int) error

	// GetActiveSilences returns all active, non-expired silences.
	GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error)

	// IsCheckSilenced checks if any active silence matches the given check.
	IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error)

	// DeactivateAllSilences sets active=false for all silences.
	DeactivateAllSilences(ctx context.Context) error
}
