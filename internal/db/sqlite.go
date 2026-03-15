package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"

	"checker/internal/config"
	"checker/internal/models"
)

// SQLiteDB implements the Repository interface using SQLite.
type SQLiteDB struct {
	DB *sql.DB
}

// NewSQLiteDB opens (or creates) a SQLite database at the given path
// and ensures the schema is ready.
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	if dbPath == "" {
		dbPath = "checker.db"
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite doesn't handle concurrency the same way as Postgres;
	// limit to a single writer to avoid SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	sdb := &SQLiteDB{DB: db}
	if err := sdb.ensureSchema(); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	logrus.Infof("SQLite database opened at %s", dbPath)
	return sdb, nil
}

// Close closes the underlying database connection.
func (s *SQLiteDB) Close() {
	if err := s.DB.Close(); err != nil {
		logrus.Errorf("Error closing SQLite database: %v", err)
	}
}

// ensureSchema creates all tables if they do not already exist.
func (s *SQLiteDB) ensureSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS check_definitions (
		uuid               TEXT PRIMARY KEY,
		name               TEXT NOT NULL,
		project            TEXT NOT NULL DEFAULT '',
		group_name         TEXT NOT NULL DEFAULT '',
		type               TEXT NOT NULL,
		description        TEXT NOT NULL DEFAULT '',
		enabled            INTEGER NOT NULL DEFAULT 1,
		created_at         DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at         DATETIME NOT NULL DEFAULT (datetime('now')),
		last_run           DATETIME NOT NULL DEFAULT '1970-01-01T00:00:00Z',
		is_healthy         INTEGER NOT NULL DEFAULT 1,
		last_message       TEXT NOT NULL DEFAULT '',
		last_alert_sent    DATETIME NOT NULL DEFAULT '1970-01-01T00:00:00Z',
		duration           TEXT NOT NULL DEFAULT '30s',
		actor_type         TEXT NOT NULL DEFAULT '',
		alert_type         TEXT NOT NULL DEFAULT '',
		alert_destination  TEXT NOT NULL DEFAULT '',
		config             TEXT NOT NULL DEFAULT '{}',
		actor_config       TEXT NOT NULL DEFAULT '{}',
		slack_thread_ts    TEXT,
		slack_channel_id   TEXT,
		severity           TEXT NOT NULL DEFAULT 'critical',
		alert_channels     TEXT,
		re_alert_interval  TEXT,
		retry_count        INTEGER NOT NULL DEFAULT 0,
		retry_interval     TEXT,
		maintenance_until  DATETIME,
		escalation_policy_name TEXT
	);

	CREATE TABLE IF NOT EXISTS alert_silences (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		scope       TEXT NOT NULL,
		target      TEXT NOT NULL,
		silenced_by TEXT NOT NULL DEFAULT '',
		silenced_at DATETIME NOT NULL DEFAULT (datetime('now')),
		expires_at  DATETIME,
		reason      TEXT NOT NULL DEFAULT '',
		active      INTEGER NOT NULL DEFAULT 1
	);
	CREATE INDEX IF NOT EXISTS idx_alert_silences_active ON alert_silences(active);

	CREATE TABLE IF NOT EXISTS slack_alert_threads (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		check_uuid  TEXT NOT NULL,
		channel_id  TEXT NOT NULL,
		thread_ts   TEXT NOT NULL,
		parent_ts   TEXT NOT NULL DEFAULT '',
		is_resolved INTEGER NOT NULL DEFAULT 0,
		created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
		resolved_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_slack_threads_unresolved ON slack_alert_threads(check_uuid, is_resolved);

	CREATE TABLE IF NOT EXISTS alert_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		check_uuid  TEXT NOT NULL,
		check_name  TEXT NOT NULL DEFAULT '',
		project     TEXT NOT NULL DEFAULT '',
		group_name  TEXT NOT NULL DEFAULT '',
		check_type  TEXT NOT NULL DEFAULT '',
		message     TEXT NOT NULL DEFAULT '',
		alert_type  TEXT NOT NULL DEFAULT '',
		created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
		resolved_at DATETIME,
		is_resolved INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_alert_history_created ON alert_history(created_at);
	CREATE INDEX IF NOT EXISTS idx_alert_history_check ON alert_history(check_uuid);
	CREATE INDEX IF NOT EXISTS idx_alert_history_project ON alert_history(project);
	CREATE INDEX IF NOT EXISTS idx_alert_history_unresolved ON alert_history(is_resolved) WHERE is_resolved = 0;

	CREATE TABLE IF NOT EXISTS escalation_policies (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL UNIQUE,
		steps      TEXT NOT NULL DEFAULT '[]',
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS escalation_notifications (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		check_uuid  TEXT NOT NULL,
		policy_name TEXT NOT NULL,
		step_index  INTEGER NOT NULL,
		notified_at DATETIME NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_escalation_notif_unique
		ON escalation_notifications(check_uuid, policy_name, step_index, notified_at);

	CREATE TABLE IF NOT EXISTS alert_channels (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL UNIQUE,
		type       TEXT NOT NULL,
		config     TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);
	`

	_, err := s.DB.Exec(schema)
	return err
}

// ---------------------------------------------------------------------------
// Helper: scanCheckDefSQL scans a *sql.Row or *sql.Rows into a CheckDefinition.
// It mirrors the Postgres scanCheckDef but works with database/sql types.
// ---------------------------------------------------------------------------

func scanCheckDefSQL(scanner interface{ Scan(dest ...interface{}) error }) (models.CheckDefinition, error) {
	var c models.CheckDefinition
	var configJSON, actorConfigJSON []byte
	var alertChannelsJSON sql.NullString
	var severity, reAlertInterval sql.NullString
	var retryCount sql.NullInt64
	var retryInterval sql.NullString
	var maintenanceUntil sql.NullTime
	var escalationPolicyName sql.NullString
	var enabled, isHealthy sql.NullInt64

	err := scanner.Scan(
		&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &c.Description, &enabled,
		&c.CreatedAt, &c.UpdatedAt, &c.LastRun, &isHealthy, &c.LastMessage, &c.LastAlertSent,
		&c.Duration, &c.ActorType, &c.AlertType, &c.AlertDestination, &configJSON, &actorConfigJSON,
		&severity, &alertChannelsJSON, &reAlertInterval, &retryCount, &retryInterval, &maintenanceUntil,
		&escalationPolicyName,
	)
	if err != nil {
		return models.CheckDefinition{}, err
	}

	// Convert integer booleans
	c.Enabled = enabled.Valid && enabled.Int64 != 0
	c.IsHealthy = isHealthy.Valid && isHealthy.Int64 != 0

	// Set severity with default
	if severity.Valid && severity.String != "" {
		c.Severity = severity.String
	} else {
		c.Severity = "critical"
	}

	// Parse alert_channels JSON array
	if alertChannelsJSON.Valid && alertChannelsJSON.String != "" {
		if err := json.Unmarshal([]byte(alertChannelsJSON.String), &c.AlertChannels); err != nil {
			logrus.Warnf("Failed to parse alert_channels for %s: %v", c.UUID, err)
		}
	}

	// Set re_alert_interval
	if reAlertInterval.Valid {
		c.ReAlertInterval = reAlertInterval.String
	}

	// Set retry configuration
	if retryCount.Valid {
		c.RetryCount = int(retryCount.Int64)
	}
	if retryInterval.Valid {
		c.RetryInterval = retryInterval.String
	}

	// Set maintenance_until
	if maintenanceUntil.Valid {
		t := maintenanceUntil.Time
		c.MaintenanceUntil = &t
	}

	// Set escalation_policy_name
	if escalationPolicyName.Valid {
		c.EscalationPolicyName = escalationPolicyName.String
	}

	// Unmarshal Polymorphic Config
	if len(configJSON) > 0 {
		conf, err := unmarshalConfig(c.Type, configJSON)
		if err != nil {
			logrus.Errorf("Failed to unmarshal config for %s: %v", c.UUID, err)
		} else {
			c.Config = conf
		}
	}

	if len(actorConfigJSON) > 0 && c.ActorType == "webhook" {
		var webhookConf models.WebhookConfig
		if err := json.Unmarshal(actorConfigJSON, &webhookConf); err == nil {
			c.ActorConfig = &webhookConf
		}
	}

	return c, nil
}

// sqliteCheckDefColumns is the column list for check_definitions queries.
const sqliteCheckDefColumns = `uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config, severity, alert_channels, re_alert_interval, retry_count, retry_interval, maintenance_until, escalation_policy_name`

// placeholders generates "?, ?, ?" for n parameters.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// ---------------------------------------------------------------------------
// Check Definition CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteDB) CountCheckDefinitions(ctx context.Context) (int, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM check_definitions").Scan(&count)
	return count, err
}

func (s *SQLiteDB) GetAllCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+sqliteCheckDefColumns+" FROM check_definitions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		c, err := scanCheckDefSQL(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (s *SQLiteDB) GetEnabledCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+sqliteCheckDefColumns+" FROM check_definitions WHERE enabled=1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		c, err := scanCheckDefSQL(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (s *SQLiteDB) GetCheckDefinitionByUUID(ctx context.Context, uuid string) (models.CheckDefinition, error) {
	row := s.DB.QueryRowContext(ctx, "SELECT "+sqliteCheckDefColumns+" FROM check_definitions WHERE uuid=?", uuid)
	return scanCheckDefSQL(row)
}

func (s *SQLiteDB) CreateCheckDefinition(ctx context.Context, def models.CheckDefinition) (string, error) {
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)
	alertChannelsJSON := marshalAlertChannels(def.AlertChannels)

	if def.Severity == "" {
		def.Severity = "critical"
	}

	_, err := s.DB.ExecContext(ctx, `INSERT INTO check_definitions
    (uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config, severity, alert_channels, re_alert_interval, retry_count, retry_interval, maintenance_until, escalation_policy_name)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		def.UUID, def.Name, def.Project, def.GroupName, def.Type, def.Description, boolToInt(def.Enabled),
		def.CreatedAt.UTC().Format(time.RFC3339), def.UpdatedAt.UTC().Format(time.RFC3339),
		def.LastRun.UTC().Format(time.RFC3339), boolToInt(def.IsHealthy), def.LastMessage,
		def.LastAlertSent.UTC().Format(time.RFC3339), def.Duration, def.ActorType, def.AlertType,
		def.AlertDestination, string(configJSON), string(actorConfigJSON), def.Severity,
		alertChannelsJSON, nilIfEmpty(def.ReAlertInterval), def.RetryCount, nilIfEmpty(def.RetryInterval),
		nullableTime(def.MaintenanceUntil), nilIfEmpty(def.EscalationPolicyName))

	if err != nil {
		return "", err
	}
	return def.UUID, nil
}

func (s *SQLiteDB) UpdateCheckDefinition(ctx context.Context, def models.CheckDefinition) error {
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)
	alertChannelsJSON := marshalAlertChannels(def.AlertChannels)

	if def.Severity == "" {
		def.Severity = "critical"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.DB.ExecContext(ctx, `UPDATE check_definitions SET
    name=?, project=?, group_name=?, type=?, description=?, enabled=?, updated_at=?, last_run=?, is_healthy=?, last_message=?, last_alert_sent=?, duration=?, actor_type=?, alert_type=?, alert_destination=?, config=?, actor_config=?, severity=?, alert_channels=?, re_alert_interval=?, retry_count=?, retry_interval=?, maintenance_until=?, escalation_policy_name=?
    WHERE uuid=?`,
		def.Name, def.Project, def.GroupName, def.Type, def.Description, boolToInt(def.Enabled),
		now, def.LastRun.UTC().Format(time.RFC3339), boolToInt(def.IsHealthy), def.LastMessage,
		def.LastAlertSent.UTC().Format(time.RFC3339), def.Duration, def.ActorType, def.AlertType,
		def.AlertDestination, string(configJSON), string(actorConfigJSON), def.Severity,
		alertChannelsJSON, nilIfEmpty(def.ReAlertInterval), def.RetryCount, nilIfEmpty(def.RetryInterval),
		nullableTime(def.MaintenanceUntil), nilIfEmpty(def.EscalationPolicyName),
		def.UUID)

	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (s *SQLiteDB) DeleteCheckDefinition(ctx context.Context, uuid string) error {
	result, err := s.DB.ExecContext(ctx, "DELETE FROM check_definitions WHERE uuid=?", uuid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (s *SQLiteDB) ToggleCheckDefinition(ctx context.Context, uuid string, enabled bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.DB.ExecContext(ctx, "UPDATE check_definitions SET enabled=?, updated_at=? WHERE uuid=?",
		boolToInt(enabled), now, uuid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (s *SQLiteDB) UpdateCheckStatus(ctx context.Context, status models.CheckStatus) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.DB.ExecContext(ctx, `UPDATE check_definitions SET
    last_run=?, is_healthy=?, last_message=?, last_alert_sent=?, updated_at=?
    WHERE uuid=?`,
		status.LastRun.UTC().Format(time.RFC3339), boolToInt(status.IsHealthy), status.Message,
		status.LastAlertSent.UTC().Format(time.RFC3339), now, status.UUID)

	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (s *SQLiteDB) BulkToggleCheckDefinitions(ctx context.Context, uuids []string, enabled bool) (int64, error) {
	if len(uuids) == 0 {
		return 0, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	args := make([]interface{}, 0, len(uuids)+2)
	args = append(args, boolToInt(enabled), now)
	for _, u := range uuids {
		args = append(args, u)
	}
	query := fmt.Sprintf("UPDATE check_definitions SET enabled=?, updated_at=? WHERE uuid IN (%s)", placeholders(len(uuids)))
	result, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *SQLiteDB) BulkDeleteCheckDefinitions(ctx context.Context, uuids []string) (int64, error) {
	if len(uuids) == 0 {
		return 0, nil
	}
	args := make([]interface{}, len(uuids))
	for i, u := range uuids {
		args[i] = u
	}
	query := fmt.Sprintf("DELETE FROM check_definitions WHERE uuid IN (%s)", placeholders(len(uuids)))
	result, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *SQLiteDB) SetMaintenanceWindow(ctx context.Context, uuid string, until *time.Time) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.DB.ExecContext(ctx,
		`UPDATE check_definitions SET maintenance_until=?, updated_at=? WHERE uuid=?`,
		nullableTime(until), now, uuid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (s *SQLiteDB) GetAllProjects(ctx context.Context) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT DISTINCT project FROM check_definitions ORDER BY project")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *SQLiteDB) GetAllCheckTypes(ctx context.Context) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT DISTINCT type FROM check_definitions ORDER BY type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

func (s *SQLiteDB) ConvertConfigToCheckDefinitions(ctx context.Context, cfg *config.Config) error {
	defaultTimeouts := s.GetAllDefaultTimeouts()

	for projectName, project := range cfg.Projects {
		for groupName, group := range project.HealthChecks {
			for checkName, check := range group.Checks {
				// Resolve duration hierarchically: check -> group -> project -> defaults
				duration := check.Parameters.Duration
				if duration == 0 {
					duration = group.Parameters.Duration
				}
				if duration == 0 {
					duration = project.Parameters.Duration
				}
				if duration == 0 {
					duration = cfg.Defaults.Duration
				}

				// Resolve timeout
				timeout := check.Timeout
				if timeout == "" {
					timeout = check.Parameters.Timeout
				}
				if timeout == "" {
					if defaultTO, ok := defaultTimeouts[check.Type]; ok {
						timeout = defaultTO
					} else {
						timeout = defaultTimeouts["default"]
					}
				}

				// Build check definition
				checkDef := models.CheckDefinition{
					UUID:        check.UUID,
					Name:        checkName,
					Project:     projectName,
					GroupName:   groupName,
					Type:        check.Type,
					Description: check.Description,
					Enabled:     true,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Duration:    duration.String(),
					ActorType:   check.ActorType,
					AlertType:   check.AlertType,
				}

				// Map config to appropriate check config type
				switch check.Type {
				case "http":
					checkDef.Config = &models.HTTPCheckConfig{
						URL:                 check.URL,
						Timeout:             timeout,
						Code:                check.Code,
						Headers:             check.Headers,
						SkipCheckSSL:        check.SkipCheckSSL,
						StopFollowRedirects: check.StopFollowRedirects,
					}
				case "tcp":
					checkDef.Config = &models.TCPCheckConfig{
						Host:    check.Host,
						Port:    check.Port,
						Timeout: timeout,
					}
				case "icmp":
					checkDef.Config = &models.ICMPCheckConfig{
						Host:    check.Host,
						Timeout: timeout,
					}
				case "passive":
					checkDef.Config = &models.PassiveCheckConfig{}
				case "domain_expiry":
					checkDef.Config = &models.DomainExpiryCheckConfig{
						Domain:  check.Host,
						Timeout: timeout,
					}
				case "mysql_query", "mysql_query_unixtime", "mysql_replication":
					logrus.Warnf("MySQL check type %s not fully implemented in config migration", check.Type)
					continue
				case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
					logrus.Warnf("PostgreSQL check type %s not fully implemented in config migration", check.Type)
					continue
				default:
					logrus.Warnf("Unknown check type: %s", check.Type)
					continue
				}

				if checkDef.UUID == "" {
					checkDef.UUID = check.Name
				}

				if _, err := s.CreateCheckDefinition(ctx, checkDef); err != nil {
					logrus.Errorf("Failed to import check %s: %v", checkName, err)
				} else {
					logrus.Infof("Imported check %s (UUID: %s) from config", checkName, checkDef.UUID)
				}
			}
		}
	}
	return nil
}

func (s *SQLiteDB) GetAllDefaultTimeouts() map[string]string {
	return map[string]string{
		"http":    "3s",
		"tcp":     "5s",
		"icmp":    "5s",
		"pgsql":   "10s",
		"mysql":   "10s",
		"default": "5s",
	}
}

// ---------------------------------------------------------------------------
// Slack thread tracking
// ---------------------------------------------------------------------------

func (s *SQLiteDB) CreateSlackThread(ctx context.Context, checkUUID, channelID, threadTs, parentTs string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO slack_alert_threads (check_uuid, channel_id, thread_ts, parent_ts) VALUES (?, ?, ?, ?)`,
		checkUUID, channelID, threadTs, parentTs)
	return err
}

func (s *SQLiteDB) GetUnresolvedThread(ctx context.Context, checkUUID string) (models.SlackAlertThread, error) {
	var t models.SlackAlertThread
	var isResolved sql.NullInt64
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, check_uuid, channel_id, thread_ts, parent_ts, is_resolved, created_at, resolved_at
		 FROM slack_alert_threads WHERE check_uuid=? AND is_resolved=0 ORDER BY created_at DESC LIMIT 1`, checkUUID).Scan(
		&t.ID, &t.CheckUUID, &t.ChannelID, &t.ThreadTs, &t.ParentTs, &isResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.SlackAlertThread{}, err
	}
	t.IsResolved = isResolved.Valid && isResolved.Int64 != 0
	return t, nil
}

func (s *SQLiteDB) ResolveThread(ctx context.Context, checkUUID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx,
		`UPDATE slack_alert_threads SET is_resolved=1, resolved_at=? WHERE check_uuid=? AND is_resolved=0`,
		now, checkUUID)
	return err
}

func (s *SQLiteDB) UpdateSlackThread(ctx context.Context, checkUUID, threadTs, channelID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx,
		`UPDATE check_definitions SET slack_thread_ts=?, slack_channel_id=?, updated_at=? WHERE uuid=?`,
		threadTs, channelID, now, checkUUID)
	return err
}

// ---------------------------------------------------------------------------
// Alert silences
// ---------------------------------------------------------------------------

func (s *SQLiteDB) CreateSilence(ctx context.Context, silence models.AlertSilence) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO alert_silences (scope, target, silenced_by, silenced_at, expires_at, reason, active) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		silence.Scope, silence.Target, silence.SilencedBy, now, nullableTime(silence.ExpiresAt), silence.Reason, boolToInt(silence.Active))
	return err
}

func (s *SQLiteDB) DeactivateSilence(ctx context.Context, scope, target string) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE alert_silences SET active = 0 WHERE scope = ? AND target = ? AND active = 1`,
		scope, target)
	return err
}

func (s *SQLiteDB) DeactivateSilenceByID(ctx context.Context, id int) error {
	result, err := s.DB.ExecContext(ctx,
		`UPDATE alert_silences SET active = 0 WHERE id = ? AND active = 1`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("silence not found or already inactive")
	}
	return nil
}

func (s *SQLiteDB) GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, scope, target, silenced_by, silenced_at, expires_at, reason
		FROM alert_silences
		WHERE active = 1 AND (expires_at IS NULL OR expires_at > ?)`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var silences []models.AlertSilence
	for rows.Next() {
		var si models.AlertSilence
		if err := rows.Scan(&si.ID, &si.Scope, &si.Target, &si.SilencedBy, &si.SilencedAt, &si.ExpiresAt, &si.Reason); err != nil {
			return nil, err
		}
		si.Active = true
		silences = append(silences, si)
	}
	return silences, rows.Err()
}

func (s *SQLiteDB) IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var count int
	err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM alert_silences
		WHERE active = 1
		AND (expires_at IS NULL OR expires_at > ?)
		AND (
			(scope = 'check' AND target = ?)
			OR (scope = 'project' AND target = ?)
		)`, now, checkUUID, project).Scan(&count)
	return count > 0, err
}

func (s *SQLiteDB) GetUnhealthyChecks(ctx context.Context) ([]models.CheckDefinition, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT uuid, name, project, group_name, type, is_healthy, last_message, last_run
		 FROM check_definitions WHERE is_healthy = 0 AND enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		var c models.CheckDefinition
		var isHealthy sql.NullInt64
		if err := rows.Scan(&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &isHealthy, &c.LastMessage, &c.LastRun); err != nil {
			return nil, err
		}
		c.IsHealthy = isHealthy.Valid && isHealthy.Int64 != 0
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

// ---------------------------------------------------------------------------
// Alert history
// ---------------------------------------------------------------------------

func (s *SQLiteDB) CreateAlertEvent(ctx context.Context, event models.AlertEvent) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO alert_history (check_uuid, check_name, project, group_name, check_type, message, alert_type)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.CheckUUID, event.CheckName, event.Project, event.GroupName, event.CheckType, event.Message, event.AlertType)
	return err
}

func (s *SQLiteDB) ResolveAlertEvent(ctx context.Context, checkUUID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx,
		`UPDATE alert_history SET is_resolved = 1, resolved_at = ?
		 WHERE check_uuid = ? AND is_resolved = 0`, now, checkUUID)
	return err
}

func (s *SQLiteDB) GetAlertHistory(ctx context.Context, limit, offset int, filters models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	// Build WHERE clause dynamically
	where := ""
	args := []interface{}{}

	if filters.Project != "" {
		where += " AND project = ?"
		args = append(args, filters.Project)
	}
	if filters.CheckUUID != "" {
		where += " AND check_uuid = ?"
		args = append(args, filters.CheckUUID)
	}
	if filters.IsResolved != nil {
		where += " AND is_resolved = ?"
		args = append(args, boolToInt(*filters.IsResolved))
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM alert_history WHERE 1=1" + where
	var total int
	if err := s.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := "SELECT id, check_uuid, check_name, project, group_name, check_type, message, alert_type, created_at, resolved_at, is_resolved FROM alert_history WHERE 1=1" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []models.AlertEvent
	for rows.Next() {
		var e models.AlertEvent
		var isResolved sql.NullInt64
		if err := rows.Scan(&e.ID, &e.CheckUUID, &e.CheckName, &e.Project, &e.GroupName, &e.CheckType, &e.Message, &e.AlertType, &e.CreatedAt, &e.ResolvedAt, &isResolved); err != nil {
			return nil, 0, err
		}
		e.IsResolved = isResolved.Valid && isResolved.Int64 != 0
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// ---------------------------------------------------------------------------
// Escalation policies
// ---------------------------------------------------------------------------

func (s *SQLiteDB) GetAllEscalationPolicies(ctx context.Context) ([]models.EscalationPolicy, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, steps, created_at FROM escalation_policies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []models.EscalationPolicy
	for rows.Next() {
		var p models.EscalationPolicy
		var stepsJSON string
		if err := rows.Scan(&p.ID, &p.Name, &stepsJSON, &p.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(stepsJSON), &p.Steps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal steps for policy %s: %w", p.Name, err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (s *SQLiteDB) GetEscalationPolicyByName(ctx context.Context, name string) (models.EscalationPolicy, error) {
	var p models.EscalationPolicy
	var stepsJSON string
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, name, steps, created_at FROM escalation_policies WHERE name=?`, name).Scan(
		&p.ID, &p.Name, &stepsJSON, &p.CreatedAt)
	if err != nil {
		return models.EscalationPolicy{}, err
	}
	if err := json.Unmarshal([]byte(stepsJSON), &p.Steps); err != nil {
		return models.EscalationPolicy{}, fmt.Errorf("failed to unmarshal steps for policy %s: %w", p.Name, err)
	}
	return p, nil
}

