package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"checker/internal/config"
	"checker/internal/models"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
}

func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	dsn := cfg.DB.DatabaseURL
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
			cfg.DB.Username, cfg.DB.Password, cfg.DB.Host, cfg.DB.Database)
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Run migrations
	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		logrus.Warnf("Could not create migration instance: %v", err)
	} else {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			// If dirty, force to previous version and retry
			if version, dirty, vErr := m.Version(); vErr == nil && dirty {
				logrus.Warnf("Migration dirty at version %d, forcing to %d and retrying", version, version-1)
				if fErr := m.Force(int(version) - 1); fErr == nil {
					if rErr := m.Up(); rErr != nil && rErr != migrate.ErrNoChange {
						logrus.Errorf("Migration retry failed: %v", rErr)
					}
				} else {
					logrus.Errorf("Migration force failed: %v", fErr)
				}
			} else {
				logrus.Errorf("Migration failed: %v", err)
			}
		}
	}

	return &PostgresDB{Pool: pool}, nil
}

func (db *PostgresDB) Close() {
	db.Pool.Close()
}

// checkDefColumns is the shared SELECT column list for check_definitions queries.
const checkDefColumns = `uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config, severity, alert_channels, re_alert_interval, retry_count, retry_interval, maintenance_until, escalation_policy_name`

