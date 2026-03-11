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
			logrus.Errorf("Migration failed: %v", err)
			// decide if we want to fail hard or continue
		}
	}

	return &PostgresDB{Pool: pool}, nil
}

func (db *PostgresDB) Close() {
	db.Pool.Close()
}

func (db *PostgresDB) GetAllCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	rows, err := db.Pool.Query(ctx, "SELECT uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config FROM check_definitions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		var c models.CheckDefinition
		var configJSON, actorConfigJSON []byte
		err := rows.Scan(
			&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &c.Description, &c.Enabled,
			&c.CreatedAt, &c.UpdatedAt, &c.LastRun, &c.IsHealthy, &c.LastMessage, &c.LastAlertSent,
			&c.Duration, &c.ActorType, &c.AlertType, &c.AlertDestination, &configJSON, &actorConfigJSON,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal Polymorphic Config
		// We need to determine the type to unmarshal into based on c.Type
		// Similar logic to UnmarshalBSON but using JSON
		if len(configJSON) > 0 {
			// simplified for now, ideally we use a helper or the model's unmarshal logic if adapted
			// But model's UnmarshalBSON is for BSON. We might need a UnmarshalJSON or do it here.
			// For now, let's implement a helper unmarshalConfig
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

		checks = append(checks, c)
	}
	return checks, nil
}

func (db *PostgresDB) GetEnabledCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	// Similar to GetAll but with WHERE enabled = true
	// For brevity, copy-paste logic or helper? Let's just implement it.
	rows, err := db.Pool.Query(ctx, "SELECT uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config FROM check_definitions WHERE enabled=$1", true)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.CheckDefinition
	for rows.Next() {
		var c models.CheckDefinition
		var configJSON, actorConfigJSON []byte
		err := rows.Scan(
			&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &c.Description, &c.Enabled,
			&c.CreatedAt, &c.UpdatedAt, &c.LastRun, &c.IsHealthy, &c.LastMessage, &c.LastAlertSent,
			&c.Duration, &c.ActorType, &c.AlertType, &c.AlertDestination, &configJSON, &actorConfigJSON,
		)
		if err != nil {
			return nil, err
		}
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
		checks = append(checks, c)
	}
	return checks, nil
}

func (db *PostgresDB) GetCheckDefinitionByUUID(ctx context.Context, uuid string) (models.CheckDefinition, error) {
	var c models.CheckDefinition
	var configJSON, actorConfigJSON []byte
	err := db.Pool.QueryRow(ctx, "SELECT uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config FROM check_definitions WHERE uuid=$1", uuid).Scan(
		&c.UUID, &c.Name, &c.Project, &c.GroupName, &c.Type, &c.Description, &c.Enabled,
		&c.CreatedAt, &c.UpdatedAt, &c.LastRun, &c.IsHealthy, &c.LastMessage, &c.LastAlertSent,
		&c.Duration, &c.ActorType, &c.AlertType, &c.AlertDestination, &configJSON, &actorConfigJSON,
	)
	if err != nil {
		return models.CheckDefinition{}, err
	}
	if len(configJSON) > 0 {
		conf, err := unmarshalConfig(c.Type, configJSON)
		if err != nil {
			return c, err
		}
		c.Config = conf
	}
	if len(actorConfigJSON) > 0 && c.ActorType == "webhook" {
		var webhookConf models.WebhookConfig
		if err := json.Unmarshal(actorConfigJSON, &webhookConf); err == nil {
			c.ActorConfig = &webhookConf
		}
	}

	return c, nil
}

func (db *PostgresDB) CreateCheckDefinition(ctx context.Context, def models.CheckDefinition) (string, error) {
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)

	// insert
	_, err := db.Pool.Exec(ctx, `INSERT INTO check_definitions 
    (uuid, name, project, group_name, type, description, enabled, created_at, updated_at, last_run, is_healthy, last_message, last_alert_sent, duration, actor_type, alert_type, alert_destination, config, actor_config)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
		def.UUID, def.Name, def.Project, def.GroupName, def.Type, def.Description, def.Enabled, def.CreatedAt, def.UpdatedAt, def.LastRun, def.IsHealthy, def.LastMessage, def.LastAlertSent, def.Duration, def.ActorType, def.AlertType, def.AlertDestination, configJSON, actorConfigJSON)

	if err != nil {
		return "", err
	}
	return def.UUID, nil
}

func (db *PostgresDB) UpdateCheckDefinition(ctx context.Context, def models.CheckDefinition) error {
	configJSON, _ := json.Marshal(def.Config)
	actorConfigJSON, _ := json.Marshal(def.ActorConfig)

	// update
	cmdTag, err := db.Pool.Exec(ctx, `UPDATE check_definitions SET
    name=$2, project=$3, group_name=$4, type=$5, description=$6, enabled=$7, updated_at=$8, last_run=$9, is_healthy=$10, last_message=$11, last_alert_sent=$12, duration=$13, actor_type=$14, alert_type=$15, alert_destination=$16, config=$17, actor_config=$18
    WHERE uuid=$1`,
		def.UUID, def.Name, def.Project, def.GroupName, def.Type, def.Description, def.Enabled, time.Now(), def.LastRun, def.IsHealthy, def.LastMessage, def.LastAlertSent, def.Duration, def.ActorType, def.AlertType, def.AlertDestination, configJSON, actorConfigJSON)

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
		"http":    "3s",
		"tcp":     "5s",
		"icmp":    "5s",
		"pgsql":   "10s",
		"mysql":   "10s",
		"default": "5s",
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
	}
	return nil, nil
}