func (s *SQLiteDB) CreateEscalationPolicy(ctx context.Context, policy models.EscalationPolicy) error {
	stepsJSON, err := json.Marshal(policy.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO escalation_policies (name, steps) VALUES (?, ?)`,
		policy.Name, string(stepsJSON))
	return err
}

func (s *SQLiteDB) UpdateEscalationPolicy(ctx context.Context, policy models.EscalationPolicy) error {
	stepsJSON, err := json.Marshal(policy.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}
	result, err := s.DB.ExecContext(ctx,
		`UPDATE escalation_policies SET steps=? WHERE name=?`,
		string(stepsJSON), policy.Name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("escalation policy not found")
	}
	return nil
}

func (s *SQLiteDB) DeleteEscalationPolicy(ctx context.Context, name string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM escalation_policies WHERE name=?`, name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("escalation policy not found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Escalation notifications
// ---------------------------------------------------------------------------

func (s *SQLiteDB) GetEscalationNotifications(ctx context.Context, checkUUID, policyName string) ([]models.EscalationNotification, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, check_uuid, policy_name, step_index, notified_at
		 FROM escalation_notifications
		 WHERE check_uuid=? AND policy_name=?
		 ORDER BY step_index`, checkUUID, policyName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.EscalationNotification
	for rows.Next() {
		var n models.EscalationNotification
		if err := rows.Scan(&n.ID, &n.CheckUUID, &n.PolicyName, &n.StepIndex, &n.NotifiedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (s *SQLiteDB) CreateEscalationNotification(ctx context.Context, notification models.EscalationNotification) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT OR IGNORE INTO escalation_notifications (check_uuid, policy_name, step_index, notified_at)
		 VALUES (?, ?, ?, ?)`,
		notification.CheckUUID, notification.PolicyName, notification.StepIndex,
		notification.NotifiedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *SQLiteDB) DeleteEscalationNotifications(ctx context.Context, checkUUID string) error {
	_, err := s.DB.ExecContext(ctx,
		`DELETE FROM escalation_notifications WHERE check_uuid=?`, checkUUID)
	return err
}

// ---------------------------------------------------------------------------
// Alert channels
// ---------------------------------------------------------------------------

func (s *SQLiteDB) GetAllAlertChannels(ctx context.Context) ([]models.AlertChannel, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, name, type, config, created_at, updated_at FROM alert_channels ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.AlertChannel
	for rows.Next() {
		var ch models.AlertChannel
		var configStr string
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &configStr, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		ch.Config = json.RawMessage(configStr)
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (s *SQLiteDB) GetAlertChannelByName(ctx context.Context, name string) (models.AlertChannel, error) {
	var ch models.AlertChannel
	var configStr string
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, name, type, config, created_at, updated_at FROM alert_channels WHERE name=?`, name).
		Scan(&ch.ID, &ch.Name, &ch.Type, &configStr, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return ch, fmt.Errorf("alert channel not found: %w", err)
	}
	ch.Config = json.RawMessage(configStr)
	return ch, nil
}

func (s *SQLiteDB) CreateAlertChannel(ctx context.Context, channel models.AlertChannel) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO alert_channels (name, type, config) VALUES (?, ?, ?)`,
		channel.Name, channel.Type, string(channel.Config))
	return err
}

func (s *SQLiteDB) UpdateAlertChannel(ctx context.Context, channel models.AlertChannel) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.DB.ExecContext(ctx,
		`UPDATE alert_channels SET type=?, config=?, updated_at=? WHERE name=?`,
		channel.Type, string(channel.Config), now, channel.Name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("alert channel not found")
	}
	return nil
}

func (s *SQLiteDB) DeleteAlertChannel(ctx context.Context, name string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM alert_channels WHERE name=?`, name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("alert channel not found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// boolToInt converts a bool to an int for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// nullableTime formats a *time.Time as RFC3339 string or returns nil.
func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