// scanCheckDef scans a row into a CheckDefinition, handling config unmarshaling
// and alert_channels JSON parsing.
func scanCheckDef(scanner interface{ Scan(dest ...interface{}) error }) (models.CheckDefinition, error) {
	var c models.CheckDefinition
	var configJSON, actorConfigJSON []byte
	var alertChannelsJSON *string
	var severity, reAlertInterval *string
	var retryCount *int
	var retryInterval *string
	var maintenanceUntil *time.Time
	var escalationPolicyName *string

	err := scanner.Scan(
		&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &c.Description, &c.Enabled,
		&c.CreatedAt, &c.UpdatedAt, &c.LastRun, &c.IsHealthy, &c.LastMessage, &c.LastAlertSent,
		&c.Duration, &c.ActorType, &c.AlertType, &c.AlertDestination, &configJSON, &actorConfigJSON,
		&severity, &alertChannelsJSON, &reAlertInterval, &retryCount, &retryInterval, &maintenanceUntil,
		&escalationPolicyName,
	)
	if err != nil {
		return models.CheckDefinition{}, err
	}

	// Set severity with default
	if severity != nil && *severity != "" {
		c.Severity = *severity
	} else {
		c.Severity = "critical"
	}

	// Parse alert_channels JSON array
	if alertChannelsJSON != nil && *alertChannelsJSON != "" {
		if err := json.Unmarshal([]byte(*alertChannelsJSON), &c.AlertChannels); err != nil {
			logrus.Warnf("Failed to parse alert_channels for %s: %v", c.UUID, err)
		}
	}

	// Set re_alert_interval
	if reAlertInterval != nil {
		c.ReAlertInterval = *reAlertInterval
	}

	// Set retry configuration
	if retryCount != nil {
		c.RetryCount = *retryCount
	}
	if retryInterval != nil {
		c.RetryInterval = *retryInterval
	}

	// Set maintenance_until
	c.MaintenanceUntil = maintenanceUntil

	// Set escalation_policy_name
	if escalationPolicyName != nil {
		c.EscalationPolicyName = *escalationPolicyName
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

// marshalAlertChannels converts the AlertChannels slice to a JSON string for storage.
func marshalAlertChannels(channels []string) *string {
	if len(channels) == 0 {
		return nil
	}
	data, err := json.Marshal(channels)
	if err != nil {
		return nil
	}
	s := string(data)
	return &s
}

func (db *PostgresDB) CountCheckDefinitions(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM check_definitions").Scan(&count)
	return count, err
}

func (db *PostgresDB) GetAllCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	rows, err := db.Pool.Query(ctx, "SELECT "+checkDefColumns+" FROM check_definitions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		c, err := scanCheckDef(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, nil
}

func (db *PostgresDB) GetEnabledCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	rows, err := db.Pool.Query(ctx, "SELECT "+checkDefColumns+" FROM check_definitions WHERE enabled=$1", true)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		c, err := scanCheckDef(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, nil
}

func (db *PostgresDB) GetCheckDefinitionByUUID(ctx context.Context, uuid string) (models.CheckDefinition, error) {
	row := db.Pool.QueryRow(ctx, "SELECT "+checkDefColumns+" FROM check_definitions WHERE uuid=$1", uuid)
	return scanCheckDef(row)
}

func (db *PostgresDB) CreateCheckDefinition(ctx context.Context, def models.CheckDefinition) (string, error) {
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)
	alertChannelsJSON := marshalAlertChannels(def.AlertChannels)

	if def.Severity == "" {
		def.Severity = "critical"
	}

	// insert
	_, err := db.Pool.Exec(ctx, `INSERT INTO check_definitions
    (uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config, severity, alert_channels, re_alert_interval, retry_count, retry_interval, maintenance_until, escalation_policy_name)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)`,
		def.UUID, def.Name, def.Project, def.GroupName, def.Type, def.Description, def.Enabled, def.CreatedAt, def.UpdatedAt, def.LastRun, def.IsHealthy, def.LastMessage, def.LastAlertSent, def.Duration, def.ActorType, def.AlertType, def.AlertDestination, configJSON, actorConfigJSON, def.Severity, alertChannelsJSON, def.ReAlertInterval, def.RetryCount, def.RetryInterval, def.MaintenanceUntil, nilIfEmpty(def.EscalationPolicyName))

	if err != nil {
		return "", err
	}
	return def.UUID, nil
}

func (db *PostgresDB) UpdateCheckDefinition(ctx context.Context, def models.CheckDefinition) error {
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)
	alertChannelsJSON := marshalAlertChannels(def.AlertChannels)

	if def.Severity == "" {
		def.Severity = "critical"
	}

	// update
	cmdTag, err := db.Pool.Exec(ctx, `UPDATE check_definitions SET
    name=$2, project=$3, group_name=$4, type=$5, description=$6, enabled=$7, updated_at=$8, last_run=$9, is_healthy=$10, last_message=$11, last_alert_sent=$12, duration=$13, actor_type=$14, alert_type=$15, alert_destination=$16, config=$17, actor_config=$18, severity=$19, alert_channels=$20, re_alert_interval=$21, retry_count=$22, retry_interval=$23, maintenance_until=$24, escalation_policy_name=$25
    WHERE uuid=$1`,
		def.UUID, def.Name, def.Project, def.GroupName, def.Type, def.Description, def.Enabled, time.Now(), def.LastRun, def.IsHealthy, def.LastMessage, def.LastAlertSent, def.Duration, def.ActorType, def.AlertType, def.AlertDestination, configJSON, actorConfigJSON, def.Severity, alertChannelsJSON, def.ReAlertInterval, def.RetryCount, def.RetryInterval, def.MaintenanceUntil, nilIfEmpty(def.EscalationPolicyName))

	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) DeleteCheckDefinition(ctx context.Context, uuid string) error {
	cmdTag, err := db.Pool.Exec(ctx, "DELETE FROM check_definitions WHERE uuid=$1", uuid)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) ToggleCheckDefinition(ctx context.Context, uuid string, enabled bool) error {
	cmdTag, err := db.Pool.Exec(ctx, "UPDATE check_definitions SET enabled=$2, updated_at=$3 WHERE uuid=$1", uuid, enabled, time.Now())
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) UpdateCheckStatus(ctx context.Context, status models.CheckStatus) error {
	cmdTag, err := db.Pool.Exec(ctx, `UPDATE check_definitions SET
    last_run=$2, is_healthy=$3, last_message=$4, last_alert_sent=$5, updated_at=$6
    WHERE uuid=$1`,
		status.UUID, status.LastRun, status.IsHealthy, status.Message, status.LastAlertSent, time.Now())

	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) BulkToggleCheckDefinitions(ctx context.Context, uuids []string, enabled bool) (int64, error) {
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE check_definitions SET enabled=$1, updated_at=$2 WHERE uuid = ANY($3)`,
		enabled, time.Now(), uuids)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

func (db *PostgresDB) BulkDeleteCheckDefinitions(ctx context.Context, uuids []string) (int64, error) {
	cmdTag, err := db.Pool.Exec(ctx,
		`DELETE FROM check_definitions WHERE uuid = ANY($1)`, uuids)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

func (db *PostgresDB) SetMaintenanceWindow(ctx context.Context, uuid string, until *time.Time) error {
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE check_definitions SET maintenance_until=$2, updated_at=$3 WHERE uuid=$1`,
		uuid, until, time.Now())
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) GetAllProjects(ctx context.Context) ([]string, error) {
	rows, err := db.Pool.Query(ctx, "SELECT DISTINCT project FROM check_definitions ORDER BY project")
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
	return projects, nil
}

