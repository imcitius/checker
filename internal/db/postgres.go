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

// UpdateSlackThread updates the Slack thread timestamp and channel ID
// on the check_definitions table for a given check UUID.
func (p *PostgresDB) UpdateSlackThread(ctx context.Context, uuid, threadTS, channelID string) error {
	query := `
		UPDATE check_definitions
		SET slack_thread_ts = $2, slack_channel_id = $3
		WHERE uuid = $1`
	_, err := p.db.ExecContext(ctx, query, uuid, threadTS, channelID)
	if err != nil {
		return fmt.Errorf("update slack thread: %w", err)
	}
	return nil
}

// GetSlackThread returns the Slack thread timestamp and channel ID
// stored on the check_definitions table for a given check UUID.
func (p *PostgresDB) GetSlackThread(ctx context.Context, uuid string) (threadTS, channelID string, err error) {
	query := `
		SELECT COALESCE(slack_thread_ts, ''), COALESCE(slack_channel_id, '')
		FROM check_definitions
		WHERE uuid = $1`
	err = p.db.QueryRowContext(ctx, query, uuid).Scan(&threadTS, &channelID)
	if err != nil {
		return "", "", fmt.Errorf("get slack thread: %w", err)
	}
	return threadTS, channelID, nil
}

// GetActiveSilence returns the first active, non-expired silence matching
// the given scope and target, or nil if none found.
func (p *PostgresDB) GetActiveSilence(ctx context.Context, scope, target string) (*models.AlertSilence, error) {
	query := `
		SELECT id, scope, target, silenced_by, silenced_at, expires_at, reason, active
		FROM alert_silences
		WHERE active = true
		AND scope = $1
		AND target = $2
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY silenced_at DESC
		LIMIT 1`

	var s models.AlertSilence
	err := p.db.QueryRowContext(ctx, query, scope, target).Scan(
		&s.ID, &s.Scope, &s.Target, &s.SilencedBy,
		&s.SilencedAt, &s.ExpiresAt, &s.Reason, &s.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active silence: %w", err)
	}
	return &s, nil
}

// CreateSilence inserts a new alert silence.
func (p *PostgresDB) CreateSilence(ctx context.Context, silence models.AlertSilence) error {
	query := `
		INSERT INTO alert_silences (scope, target, silenced_by, reason, expires_at, active)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := p.db.ExecContext(ctx, query,
		silence.Scope, silence.Target, silence.SilencedBy,
		silence.Reason, silence.ExpiresAt, silence.Active,
	)
	if err != nil {
		return fmt.Errorf("create silence: %w", err)
	}
	return nil
}

// DeactivateSilence sets active=false for the given silence ID.
func (p *PostgresDB) DeactivateSilence(ctx context.Context, id int) error {
	query := `UPDATE alert_silences SET active = false WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deactivate silence: %w", err)
	}
	return nil
}

// GetActiveSilences returns all active, non-expired silences.
func (p *PostgresDB) GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error) {
	query := `
		SELECT id, scope, target, silenced_by, reason, silenced_at, expires_at, active
		FROM alert_silences
		WHERE active = true
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY silenced_at DESC`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get active silences: %w", err)
	}
	defer rows.Close()

	var silences []models.AlertSilence
	for rows.Next() {
		var s models.AlertSilence
		if err := rows.Scan(
			&s.ID, &s.Scope, &s.Target, &s.SilencedBy,
			&s.Reason, &s.SilencedAt, &s.ExpiresAt, &s.Active,
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
			WHERE active = true
			AND (expires_at IS NULL OR expires_at > NOW())
			AND ((scope = 'check' AND target = $1) OR (scope = 'project' AND target = $2))
		)`

	var silenced bool
	err := p.db.QueryRowContext(ctx, query, checkUUID, project).Scan(&silenced)
	if err != nil {
		return false, fmt.Errorf("is check silenced: %w", err)
	}
	return silenced, nil
}

// DeactivateAllSilences sets active=false for all silences.
func (p *PostgresDB) DeactivateAllSilences(ctx context.Context) error {
	query := `UPDATE alert_silences SET active = false WHERE active = true`
	_, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("deactivate all silences: %w", err)
	}
	return nil
}
