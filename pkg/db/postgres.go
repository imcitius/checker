// SPDX-License-Identifier: BUSL-1.1

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/migrations"
	"github.com/imcitius/checker/pkg/config"
	"github.com/imcitius/checker/pkg/models"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
	// TenantIDFunc, when non-nil, enables multi-tenant mode.
	// All queries include tenant_id filtering using the value returned by this function.
	// When nil (default), queries run without tenant scoping (single-tenant / standalone mode).
	TenantIDFunc func(ctx context.Context) string
}

// tenantScope holds tenant scoping state for query building.
type tenantScope struct {
	id string
}

// scope returns the tenant scope for the current context.
// Returns nil if not in multi-tenant mode.
func (db *PostgresDB) scope(ctx context.Context) *tenantScope {
	if db.TenantIDFunc == nil {
		return nil
	}
	return &tenantScope{id: db.TenantIDFunc(ctx)}
}

// where returns " WHERE tenant_id = $N" or empty string.
func (s *tenantScope) where(argIdx int) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf(" WHERE tenant_id = $%d", argIdx)
}

// and returns " AND tenant_id = $N" or empty string.
func (s *tenantScope) and(argIdx int) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf(" AND tenant_id = $%d", argIdx)
}

// insertCol returns ", tenant_id" or empty string.
func (s *tenantScope) insertCol() string {
	if s == nil {
		return ""
	}
	return ", tenant_id"
}

// insertPlaceholder returns ", $N" or empty string.
func (s *tenantScope) insertPlaceholder(argIdx int) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf(", $%d", argIdx)
}