func (db *PostgresDB) GetAllCheckTypes(ctx context.Context) ([]string, error) {
	rows, err := db.Pool.Query(ctx, "SELECT DISTINCT type FROM check_definitions ORDER BY type")
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
	return types, nil
}

func (db *PostgresDB) ConvertConfigToCheckDefinitions(ctx context.Context, cfg *config.Config) error {
	defaultTimeouts := db.GetAllDefaultTimeouts()

	for projectName, project := range cfg.Projects {
		for groupName, group := range project.HealthChecks {
			for checkName, check := range group.Checks {
				// Resolve duration hierarchically: check → group → project → defaults
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
				case "dns":
					checkDef.Config = &models.DNSCheckConfig{
						Host:    check.Host,
						Timeout: timeout,
					}
				case "ssl_cert":
					checkDef.Config = &models.SSLCertCheckConfig{
						Host:    check.Host,
						Port:    check.Port,
						Timeout: timeout,
					}
				case "ssh":
					checkDef.Config = &models.SSHCheckConfig{
						Host:    check.Host,
						Port:    check.Port,
						Timeout: timeout,
					}
				case "redis":
					checkDef.Config = &models.RedisCheckConfig{
						Host:    check.Host,
						Port:    check.Port,
						Timeout: timeout,
					}
				case "mongodb":
					checkDef.Config = &models.MongoDBCheckConfig{
						URI:     check.URL,
						Timeout: timeout,
					}
				case "domain_expiry":
					checkDef.Config = &models.DomainExpiryCheckConfig{
						Domain:  check.Host,
						Timeout: timeout,
					}
				case "smtp":
					checkDef.Config = &models.SMTPCheckConfig{
						Host:    check.Host,
						Port:    check.Port,
						Timeout: timeout,
					}
				case "grpc_health":
					checkDef.Config = &models.GRPCHealthCheckConfig{
						Host:    check.Host,
						Timeout: timeout,
					}
				case "websocket":
					checkDef.Config = &models.WebSocketCheckConfig{
						URL:     check.URL,
						Timeout: timeout,
					}
				case "mysql_query", "mysql_query_unixtime", "mysql_replication":
					// MySQL checks would need DB connection details from config
					logrus.Warnf("MySQL check type %s not fully implemented in config migration", check.Type)
					continue
				case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
					// PostgreSQL checks would need DB connection details from config
					logrus.Warnf("PostgreSQL check type %s not fully implemented in config migration", check.Type)
					continue
				default:
					logrus.Warnf("Unknown check type: %s", check.Type)
					continue
				}

				if checkDef.UUID == "" {
					checkDef.UUID = check.Name // fallback
				}

				// Try to create, log but continue on error
				if _, err := db.CreateCheckDefinition(ctx, checkDef); err != nil {
					logrus.Errorf("Failed to import check %s: %v", checkName, err)
				} else {
					logrus.Infof("Imported check %s (UUID: %s) from config", checkName, checkDef.UUID)
				}
			}
		}
	}
	return nil
}

