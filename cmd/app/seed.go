package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/imcitius/checker/demo"
	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// wipAndReseed deletes all existing check definitions and reseeds from the seed file.
// Used in DEMO_MODE to ensure a clean state on every startup.
func wipAndReseed(ctx context.Context, repo db.Repository, filePath string) error {
	all, err := repo.GetAllCheckDefinitions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list checks for wipe: %w", err)
	}
	uuids := make([]string, 0, len(all))
	for _, c := range all {
		uuids = append(uuids, c.UUID)
	}
	if len(uuids) > 0 {
		if _, err := repo.BulkDeleteCheckDefinitions(ctx, uuids); err != nil {
			return fmt.Errorf("failed to wipe checks: %w", err)
		}
		logrus.Infof("Demo mode: wiped %d existing checks", len(uuids))
	}
	return seedFromFile(repo, filePath)
}

// seedFromFile reads the YAML seed data and imports check definitions into the repository.
// It prefers reading from the on-disk file at the given path (allows runtime override),
// falling back to the embedded seed data compiled into the binary.
func seedFromFile(repo db.Repository, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Fall back to embedded seed data
		logrus.Debugf("Seed file %s not found on disk, using embedded version", filePath)
		data = demo.SeedYAML
	}

	if len(data) == 0 {
		return fmt.Errorf("seed data is empty")
	}

	var payload models.CheckImportPayload
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to parse seed YAML: %w", err)
	}

	if len(payload.Checks) == 0 {
		return fmt.Errorf("seed file contains no checks")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created := 0
	for _, item := range payload.Checks {
		def := seedItemToCheckDefinition(item)
		def.UUID = uuid.New().String()
		def.CreatedAt = time.Now()
		def.UpdatedAt = time.Now()

		if _, err := repo.CreateCheckDefinition(ctx, def); err != nil {
			logrus.Warnf("Failed to seed check %q: %v", item.Name, err)
			continue
		}
		created++
	}

	logrus.Infof("Demo mode: seeded %d/%d checks from %s", created, len(payload.Checks), filePath)
	return nil
}

// seedItemToCheckDefinition converts a CheckImportItem to a CheckDefinition.
// This mirrors the logic in web/import_handlers.go but avoids depending on that package.
func seedItemToCheckDefinition(item models.CheckImportItem) models.CheckDefinition {
	def := models.CheckDefinition{
		Name:             item.Name,
		Project:          item.Project,
		GroupName:        item.GroupName,
		Type:             item.Type,
		Description:      item.Description,
		Duration:         item.Duration,
		ActorType: item.ActorType,
		Severity:         item.Severity,
		AlertChannels:    item.AlertChannels,
		ReAlertInterval:  item.ReAlertInterval,
		RetryCount:       item.RetryCount,
		RetryInterval:    item.RetryInterval,
	}

	if item.Enabled != nil {
		def.Enabled = *item.Enabled
	} else {
		def.Enabled = true
	}

	// Set fallback duration
	if def.Duration == "" {
		def.Duration = "1m"
	}

	// Create type-specific config
	switch item.Type {
	case "http":
		httpCfg := &models.HTTPCheckConfig{
			URL:                 item.URL,
			Timeout:             item.Timeout,
			Answer:              item.Answer,
			Code:                item.Code,
			Headers:             item.Headers,
			Cookies:             item.Cookies,
			SSLExpirationPeriod: item.SSLExpirationPeriod,
		}
		if item.AnswerPresent != nil {
			httpCfg.AnswerPresent = *item.AnswerPresent
		}
		if item.SkipCheckSSL != nil {
			httpCfg.SkipCheckSSL = *item.SkipCheckSSL
		}
		if item.StopFollowRedirects != nil {
			httpCfg.StopFollowRedirects = *item.StopFollowRedirects
		}
		if item.Auth != nil {
			httpCfg.Auth = models.AuthConfig{
				User:     item.Auth.User,
				Password: item.Auth.Password,
			}
		}
		def.Config = httpCfg
	case "tcp":
		def.Config = &models.TCPCheckConfig{
			Host:    item.Host,
			Port:    item.Port,
			Timeout: item.Timeout,
		}
	case "icmp":
		def.Config = &models.ICMPCheckConfig{
			Host:    item.Host,
			Timeout: item.Timeout,
		}
	case "dns":
		def.Config = &models.DNSCheckConfig{
			Host:       item.Host,
			Domain:     item.Domain,
			RecordType: item.RecordType,
			Timeout:    item.Timeout,
			Expected:   item.Expected,
		}
	case "ssl_cert":
		def.Config = &models.SSLCertCheckConfig{
			Host:              item.Host,
			Port:              item.Port,
			Timeout:           item.Timeout,
			ExpiryWarningDays: item.ExpiryWarningDays,
			ValidateChain:     item.ValidateChain,
		}
	case "domain_expiry":
		def.Config = &models.DomainExpiryCheckConfig{
			Domain:            item.Domain,
			Timeout:           item.Timeout,
			ExpiryWarningDays: item.ExpiryWarningDays,
		}
	case "mongodb":
		def.Config = &models.MongoDBCheckConfig{
			URI:     item.MongoDBURI,
			Timeout: item.Timeout,
		}
	case "ssh":
		def.Config = &models.SSHCheckConfig{
			Host:         item.Host,
			Port:         item.Port,
			Timeout:      item.Timeout,
			ExpectBanner: item.ExpectBanner,
		}
	case "smtp":
		def.Config = &models.SMTPCheckConfig{
			Host:     item.Host,
			Port:     item.Port,
			Timeout:  item.Timeout,
			StartTLS: item.StartTLS,
			Username: item.SMTPUsername,
			Password: item.SMTPPassword,
		}
	case "grpc_health":
		def.Config = &models.GRPCHealthCheckConfig{
			Host:    item.Host,
			Timeout: item.Timeout,
			UseTLS:  item.UseTLS,
		}
	case "websocket":
		def.Config = &models.WebSocketCheckConfig{
			URL:     item.URL,
			Timeout: item.Timeout,
		}
	case "mysql_query", "mysql_query_unixtime", "mysql_replication":
		cfg := &models.MySQLCheckConfig{
			Host:    item.Host,
			Port:    item.Port,
			Timeout: item.Timeout,
		}
		if item.MySQL != nil {
			cfg.UserName = item.MySQL.UserName
			cfg.Password = item.MySQL.Password
			cfg.DBName = item.MySQL.DBName
			cfg.Query = item.MySQL.Query
			cfg.Difference = item.MySQL.Difference
			cfg.Lag = item.MySQL.Lag
			cfg.ServerList = item.MySQL.ServerList
		}
		def.Config = cfg
	case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
		cfg := &models.PostgreSQLCheckConfig{
			Host:    item.Host,
			Port:    item.Port,
			Timeout: item.Timeout,
		}
		if item.PgSQL != nil {
			cfg.UserName = item.PgSQL.UserName
			cfg.Password = item.PgSQL.Password
			cfg.DBName = item.PgSQL.DBName
			cfg.Query = item.PgSQL.Query
			cfg.Difference = item.PgSQL.Difference
			cfg.Lag = item.PgSQL.Lag
			cfg.ServerList = item.PgSQL.ServerList
		}
		def.Config = cfg
	case "redis":
		def.Config = &models.RedisCheckConfig{
			Host:     item.Host,
			Port:     item.Port,
			Timeout:  item.Timeout,
			Password: item.RedisPassword,
			DB:       item.RedisDB,
		}
	case "passive":
		def.Config = &models.PassiveCheckConfig{
			Timeout: item.Timeout,
		}
	}

	return def
}