// args returns the tenant ID as a single-element slice, or nil.
func (s *tenantScope) args() []interface{} {
	if s == nil {
		return nil
	}
	return []interface{}{s.id}
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
	source, err := iofs.New(migrations.PostgresFS, "postgres")
	if err != nil {
		logrus.Warnf("Could not create migration source: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
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

// NewPostgresDBFromPool creates a PostgresDB using an existing connection pool.
// No migrations are run. Use this when the caller manages the pool lifecycle
// (e.g. the SaaS layer sharing a pool with its own migration runner).
func NewPostgresDBFromPool(pool *pgxpool.Pool) *PostgresDB {
	return &PostgresDB{Pool: pool}
}

func (db *PostgresDB) Close() {
	db.Pool.Close()
}

// CheckDefColumns is the shared SELECT column list for check_definitions queries.
const CheckDefColumns = `uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, config, actor_config, severity, alert_channels, re_alert_interval, retry_count, retry_interval, maintenance_until, escalation_policy_name, run_mode, target_regions, edge_min_unhealthy`

// ScanCheckDef scans a row into a CheckDefinition, handling config unmarshaling
// and alert_channels JSON parsing. The row must contain columns matching CheckDefColumns.
func ScanCheckDef(scanner interface{ Scan(dest ...interface{}) error }) (models.CheckDefinition, error) {
	var c models.CheckDefinition
	var configJSON, actorConfigJSON []byte
	var alertChannelsJSON *string
	var severity, reAlertInterval *string
	var retryCount *int
	var retryInterval *string
	var maintenanceUntil *time.Time
	var escalationPolicyName *string
	var runMode *string
	var targetRegionsJSON []byte
	var edgeMinUnhealthy int

	// Nullable status columns — use pointers so NULL doesn't cause a scan error.
	var lastRun, lastAlertSent *time.Time
	var isHealthy *bool
	var lastMessage *string
	var duration, actorType *string

	err := scanner.Scan(
		&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &c.Description, &c.Enabled,
		&c.CreatedAt, &c.UpdatedAt, &lastRun, &isHealthy, &lastMessage, &lastAlertSent,
		&duration, &actorType, &configJSON, &actorConfigJSON,
		&severity, &alertChannelsJSON, &reAlertInterval, &retryCount, &retryInterval, &maintenanceUntil,
		&escalationPolicyName, &runMode, &targetRegionsJSON, &edgeMinUnhealthy,
	)
	if err != nil {
		return models.CheckDefinition{}, err
	}

	if lastRun != nil {
		c.LastRun = *lastRun
	}
	if isHealthy != nil {
		c.IsHealthy = *isHealthy
	}
	if lastMessage != nil {
		c.LastMessage = *lastMessage
	}
	if lastAlertSent != nil {
		c.LastAlertSent = *lastAlertSent
	}
	if duration != nil {
		c.Duration = *duration
	}
	if actorType != nil {
		c.ActorType = *actorType
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

	// Set run_mode
	if runMode != nil {
		c.RunMode = *runMode
	}

	// Parse target_regions JSON array
	if len(targetRegionsJSON) > 0 {
		if err := json.Unmarshal(targetRegionsJSON, &c.TargetRegions); err != nil {
			logrus.Warnf("Failed to parse target_regions for %s: %v", c.UUID, err)
		}
	}

	c.EdgeMinUnhealthy = edgeMinUnhealthy

	// Unmarshal Polymorphic Config
	if len(configJSON) > 0 {
		conf, err := UnmarshalConfig(c.Type, configJSON)
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

// MarshalAlertChannels converts the AlertChannels slice to a JSON string for storage.
func MarshalAlertChannels(channels []string) *string {
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

// MarshalStringSlice converts a string slice to a JSON string for storage.
func MarshalStringSlice(slice []string) *string {
	if len(slice) == 0 {
		return nil
	}
	data, err := json.Marshal(slice)
	if err != nil {
		return nil
	}
	s := string(data)
	return &s
}

func (db *PostgresDB) CountCheckDefinitions(ctx context.Context) (int, error) {
	s := db.scope(ctx)
	var count int
	query := "SELECT COUNT(*) FROM check_definitions" + s.where(1)
	err := db.Pool.QueryRow(ctx, query, s.args()...).Scan(&count)
	return count, err
}

func (db *PostgresDB) GetAllCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	s := db.scope(ctx)
	query := "SELECT " + CheckDefColumns + " FROM check_definitions" + s.where(1)
	rows, err := db.Pool.Query(ctx, query, s.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	checks := make([]models.CheckDefinition, 0)
	for rows.Next() {
		c, err := ScanCheckDef(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (db *PostgresDB) GetEnabledCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	s := db.scope(ctx)
	query := "SELECT " + CheckDefColumns + " FROM check_definitions WHERE enabled = $1" + s.and(2)
	args := []interface{}{true}
	args = append(args, s.args()...)
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		c, err := ScanCheckDef(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (db *PostgresDB) GetCheckDefinitionByUUID(ctx context.Context, uuid string) (models.CheckDefinition, error) {
	s := db.scope(ctx)
	query := "SELECT " + CheckDefColumns + " FROM check_definitions WHERE uuid = $1" + s.and(2)
	args := []interface{}{uuid}
	args = append(args, s.args()...)
	row := db.Pool.QueryRow(ctx, query, args...)
	return ScanCheckDef(row)
}

func (db *PostgresDB) CreateCheckDefinition(ctx context.Context, def models.CheckDefinition) (string, error) {
	s := db.scope(ctx)
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)
	alertChannelsJSON := MarshalAlertChannels(def.AlertChannels)

	if def.Severity == "" {
		def.Severity = "critical"
	}

	var targetRegionsJSON []byte
	if len(def.TargetRegions) > 0 {
		targetRegionsJSON, _ = json.Marshal(def.TargetRegions)
	}

	args := []interface{}{
		def.UUID, def.Name, def.Project, def.GroupName, def.Type, def.Description, def.Enabled,
		def.CreatedAt, def.UpdatedAt, def.LastRun, def.IsHealthy, def.LastMessage, def.LastAlertSent,
		def.Duration, def.ActorType, configJSON, actorConfigJSON, def.Severity, alertChannelsJSON,
		def.ReAlertInterval, def.RetryCount, def.RetryInterval, def.MaintenanceUntil,
		NilIfEmpty(def.EscalationPolicyName), NilIfEmpty(def.RunMode), targetRegionsJSON, def.EdgeMinUnhealthy,
	}
	args = append(args, s.args()...)

	query := fmt.Sprintf(`INSERT INTO check_definitions
    (uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, config, actor_config, severity, alert_channels, re_alert_interval, retry_count, retry_interval, maintenance_until, escalation_policy_name, run_mode, target_regions, edge_min_unhealthy%s)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27%s)`,
		s.insertCol(), s.insertPlaceholder(28))

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return "", err
	}
	return def.UUID, nil
}

func (db *PostgresDB) UpdateCheckDefinition(ctx context.Context, def models.CheckDefinition) error {
	s := db.scope(ctx)
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)
	alertChannelsJSON := MarshalAlertChannels(def.AlertChannels)

	if def.Severity == "" {
		def.Severity = "critical"
	}

	var targetRegionsJSON []byte
	if len(def.TargetRegions) > 0 {
		targetRegionsJSON, _ = json.Marshal(def.TargetRegions)
	}

	args := []interface{}{
		def.UUID, def.Name, def.Project, def.GroupName, def.Type, def.Description, def.Enabled,
		time.Now().UTC(), def.LastRun, def.IsHealthy, def.LastMessage, def.LastAlertSent,
		def.Duration, def.ActorType, configJSON, actorConfigJSON, def.Severity, alertChannelsJSON,
		def.ReAlertInterval, def.RetryCount, def.RetryInterval, def.MaintenanceUntil,
		NilIfEmpty(def.EscalationPolicyName), NilIfEmpty(def.RunMode), targetRegionsJSON, def.EdgeMinUnhealthy,
	}
	args = append(args, s.args()...)

	query := fmt.Sprintf(`UPDATE check_definitions SET
    name=$2, project=$3, group_name=$4, type=$5, description=$6, enabled=$7, updated_at=$8, last_run=$9, is_healthy=$10, last_message=$11, last_alert_sent=$12, duration=$13, actor_type=$14, config=$15, actor_config=$16, severity=$17, alert_channels=$18, re_alert_interval=$19, retry_count=$20, retry_interval=$21, maintenance_until=$22, escalation_policy_name=$23, run_mode=$24, target_regions=$25, edge_min_unhealthy=$26
    WHERE uuid=$1%s`, s.and(27))

	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) DeleteCheckDefinition(ctx context.Context, uuid string) error {
	s := db.scope(ctx)
	query := "DELETE FROM check_definitions WHERE uuid = $1" + s.and(2)
	args := []interface{}{uuid}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) ToggleCheckDefinition(ctx context.Context, uuid string, enabled bool) error {
	s := db.scope(ctx)
	now := time.Now().UTC()
	var query string
	if enabled {
		// When re-enabling, reset health status so it shows as "pending"
		// until the first probe result arrives, instead of displaying
		// stale failure state from before the check was disabled.
		query = "UPDATE check_definitions SET enabled = $2, is_healthy = true, last_message = '', updated_at = $3 WHERE uuid = $1" + s.and(4)
	} else {
		query = "UPDATE check_definitions SET enabled = $2, updated_at = $3 WHERE uuid = $1" + s.and(4)
	}
	args := []interface{}{uuid, enabled, now}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) UpdateCheckStatus(ctx context.Context, status models.CheckStatus) error {
	s := db.scope(ctx)
	// Do NOT update updated_at here. updated_at tracks configuration changes
	// (name, URL, channels, etc.), not health status updates.
	// configChangedSinceLastAlert() compares updated_at vs last_alert_sent
	// to detect config changes — advancing updated_at on every status update
	// causes false "config changed" alerts on every evaluation cycle.
	query := `UPDATE check_definitions SET
    last_run=$2, is_healthy=$3, last_message=$4, last_alert_sent=$5
    WHERE uuid=$1` + s.and(6)
	args := []interface{}{status.UUID, status.LastRun, status.IsHealthy, status.Message, status.LastAlertSent}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) BulkToggleCheckDefinitions(ctx context.Context, uuids []string, enabled bool) (int64, error) {
	s := db.scope(ctx)
	query := `UPDATE check_definitions SET enabled=$1, updated_at=$2 WHERE uuid = ANY($3)` + s.and(4)
	args := []interface{}{enabled, time.Now().UTC(), uuids}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

func (db *PostgresDB) BulkDeleteCheckDefinitions(ctx context.Context, uuids []string) (int64, error) {
	s := db.scope(ctx)
	query := `DELETE FROM check_definitions WHERE uuid = ANY($1)` + s.and(2)
	args := []interface{}{uuids}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

func (db *PostgresDB) BulkUpdateAlertChannels(ctx context.Context, uuids []string, action string, channels []string) (int64, error) {
	if len(uuids) == 0 {
		return 0, nil
	}

	s := db.scope(ctx)
	channelsJSON, err := json.Marshal(channels)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal channels: %w", err)
	}

	now := time.Now().UTC()
	tenantAnd := s.and(4)

	switch action {
	case "replace":
		query := `UPDATE check_definitions SET alert_channels=$1, updated_at=$2 WHERE uuid = ANY($3)` + tenantAnd
		args := []interface{}{string(channelsJSON), now, uuids}
		args = append(args, s.args()...)
		cmdTag, err := db.Pool.Exec(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return cmdTag.RowsAffected(), nil

	case "add":
		query := `UPDATE check_definitions SET
			alert_channels = (
				SELECT json_agg(DISTINCT ch)::text FROM (
					SELECT jsonb_array_elements_text(COALESCE(NULLIF(alert_channels,''),'[]')::jsonb) AS ch
					UNION
					SELECT jsonb_array_elements_text($1::jsonb) AS ch
				) sub
			),
			updated_at=$2
			WHERE uuid = ANY($3)` + tenantAnd
		args := []interface{}{string(channelsJSON), now, uuids}
		args = append(args, s.args()...)
		cmdTag, err := db.Pool.Exec(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return cmdTag.RowsAffected(), nil

	case "remove":
		query := `UPDATE check_definitions SET
			alert_channels = (
				SELECT COALESCE(json_agg(ch)::text, '[]') FROM (
					SELECT jsonb_array_elements_text(COALESCE(NULLIF(alert_channels,''),'[]')::jsonb) AS ch
					EXCEPT
					SELECT jsonb_array_elements_text($1::jsonb) AS ch
				) sub
			),
			updated_at=$2
			WHERE uuid = ANY($3)` + tenantAnd
		args := []interface{}{string(channelsJSON), now, uuids}
		args = append(args, s.args()...)
		cmdTag, err := db.Pool.Exec(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return cmdTag.RowsAffected(), nil

	default:
		return 0, fmt.Errorf("invalid action: %s (must be add, remove, or replace)", action)
	}
}

func (db *PostgresDB) SetMaintenanceWindow(ctx context.Context, uuid string, until *time.Time) error {
	s := db.scope(ctx)
	query := `UPDATE check_definitions SET maintenance_until=$2, updated_at=$3 WHERE uuid=$1` + s.and(4)
	args := []interface{}{uuid, until, time.Now().UTC()}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("check definition not found")
	}
	return nil
}

func (db *PostgresDB) GetAllProjects(ctx context.Context) ([]string, error) {
	s := db.scope(ctx)
	query := "SELECT DISTINCT project FROM check_definitions" + s.where(1) + " ORDER BY project"
	rows, err := db.Pool.Query(ctx, query, s.args()...)
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

func (db *PostgresDB) GetAllCheckTypes(ctx context.Context) ([]string, error) {
	s := db.scope(ctx)
	query := "SELECT DISTINCT type FROM check_definitions" + s.where(1) + " ORDER BY type"
	rows, err := db.Pool.Query(ctx, query, s.args()...)
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

// Settings

func (db *PostgresDB) GetSetting(ctx context.Context, key string) (string, error) {
	s := db.scope(ctx)
	query := `SELECT value FROM settings WHERE key = $1` + s.and(2)
	args := []interface{}{key}
	args = append(args, s.args()...)
	var value string
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (db *PostgresDB) SetSetting(ctx context.Context, key, value string) error {
	s := db.scope(ctx)
	if s != nil {
		// Multi-tenant: ON CONFLICT includes tenant_id
		_, err := db.Pool.Exec(ctx,
			`INSERT INTO settings (key, value, updated_at, tenant_id) VALUES ($1, $2, NOW(), $3)
			 ON CONFLICT (key, tenant_id) DO UPDATE SET value = $2, updated_at = NOW()`,
			key, value, s.id)
		return err
	}
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
		key, value)
	return err
}

func (db *PostgresDB) GetCheckDefaults(ctx context.Context) (models.CheckDefaults, error) {
	defaults := models.CheckDefaults{
		Timeouts: db.GetAllDefaultTimeouts(),
	}
	raw, err := db.GetSetting(ctx, "check_defaults")
	if err != nil {
		return defaults, nil // no saved defaults yet, return hardcoded
	}
	var saved models.CheckDefaults
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		return defaults, fmt.Errorf("unmarshal check_defaults: %w", err)
	}
	// Merge saved timeouts over hardcoded defaults
	if saved.Timeouts != nil {
		for k, v := range saved.Timeouts {
			defaults.Timeouts[k] = v
		}
	}
	defaults.RetryCount = saved.RetryCount
	defaults.RetryInterval = saved.RetryInterval
	defaults.CheckInterval = saved.CheckInterval
	defaults.ReAlertInterval = saved.ReAlertInterval
	defaults.Severity = saved.Severity
	defaults.AlertChannels = saved.AlertChannels
	defaults.EscalationPolicy = saved.EscalationPolicy
	return defaults, nil
}

func (db *PostgresDB) SaveCheckDefaults(ctx context.Context, defaults models.CheckDefaults) error {
	raw, err := json.Marshal(defaults)
	if err != nil {
		return fmt.Errorf("marshal check_defaults: %w", err)
	}
	return db.SetSetting(ctx, "check_defaults", string(raw))
}

// Slack thread tracking

func (db *PostgresDB) CreateSlackThread(ctx context.Context, checkUUID, channelID, threadTs, parentTs string) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(
		`INSERT INTO slack_alert_threads (check_uuid, channel_id, thread_ts, parent_ts%s) VALUES ($1, $2, $3, $4%s)`,
		s.insertCol(), s.insertPlaceholder(5))
	args := []interface{}{checkUUID, channelID, threadTs, parentTs}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) GetUnresolvedThread(ctx context.Context, checkUUID string) (models.SlackAlertThread, error) {
	s := db.scope(ctx)
	query := `SELECT id, check_uuid, channel_id, thread_ts, parent_ts, is_resolved, created_at, resolved_at
		 FROM slack_alert_threads WHERE check_uuid=$1 AND is_resolved=false` + s.and(2) + ` ORDER BY created_at DESC LIMIT 1`
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	var t models.SlackAlertThread
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.CheckUUID, &t.ChannelID, &t.ThreadTs, &t.ParentTs, &t.IsResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.SlackAlertThread{}, err
	}
	return t, nil
}

func (db *PostgresDB) ResolveThread(ctx context.Context, checkUUID string) error {
	s := db.scope(ctx)
	query := `UPDATE slack_alert_threads SET is_resolved=true, resolved_at=NOW() WHERE check_uuid=$1 AND is_resolved=false` + s.and(2)
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) UpdateSlackThread(ctx context.Context, checkUUID, threadTs, channelID string) error {
	s := db.scope(ctx)
	query := `UPDATE check_definitions SET slack_thread_ts=$2, slack_channel_id=$3, updated_at=NOW() WHERE uuid=$1` + s.and(4)
	args := []interface{}{checkUUID, threadTs, channelID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

// Alert silences

func (db *PostgresDB) CreateSilence(ctx context.Context, silence models.AlertSilence) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(
		`INSERT INTO alert_silences (scope, target, channel, silenced_by, silenced_at, expires_at, reason, active%s)
		 VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7%s)`,
		s.insertCol(), s.insertPlaceholder(8))
	args := []interface{}{silence.Scope, silence.Target, silence.Channel, silence.SilencedBy, silence.ExpiresAt, silence.Reason, silence.Active}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) DeactivateSilence(ctx context.Context, scope, target string) error {
	s := db.scope(ctx)
	query := `UPDATE alert_silences SET active = false WHERE scope = $1 AND target = $2 AND active = true` + s.and(3)
	args := []interface{}{scope, target}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) DeactivateSilenceByID(ctx context.Context, id int) error {
	s := db.scope(ctx)
	query := `UPDATE alert_silences SET active = false WHERE id = $1 AND active = true` + s.and(2)
	args := []interface{}{id}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("silence not found or already inactive")
	}
	return nil
}

func (db *PostgresDB) GetActiveSilences(ctx context.Context) ([]models.AlertSilence, error) {
	s := db.scope(ctx)
	query := `SELECT id, scope, target, channel, silenced_by, silenced_at, expires_at, reason
		FROM alert_silences
		WHERE active = true AND (expires_at IS NULL OR expires_at > NOW())` + s.and(1)
	rows, err := db.Pool.Query(ctx, query, s.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var silences []models.AlertSilence
	for rows.Next() {
		var si models.AlertSilence
		if err := rows.Scan(&si.ID, &si.Scope, &si.Target, &si.Channel, &si.SilencedBy, &si.SilencedAt, &si.ExpiresAt, &si.Reason); err != nil {
			return nil, err
		}
		si.Active = true
		silences = append(silences, si)
	}
	return silences, rows.Err()
}

func (db *PostgresDB) IsCheckSilenced(ctx context.Context, checkUUID, project string) (bool, error) {
	s := db.scope(ctx)
	query := `SELECT EXISTS(
			SELECT 1 FROM alert_silences
			WHERE active = true
			AND (expires_at IS NULL OR expires_at > NOW())` + s.and(3) + `
			AND (
				(scope = 'check' AND target = $1)
				OR (scope = 'project' AND target = $2)
			)
		)`
	args := []interface{}{checkUUID, project}
	args = append(args, s.args()...)
	var exists bool
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// IsChannelSilenced checks if a specific channel is silenced for a check.
// Returns true if there's a silence matching the check/project AND (channel='' OR channel=channelName).
func (db *PostgresDB) IsChannelSilenced(ctx context.Context, checkUUID, project, channelName string) (bool, error) {
	s := db.scope(ctx)
	query := `SELECT EXISTS(
			SELECT 1 FROM alert_silences
			WHERE active = true
			AND (expires_at IS NULL OR expires_at > NOW())` + s.and(4) + `
			AND (
				(scope = 'check' AND target = $1)
				OR (scope = 'project' AND target = $2)
			)
			AND (channel = '' OR channel = $3)
		)`
	args := []interface{}{checkUUID, project, channelName}
	args = append(args, s.args()...)
	var exists bool
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

func (db *PostgresDB) GetUnhealthyChecks(ctx context.Context) ([]models.CheckDefinition, error) {
	s := db.scope(ctx)
	query := `SELECT uuid, name, project, group_name, type, is_healthy, last_message, last_run
		 FROM check_definitions WHERE is_healthy = false AND enabled = true` + s.and(1)
	rows, err := db.Pool.Query(ctx, query, s.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		var c models.CheckDefinition
		var isHealthy *bool
		var lastMessage *string
		var lastRun *time.Time
		if err := rows.Scan(&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &isHealthy, &lastMessage, &lastRun); err != nil {
			return nil, err
		}
		if isHealthy != nil {
			c.IsHealthy = *isHealthy
		}
		if lastMessage != nil {
			c.LastMessage = *lastMessage
		}
		if lastRun != nil {
			c.LastRun = *lastRun
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

// Alert history

func (db *PostgresDB) CreateAlertEvent(ctx context.Context, event models.AlertEvent) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(
		`INSERT INTO alert_history (check_uuid, check_name, project, group_name, check_type, message, alert_type, region%s)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8%s)`,
		s.insertCol(), s.insertPlaceholder(9))
	args := []interface{}{event.CheckUUID, event.CheckName, event.Project, event.GroupName, event.CheckType, event.Message, event.AlertType, event.Region}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) ResolveAlertEvent(ctx context.Context, checkUUID string) error {
	s := db.scope(ctx)
	query := `UPDATE alert_history SET is_resolved = true, resolved_at = NOW()
		 WHERE check_uuid = $1 AND is_resolved = false` + s.and(2)
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) GetAlertHistory(ctx context.Context, limit, offset int, filters models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	s := db.scope(ctx)

	// Build WHERE clause dynamically; always start with tenant_id filter when scoped.
	where := ""
	args := []interface{}{}
	argIdx := 1

	if s != nil {
		where += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, s.id)
		argIdx++
	}

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
	if filters.Since != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filters.Since)
		argIdx++
	}
	if filters.Until != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filters.Until)
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
		"SELECT id, check_uuid, check_name, project, group_name, check_type, message, alert_type, region, created_at, resolved_at, is_resolved FROM alert_history WHERE 1=1%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
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
		if err := rows.Scan(&e.ID, &e.CheckUUID, &e.CheckName, &e.Project, &e.GroupName, &e.CheckType, &e.Message, &e.AlertType, &e.Region, &e.CreatedAt, &e.ResolvedAt, &e.IsResolved); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// NilIfEmpty returns nil if s is empty, otherwise a pointer to s.
func NilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// UnmarshalConfig deserializes a check config JSON blob into the appropriate typed struct.
func UnmarshalConfig(checkType string, data []byte) (models.CheckConfig, error) {
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
	s := db.scope(ctx)
	query := `SELECT id, name, steps, created_at FROM escalation_policies` + s.where(1) + ` ORDER BY name`
	rows, err := db.Pool.Query(ctx, query, s.args()...)
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
	s := db.scope(ctx)
	query := `SELECT id, name, steps, created_at FROM escalation_policies WHERE name=$1` + s.and(2)
	args := []interface{}{name}
	args = append(args, s.args()...)
	var p models.EscalationPolicy
	var stepsJSON []byte
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
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
	s := db.scope(ctx)
	stepsJSON, err := json.Marshal(policy.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}
	query := fmt.Sprintf(`INSERT INTO escalation_policies (name, steps%s) VALUES ($1, $2%s)`,
		s.insertCol(), s.insertPlaceholder(3))
	args := []interface{}{policy.Name, stepsJSON}
	args = append(args, s.args()...)
	_, err = db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) UpdateEscalationPolicy(ctx context.Context, policy models.EscalationPolicy) error {
	s := db.scope(ctx)
	stepsJSON, err := json.Marshal(policy.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}
	query := `UPDATE escalation_policies SET steps=$2 WHERE name=$1` + s.and(3)
	args := []interface{}{policy.Name, stepsJSON}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("escalation policy not found")
	}
	return nil
}

func (db *PostgresDB) DeleteEscalationPolicy(ctx context.Context, name string) error {
	s := db.scope(ctx)
	query := `DELETE FROM escalation_policies WHERE name=$1` + s.and(2)
	args := []interface{}{name}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
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
	s := db.scope(ctx)
	query := `SELECT id, check_uuid, policy_name, step_index, notified_at
		 FROM escalation_notifications
		 WHERE check_uuid=$1 AND policy_name=$2` + s.and(3) + ` ORDER BY step_index`
	args := []interface{}{checkUUID, policyName}
	args = append(args, s.args()...)
	rows, err := db.Pool.Query(ctx, query, args...)
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
	s := db.scope(ctx)
	query := fmt.Sprintf(
		`INSERT INTO escalation_notifications (check_uuid, policy_name, step_index, notified_at%s)
		 VALUES ($1, $2, $3, $4%s)
		 ON CONFLICT (check_uuid, policy_name, step_index, notified_at) DO NOTHING`,
		s.insertCol(), s.insertPlaceholder(5))
	args := []interface{}{notification.CheckUUID, notification.PolicyName, notification.StepIndex, notification.NotifiedAt}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) DeleteEscalationNotifications(ctx context.Context, checkUUID string) error {
	s := db.scope(ctx)
	query := `DELETE FROM escalation_notifications WHERE check_uuid=$1` + s.and(2)
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

// Alert channels

func (db *PostgresDB) GetAllAlertChannels(ctx context.Context) ([]models.AlertChannel, error) {
	s := db.scope(ctx)
	query := `SELECT id, name, type, config, created_at, updated_at
		 FROM alert_channels` + s.where(1) + ` ORDER BY name`
	rows, err := db.Pool.Query(ctx, query, s.args()...)
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
	s := db.scope(ctx)
	query := `SELECT id, name, type, config, created_at, updated_at
		 FROM alert_channels WHERE name=$1` + s.and(2)
	args := []interface{}{name}
	args = append(args, s.args()...)
	var ch models.AlertChannel
	err := db.Pool.QueryRow(ctx, query, args...).
		Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return ch, fmt.Errorf("alert channel not found: %w", err)
	}
	return ch, nil
}

func (db *PostgresDB) CreateAlertChannel(ctx context.Context, channel models.AlertChannel) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(`INSERT INTO alert_channels (name, type, config%s) VALUES ($1, $2, $3%s)`,
		s.insertCol(), s.insertPlaceholder(4))
	args := []interface{}{channel.Name, channel.Type, channel.Config}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) UpdateAlertChannel(ctx context.Context, channel models.AlertChannel) error {
	s := db.scope(ctx)
	query := `UPDATE alert_channels SET type=$2, config=$3, updated_at=NOW() WHERE name=$1` + s.and(4)
	args := []interface{}{channel.Name, channel.Type, channel.Config}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("alert channel not found")
	}
	return nil
}

func (db *PostgresDB) DeleteAlertChannel(ctx context.Context, name string) error {
	s := db.scope(ctx)
	query := `DELETE FROM alert_channels WHERE name=$1` + s.and(2)
	args := []interface{}{name}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
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
	s := db.scope(ctx)
	query := fmt.Sprintf(
		`INSERT INTO telegram_alert_threads (check_uuid, chat_id, message_id%s) VALUES ($1, $2, $3%s)`,
		s.insertCol(), s.insertPlaceholder(4))
	args := []interface{}{checkUUID, chatID, messageID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) GetUnresolvedTelegramThread(ctx context.Context, checkUUID string) (models.TelegramAlertThread, error) {
	s := db.scope(ctx)
	query := `SELECT id, check_uuid, chat_id, message_id, is_resolved, created_at, resolved_at
		 FROM telegram_alert_threads WHERE check_uuid=$1 AND is_resolved=false` + s.and(2) + ` ORDER BY created_at DESC LIMIT 1`
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	var t models.TelegramAlertThread
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.CheckUUID, &t.ChatID, &t.MessageID, &t.IsResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.TelegramAlertThread{}, err
	}
	return t, nil
}

func (db *PostgresDB) GetTelegramThreadByMessage(ctx context.Context, chatID string, messageID int) (models.TelegramAlertThread, error) {
	s := db.scope(ctx)
	query := `SELECT id, check_uuid, chat_id, message_id, is_resolved, created_at, resolved_at
		 FROM telegram_alert_threads WHERE chat_id=$1 AND message_id=$2 AND is_resolved=false` + s.and(3)
	args := []interface{}{chatID, messageID}
	args = append(args, s.args()...)
	var t models.TelegramAlertThread
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.CheckUUID, &t.ChatID, &t.MessageID, &t.IsResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.TelegramAlertThread{}, err
	}
	return t, nil
}

func (db *PostgresDB) ResolveTelegramThread(ctx context.Context, checkUUID string) error {
	s := db.scope(ctx)
	query := `UPDATE telegram_alert_threads SET is_resolved=true, resolved_at=NOW() WHERE check_uuid=$1 AND is_resolved=false` + s.and(2)
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) CreateDiscordThread(ctx context.Context, checkUUID, channelID, messageID, threadID string) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(
		`INSERT INTO discord_alert_threads (check_uuid, channel_id, message_id, thread_id%s) VALUES ($1, $2, $3, $4%s)`,
		s.insertCol(), s.insertPlaceholder(5))
	args := []interface{}{checkUUID, channelID, messageID, threadID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) GetUnresolvedDiscordThread(ctx context.Context, checkUUID string) (models.DiscordAlertThread, error) {
	s := db.scope(ctx)
	query := `SELECT id, check_uuid, channel_id, message_id, thread_id, is_resolved, created_at, resolved_at
		 FROM discord_alert_threads WHERE check_uuid=$1 AND is_resolved=false` + s.and(2) + ` ORDER BY created_at DESC LIMIT 1`
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	var t models.DiscordAlertThread
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.CheckUUID, &t.ChannelID, &t.MessageID, &t.ThreadID, &t.IsResolved, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return models.DiscordAlertThread{}, err
	}
	return t, nil
}

func (db *PostgresDB) ResolveDiscordThread(ctx context.Context, checkUUID string) error {
	s := db.scope(ctx)
	query := `UPDATE discord_alert_threads SET is_resolved=true, resolved_at=NOW() WHERE check_uuid=$1 AND is_resolved=false` + s.and(2)
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

// MigrateLegacyAlertFields is a no-op. The legacy alert_type and alert_destination
// columns have been dropped. Kept for interface compatibility.
func (db *PostgresDB) MigrateLegacyAlertFields(ctx context.Context) (int, error) {
	return 0, nil
}

// --- Multi-region check results ---

func (db *PostgresDB) GetLatestRegionResults(ctx context.Context, checkUUID string) ([]models.CheckResult, error) {
	s := db.scope(ctx)
	query := `SELECT DISTINCT ON (region) id, check_uuid, region, is_healthy, message, created_at, cycle_key, evaluated_at
		 FROM check_results
		 WHERE check_uuid = $1` + s.and(2) + `
		 ORDER BY region, created_at DESC`
	args := []interface{}{checkUUID}
	args = append(args, s.args()...)
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.CheckResult
	for rows.Next() {
		var r models.CheckResult
		if err := rows.Scan(&r.ID, &r.CheckUUID, &r.Region, &r.IsHealthy, &r.Message, &r.CreatedAt, &r.CycleKey, &r.EvaluatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *PostgresDB) InsertCheckResult(ctx context.Context, result models.CheckResult) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(
		`INSERT INTO check_results (check_uuid, region, is_healthy, message, created_at, cycle_key%s)
		 VALUES ($1, $2, $3, $4, $5, $6%s)`,
		s.insertCol(), s.insertPlaceholder(7))
	args := []interface{}{result.CheckUUID, result.Region, result.IsHealthy, result.Message, result.CreatedAt, result.CycleKey}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) GetUnevaluatedCycles(ctx context.Context, minRegions int, timeout time.Duration) ([]UnevaluatedCycle, error) {
	s := db.scope(ctx)
	cutoff := time.Now().Add(-timeout)
	query := `SELECT check_uuid, cycle_key, COUNT(DISTINCT region) AS region_count
		 FROM check_results
		 WHERE evaluated_at IS NULL` + s.and(3) + `
		 GROUP BY check_uuid, cycle_key
		 HAVING COUNT(DISTINCT region) >= $1 OR MIN(created_at) < $2
		 ORDER BY cycle_key ASC
		 LIMIT 500`
	args := []interface{}{minRegions, cutoff}
	args = append(args, s.args()...)
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cycles []UnevaluatedCycle
	for rows.Next() {
		var c UnevaluatedCycle
		if err := rows.Scan(&c.CheckUUID, &c.CycleKey, &c.RegionCount); err != nil {
			return nil, err
		}
		cycles = append(cycles, c)
	}
	return cycles, rows.Err()
}

func (db *PostgresDB) ClaimCycleForEvaluation(ctx context.Context, checkUUID string, cycleKey time.Time) (bool, error) {
	s := db.scope(ctx)
	query := `UPDATE check_results SET evaluated_at = NOW()
		 WHERE check_uuid = $1 AND cycle_key = $2 AND evaluated_at IS NULL` + s.and(3)
	args := []interface{}{checkUUID, cycleKey}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return cmdTag.RowsAffected() > 0, nil
}

func (db *PostgresDB) GetCycleResults(ctx context.Context, checkUUID string, cycleKey time.Time) ([]models.CheckResult, error) {
	s := db.scope(ctx)
	query := `SELECT id, check_uuid, region, is_healthy, message, created_at, cycle_key, evaluated_at
		 FROM check_results
		 WHERE check_uuid = $1 AND cycle_key = $2` + s.and(3) + `
		 ORDER BY created_at ASC`
	args := []interface{}{checkUUID, cycleKey}
	args = append(args, s.args()...)
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.CheckResult
	for rows.Next() {
		var r models.CheckResult
		if err := rows.Scan(&r.ID, &r.CheckUUID, &r.Region, &r.IsHealthy, &r.Message, &r.CreatedAt, &r.CycleKey, &r.EvaluatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// --- Project & Group Settings (hierarchical overrides) ---

func (db *PostgresDB) GetProjectSettings(ctx context.Context, project string) (*models.ProjectSettings, error) {
	s := db.scope(ctx)
	query := `SELECT project, enabled, duration, re_alert_interval, maintenance_until, maintenance_reason, updated_at FROM project_settings WHERE project = $1` + s.and(2)
	args := []interface{}{project}
	args = append(args, s.args()...)
	row := db.Pool.QueryRow(ctx, query, args...)

	var ps models.ProjectSettings
	var maintenanceUntil *time.Time
	err := row.Scan(&ps.Project, &ps.Enabled, &ps.Duration, &ps.ReAlertInterval, &maintenanceUntil, &ps.MaintenanceReason, &ps.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	ps.MaintenanceUntil = maintenanceUntil
	return &ps, nil
}

func (db *PostgresDB) UpsertProjectSettings(ctx context.Context, settings models.ProjectSettings) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(`INSERT INTO project_settings (project, enabled, duration, re_alert_interval, maintenance_until, maintenance_reason, updated_at%s)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()%s)
		ON CONFLICT (project%s) DO UPDATE SET
			enabled = $2, duration = $3, re_alert_interval = $4,
			maintenance_until = $5, maintenance_reason = $6, updated_at = NOW()`,
		s.insertCol(), s.insertPlaceholder(7),
		func() string {
			if s != nil {
				return ", tenant_id"
			}
			return ""
		}())
	args := []interface{}{settings.Project, settings.Enabled, settings.Duration, settings.ReAlertInterval, settings.MaintenanceUntil, settings.MaintenanceReason}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) GetAllProjectSettings(ctx context.Context) ([]models.ProjectSettings, error) {
	s := db.scope(ctx)
	query := `SELECT project, enabled, duration, re_alert_interval, maintenance_until, maintenance_reason, updated_at FROM project_settings` + s.where(1)
	args := s.args()
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProjectSettings
	for rows.Next() {
		var ps models.ProjectSettings
		var maintenanceUntil *time.Time
		if err := rows.Scan(&ps.Project, &ps.Enabled, &ps.Duration, &ps.ReAlertInterval, &maintenanceUntil, &ps.MaintenanceReason, &ps.UpdatedAt); err != nil {
			return nil, err
		}
		ps.MaintenanceUntil = maintenanceUntil
		result = append(result, ps)
	}
	return result, rows.Err()
}

func (db *PostgresDB) GetGroupSettings(ctx context.Context, project, groupName string) (*models.GroupSettings, error) {
	s := db.scope(ctx)
	query := `SELECT project, group_name, enabled, duration, re_alert_interval, maintenance_until, maintenance_reason, updated_at FROM group_settings WHERE project = $1 AND group_name = $2` + s.and(3)
	args := []interface{}{project, groupName}
	args = append(args, s.args()...)
	row := db.Pool.QueryRow(ctx, query, args...)

	var gs models.GroupSettings
	var maintenanceUntil *time.Time
	err := row.Scan(&gs.Project, &gs.GroupName, &gs.Enabled, &gs.Duration, &gs.ReAlertInterval, &maintenanceUntil, &gs.MaintenanceReason, &gs.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	gs.MaintenanceUntil = maintenanceUntil
	return &gs, nil
}

func (db *PostgresDB) UpsertGroupSettings(ctx context.Context, settings models.GroupSettings) error {
	s := db.scope(ctx)
	query := fmt.Sprintf(`INSERT INTO group_settings (project, group_name, enabled, duration, re_alert_interval, maintenance_until, maintenance_reason, updated_at%s)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW()%s)
		ON CONFLICT (project, group_name%s) DO UPDATE SET
			enabled = $3, duration = $4, re_alert_interval = $5,
			maintenance_until = $6, maintenance_reason = $7, updated_at = NOW()`,
		s.insertCol(), s.insertPlaceholder(8),
		func() string {
			if s != nil {
				return ", tenant_id"
			}
			return ""
		}())
	args := []interface{}{settings.Project, settings.GroupName, settings.Enabled, settings.Duration, settings.ReAlertInterval, settings.MaintenanceUntil, settings.MaintenanceReason}
	args = append(args, s.args()...)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) GetAllGroupSettings(ctx context.Context) ([]models.GroupSettings, error) {
	s := db.scope(ctx)
	query := `SELECT project, group_name, enabled, duration, re_alert_interval, maintenance_until, maintenance_reason, updated_at FROM group_settings` + s.where(1)
	args := s.args()
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.GroupSettings
	for rows.Next() {
		var gs models.GroupSettings
		var maintenanceUntil *time.Time
		if err := rows.Scan(&gs.Project, &gs.GroupName, &gs.Enabled, &gs.Duration, &gs.ReAlertInterval, &maintenanceUntil, &gs.MaintenanceReason, &gs.UpdatedAt); err != nil {
			return nil, err
		}
		gs.MaintenanceUntil = maintenanceUntil
		result = append(result, gs)
	}
	return result, rows.Err()
}

func (db *PostgresDB) PurgeOldCheckResults(ctx context.Context, olderThan time.Duration) (int64, error) {
	s := db.scope(ctx)
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM check_results WHERE created_at < $1` + s.and(2)
	args := []interface{}{cutoff}
	args = append(args, s.args()...)
	cmdTag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}