func (db *PostgresDB) GetAllDefaultTimeouts() map[string]string {
	return map[string]string{
		"http":          "3s",
		"tcp":           "5s",
		"icmp":          "5s",
		"pgsql":         "10s",
		"mysql":         "10s",
		"dns":           "5s",
		"ssl_cert":      "10s",
		"ssh":           "5s",
		"redis":         "5s",
		"mongodb":       "10s",
		"domain_expiry": "10s",
		"smtp":          "10s",
		"grpc_health":   "5s",
		"websocket":     "5s",
		"default":       "5s",
	}
}

// Slack thread tracking

func (db *PostgresDB) CreateSlackThread(ctx context.Context, checkUUID, channelID, threadTs, parentTs string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO slack_alert_threads (check_uuid, channel_id, thread_ts, parent_ts) VALUES ($1, $2, $3, $4)`,
		checkUUID, channelID, threadTs, parentTs)
	return err
}

func (db *PostgresDB) GetUnresolvedThread(ctx context.Context, checkUUID string) (models.SlackAlertThread, error) {
	var t models.SlackAlertThread
	err := db.Pool.QueryRow(ctx,
		`SELECT id, check_uuid, channel_id, thread_ts, parent_ts, is_resolved, created_at, resolved_at
		 FROM slack_alert_threads WHERE check_uuid=$1 AND is_resolved=false ORDER BY created_at DESC LIMIT 1`, checkUUID).Scan(
		&t.ID, &t.CheckUUID, &t.ChannelID, &t.ThreadTs, &t.ParentTs, &t.IsResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.SlackAlertThread{}, err
	}
	return t, nil
}

func (db *PostgresDB) ResolveThread(ctx context.Context, checkUUID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE slack_alert_threads SET is_resolved=true, resolved_at=NOW() WHERE check_uuid=$1 AND is_resolved=false`,
		checkUUID)
	return err
}

func (db *PostgresDB) UpdateSlackThread(ctx context.Context, checkUUID, threadTs, channelID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE check_definitions SET slack_thread_ts=$2, slack_channel_id=$3, updated_at=NOW() WHERE uuid=$1`,
		checkUUID, threadTs, channelID)
	return err
}

// Alert silences

func (db *PostgresDB) CreateSilence(ctx context.Context, silence models.AlertSilence) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO alert_silences (scope, target, silenced_by, silenced_at, expires_at, reason, active) VALUES ($1, $2, $3, NOW(), $4, $5, $6)`,
		silence.Scope, silence.Target, silence.SilencedBy, silence.ExpiresAt, silence.Reason, silence.Active)
	return err
}

func (db *PostgresDB) DeactivateSilence(ctx context.Context, scope, target string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE alert_silences SET active = false WHERE scope = $1 AND target = $2 AND active = true`,
		scope, target)
	return err
}

func (db *PostgresDB) DeactivateSilenceByID(ctx context.Context, id int) error {
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE alert_silences SET active = false WHERE id = $1 AND active = true`, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("silence not found or already inactive")
	}
	return nil
}

func (db *PostgresDB) GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, scope, target, silenced_by, silenced_at, expires_at, reason
		FROM alert_silences
		WHERE active = true AND (expires_at IS NULL OR expires_at > NOW())`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var silences []models.AlertSilence
	for rows.Next() {
		var s models.AlertSilence
		if err := rows.Scan(&s.ID, &s.Scope, &s.Target, &s.SilencedBy, &s.SilencedAt, &s.ExpiresAt, &s.Reason); err != nil {
			return nil, err
		}
		s.Active = true
		silences = append(silences, s)
	}
	return silences, rows.Err()
}

func (db *PostgresDB) IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM alert_silences
			WHERE active = true
			AND (expires_at IS NULL OR expires_at > NOW())
			AND (
				(scope = 'check' AND target = $1)
				OR (scope = 'project' AND target = $2)
			)
		)`, checkUUID, project).Scan(&exists)
	return exists, err
}

