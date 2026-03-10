package db

import (
	"context"
	"database/sql"
	"fmt"

	"my/checker/internal/models"
)

// PostgresDB implements Repository using a PostgreSQL database.
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgresDB with the given sql.DB connection.
func NewPostgresDB(db *sql.DB) *PostgresDB {
	return &PostgresDB{db: db}
}

// CreateSlackThread inserts a new Slack alert thread record.
func (p *PostgresDB) CreateSlackThread(ctx context.Context, checkUUID, channelID, threadTs, parentTs string) error {
	query := `
		INSERT INTO slack_alert_threads (check_uuid, channel_id, thread_ts, parent_ts)
		VALUES ($1, $2, $3, $4)`
	_, err := p.db.ExecContext(ctx, query, checkUUID, channelID, threadTs, parentTs)
	if err != nil {
		return fmt.Errorf("create slack thread: %w", err)
	}
	return nil
}

// GetUnresolvedThread returns the latest unresolved thread for a check.
func (p *PostgresDB) GetUnresolvedThread(ctx context.Context, checkUUID string) (models.SlackAlertThread, error) {
	query := `
		SELECT id, check_uuid, channel_id, thread_ts, parent_ts, is_resolved, created_at, resolved_at
		FROM slack_alert_threads
		WHERE check_uuid = $1 AND is_resolved = false
		ORDER BY created_at DESC
		LIMIT 1`

	var t models.SlackAlertThread
	err := p.db.QueryRowContext(ctx, query, checkUUID).Scan(
		&t.ID, &t.CheckUUID, &t.ChannelID, &t.ThreadTs, &t.ParentTs,
		&t.IsResolved, &t.CreatedAt, &t.ResolvedAt,
	)
	if err != nil {
		return models.SlackAlertThread{}, fmt.Errorf("get unresolved thread: %w", err)
	}
	return t, nil
}

// ResolveThread marks all unresolved threads for a check as resolved.
func (p *PostgresDB) ResolveThread(ctx context.Context, checkUUID string) error {
	query := `
		UPDATE slack_alert_threads
		SET is_resolved = true, resolved_at = NOW()
		WHERE check_uuid = $1 AND is_resolved = false`
	_, err := p.db.ExecContext(ctx, query, checkUUID)
	if err != nil {
		return fmt.Errorf("resolve thread: %w", err)
	}
	return nil
}

// CreateSilence inserts a new alert silence and returns its ID.
func (p *PostgresDB) CreateSilence(ctx context.Context, silence models.AlertSilence) (int, error) {
	query := `
		INSERT INTO alert_silences (scope, target, created_by, reason, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int
	err := p.db.QueryRowContext(ctx, query,
		silence.Scope, silence.Target, silence.CreatedBy,
		silence.Reason, silence.ExpiresAt, silence.IsActive,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create silence: %w", err)
	}
	return id, nil
}

// GetActiveSilences returns all active, non-expired silences.
func (p *PostgresDB) GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error) {
	query := `
		SELECT id, scope, target, created_by, reason, created_at, expires_at, is_active
		FROM alert_silences
		WHERE is_active = true
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get active silences: %w", err)
	}
	defer rows.Close()

	var silences []models.AlertSilence
	for rows.Next() {
		var s models.AlertSilence
		if err := rows.Scan(
			&s.ID, &s.Scope, &s.Target, &s.CreatedBy,
			&s.Reason, &s.CreatedAt, &s.ExpiresAt, &s.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan silence: %w", err)
		}
		silences = append(silences, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate silences: %w", err)
	}
	return silences, nil
}

// IsCheckSilenced checks if any active silence matches the given check UUID or project.
func (p *PostgresDB) IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM alert_silences
			WHERE is_active = true
			AND (expires_at IS NULL OR expires_at > NOW())
			AND (scope = 'all' OR (scope = 'check' AND target = $1) OR (scope = 'project' AND target = $2))
		)`

	var silenced bool
	err := p.db.QueryRowContext(ctx, query, checkUUID, project).Scan(&silenced)
	if err != nil {
		return false, fmt.Errorf("is check silenced: %w", err)
	}
	return silenced, nil
}

// DeactivateSilence sets is_active=false for the given silence ID.
func (p *PostgresDB) DeactivateSilence(ctx context.Context, silenceID int) error {
	query := `UPDATE alert_silences SET is_active = false WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, silenceID)
	if err != nil {
		return fmt.Errorf("deactivate silence: %w", err)
	}
	return nil
}

// DeactivateAllSilences sets is_active=false for all silences.
func (p *PostgresDB) DeactivateAllSilences(ctx context.Context) error {
	query := `UPDATE alert_silences SET is_active = false WHERE is_active = true`
	_, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("deactivate all silences: %w", err)
	}
	return nil
}