func (db *PostgresDB) GetUnhealthyChecks(ctx context.Context) ([]models.CheckDefinition, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT uuid, name, project, group_name, type, is_healthy, last_message, last_run
		 FROM check_definitions WHERE is_healthy = false AND enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		var c models.CheckDefinition
		if err := rows.Scan(&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &c.IsHealthy, &c.LastMessage, &c.LastRun); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, nil
}

// Alert history

func (db *PostgresDB) CreateAlertEvent(ctx context.Context, event models.AlertEvent) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO alert_history (check_uuid, check_name, project, group_name, check_type, message, alert_type)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		event.CheckUUID, event.CheckName, event.Project, event.GroupName, event.CheckType, event.Message, event.AlertType)
	return err
}

func (db *PostgresDB) ResolveAlertEvent(ctx context.Context, checkUUID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE alert_history SET is_resolved = true, resolved_at = NOW()
		 WHERE check_uuid = $1 AND is_resolved = false`, checkUUID)
	return err
}

func (db *PostgresDB) GetAlertHistory(ctx context.Context, limit, offset int, filters models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	// Build WHERE clause dynamically
	where := ""
	args := []interface{}{}
	argIdx := 1

	if filters.Project != "" {
		where += fmt.Sprintf(" AND project = $%d", argIdx)
		args = append(args, filters.Project)
		argIdx++
	}
	if filters.CheckUUID != "" {
		where += fmt.Sprintf(" AND check_uuid = $%d", argIdx)
		args = append(args, filters.CheckUUID)
		argIdx++
	}
	if filters.IsResolved != nil {
		where += fmt.Sprintf(" AND is_resolved = $%d", argIdx)
		args = append(args, *filters.IsResolved)
		argIdx++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM alert_history WHERE 1=1" + where
	var total int
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := fmt.Sprintf(
		"SELECT id, check_uuid, check_name, project, group_name, check_type, message, alert_type, created_at, resolved_at, is_resolved FROM alert_history WHERE 1=1%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []models.AlertEvent
	for rows.Next() {
		var e models.AlertEvent
		if err := rows.Scan(&e.ID, &e.CheckUUID, &e.CheckName, &e.Project, &e.GroupName, &e.CheckType, &e.Message, &e.AlertType, &e.CreatedAt, &e.ResolvedAt, &e.IsResolved); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// nilIfEmpty returns nil if s is empty, otherwise a pointer to s.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func unmarshalConfig(checkType string, data []byte) (models.CheckConfig, error) {
	switch checkType {
	case "http":
		var conf models.HTTPCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "tcp":
		var conf models.TCPCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "icmp":
		var conf models.ICMPCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "passive":
		var conf models.PassiveCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "mysql_query", "mysql_query_unixtime", "mysql_replication":
		var conf models.MySQLCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
		var conf models.PostgreSQLCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "domain_expiry":
		var conf models.DomainExpiryCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "dns":
		var conf models.DNSCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "ssl_cert":
		var conf models.SSLCertCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "ssh":
		var conf models.SSHCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "redis":
		var conf models.RedisCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "mongodb":
		var conf models.MongoDBCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "smtp":
		var conf models.SMTPCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "grpc_health":
		var conf models.GRPCHealthCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	case "websocket":
		var conf models.WebSocketCheckConfig
		if err := json.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
		return &conf, nil
	}
	return nil, nil
}

// Escalation policies

func (db *PostgresDB) GetAllEscalationPolicies(ctx context.Context) ([]models.EscalationPolicy, error) {
	rows, err := db.Pool.Query(ctx, `SELECT id, name, steps, created_at FROM escalation_policies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []models.EscalationPolicy
	for rows.Next() {
		var p models.EscalationPolicy
		var stepsJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &stepsJSON, &p.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(stepsJSON, &p.Steps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal steps for policy %s: %w", p.Name, err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (db *PostgresDB) GetEscalationPolicyByName(ctx context.Context, name string) (models.EscalationPolicy, error) {
	var p models.EscalationPolicy
	var stepsJSON []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT id, name, steps, created_at FROM escalation_policies WHERE name=$1`, name).Scan(
		&p.ID, &p.Name, &stepsJSON, &p.CreatedAt)
	if err != nil {
		return models.EscalationPolicy{}, err
	}
	if err := json.Unmarshal(stepsJSON, &p.Steps); err != nil {
		return models.EscalationPolicy{}, fmt.Errorf("failed to unmarshal steps for policy %s: %w", p.Name, err)
	}
	return p, nil
}

func (db *PostgresDB) CreateEscalationPolicy(ctx context.Context, policy models.EscalationPolicy) error {
	stepsJSON, err := json.Marshal(policy.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO escalation_policies (name, steps) VALUES ($1, $2)`,
		policy.Name, stepsJSON)
	return err
}

func (db *PostgresDB) UpdateEscalationPolicy(ctx context.Context, policy models.EscalationPolicy) error {
	stepsJSON, err := json.Marshal(policy.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE escalation_policies SET steps=$2 WHERE name=$1`,
		policy.Name, stepsJSON)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("escalation policy not found")
	}
	return nil
}

func (db *PostgresDB) DeleteEscalationPolicy(ctx context.Context, name string) error {
	cmdTag, err := db.Pool.Exec(ctx, `DELETE FROM escalation_policies WHERE name=$1`, name)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("escalation policy not found")
	}
	return nil
}

// Escalation notifications

func (db *PostgresDB) GetEscalationNotifications(ctx context.Context, checkUUID, policyName string) ([]models.EscalationNotification, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, check_uuid, policy_name, step_index, notified_at
		 FROM escalation_notifications
		 WHERE check_uuid=$1 AND policy_name=$2
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

func (db *PostgresDB) CreateEscalationNotification(ctx context.Context, notification models.EscalationNotification) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO escalation_notifications (check_uuid, policy_name, step_index, notified_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (check_uuid, policy_name, step_index, notified_at) DO NOTHING`,
		notification.CheckUUID, notification.PolicyName, notification.StepIndex, notification.NotifiedAt)
	return err
}

func (db *PostgresDB) DeleteEscalationNotifications(ctx context.Context, checkUUID string) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM escalation_notifications WHERE check_uuid=$1`, checkUUID)
	return err
}

// Alert channels

func (db *PostgresDB) GetAllAlertChannels(ctx context.Context) ([]models.AlertChannel, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, name, type, config, created_at, updated_at
		 FROM alert_channels
		 ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.AlertChannel
	for rows.Next() {
		var ch models.AlertChannel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (db *PostgresDB) GetAlertChannelByName(ctx context.Context, name string) (models.AlertChannel, error) {
	var ch models.AlertChannel
	err := db.Pool.QueryRow(ctx,
		`SELECT id, name, type, config, created_at, updated_at
		 FROM alert_channels WHERE name=$1`, name).
		Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return ch, fmt.Errorf("alert channel not found: %w", err)
	}
	return ch, nil
}

func (db *PostgresDB) CreateAlertChannel(ctx context.Context, channel models.AlertChannel) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO alert_channels (name, type, config) VALUES ($1, $2, $3)`,
		channel.Name, channel.Type, channel.Config)
	return err
}

func (db *PostgresDB) UpdateAlertChannel(ctx context.Context, channel models.AlertChannel) error {
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE alert_channels SET type=$2, config=$3, updated_at=NOW() WHERE name=$1`,
		channel.Name, channel.Type, channel.Config)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("alert channel not found")
	}
	return nil
}

func (db *PostgresDB) DeleteAlertChannel(ctx context.Context, name string) error {
	cmdTag, err := db.Pool.Exec(ctx, `DELETE FROM alert_channels WHERE name=$1`, name)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("alert channel not found")
	}
	return nil
}

// Telegram thread tracking

func (db *PostgresDB) CreateTelegramThread(ctx context.Context, checkUUID, chatID string, messageID int) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO telegram_alert_threads (check_uuid, chat_id, message_id) VALUES ($1, $2, $3)`,
		checkUUID, chatID, messageID)
	return err
}

func (db *PostgresDB) GetUnresolvedTelegramThread(ctx context.Context, checkUUID string) (models.TelegramAlertThread, error) {
	var t models.TelegramAlertThread
	err := db.Pool.QueryRow(ctx,
		`SELECT id, check_uuid, chat_id, message_id, is_resolved, created_at, resolved_at
		 FROM telegram_alert_threads WHERE check_uuid=$1 AND is_resolved=false ORDER BY created_at DESC LIMIT 1`, checkUUID).Scan(
		&t.ID, &t.CheckUUID, &t.ChatID, &t.MessageID, &t.IsResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.TelegramAlertThread{}, err
	}
	return t, nil
}

func (db *PostgresDB) GetTelegramThreadByMessage(ctx context.Context, chatID string, messageID int) (models.TelegramAlertThread, error) {
	var t models.TelegramAlertThread
	err := db.Pool.QueryRow(ctx,
		`SELECT id, check_uuid, chat_id, message_id, is_resolved, created_at, resolved_at
		 FROM telegram_alert_threads WHERE chat_id=$1 AND message_id=$2 AND is_resolved=false`, chatID, messageID).Scan(
		&t.ID, &t.CheckUUID, &t.ChatID, &t.MessageID, &t.IsResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.TelegramAlertThread{}, err
	}
	return t, nil
}

func (db *PostgresDB) ResolveTelegramThread(ctx context.Context, checkUUID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE telegram_alert_threads SET is_resolved=true, resolved_at=NOW() WHERE check_uuid=$1 AND is_resolved=false`,
		checkUUID)
	return err
}

// MigrateLegacyAlertFields converts checks using the old ActorType="alert" + AlertType
// config to use AlertChannels. Idempotent — safe to run multiple times.
func (db *PostgresDB) MigrateLegacyAlertFields(ctx context.Context) (int, error) {
	// First, warn about checks that had AlertDestination set
	rows, err := db.Pool.Query(ctx,
		`SELECT uuid, name, alert_destination FROM check_definitions
		 WHERE actor_type = 'alert'
		   AND alert_type IS NOT NULL AND alert_type != ''
		   AND (alert_channels IS NULL OR alert_channels = '[]' OR alert_channels = '')
		   AND alert_destination IS NOT NULL AND alert_destination != ''`)
	if err != nil {
		return 0, fmt.Errorf("querying checks with alert_destination: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uuid, name, dest string
		if err := rows.Scan(&uuid, &name, &dest); err != nil {
			logrus.Warnf("Failed to scan alert_destination row: %v", err)
			continue
		}
		logrus.Warnf("Check %q (%s) had AlertDestination=%q which is NOT auto-migrated. Please create a named alert channel in the Settings UI.", name, uuid, dest)
	}
	rows.Close()

	// Perform the migration
	tag, err := db.Pool.Exec(ctx, `
		UPDATE check_definitions
		SET alert_channels = json_build_array(alert_type),
		    actor_type = NULL,
		    alert_type = NULL,
		    alert_destination = NULL
		WHERE actor_type = 'alert'
		  AND alert_type IS NOT NULL AND alert_type != ''
		  AND (alert_channels IS NULL OR alert_channels = '[]' OR alert_channels = '')`)
	if err != nil {
		return 0, fmt.Errorf("migrating legacy alert fields: %w", err)
	}

	return int(tag.RowsAffected()), nil
}
