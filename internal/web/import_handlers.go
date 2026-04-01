package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// ImportCheckDefinitions handles bulk import of check definitions from YAML or JSON.
//
// It supports two content types:
//   - application/json — parsed as JSON
//   - application/x-yaml, text/yaml, text/plain — parsed as YAML
//
// The payload is a CheckImportPayload (see models/import.go).
// When project + environment are set, checks are scoped to that combination.
// When prune=true, existing checks in scope not present in the payload are deleted.
func ImportCheckDefinitions(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	payload, err := parseImportPayload(c)
	if err != nil {
		logrus.Errorf("Failed to parse import payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to parse import payload: %v", err),
		})
		return
	}

	// Validate
	if len(payload.Checks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No checks provided in the import payload",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := executeImport(ctx, repo, payload)
	if err != nil {
		logrus.Errorf("Import execution failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Import failed: %v", err),
		})
		return
	}

	statusCode := http.StatusOK
	if result.Summary.Errors > 0 && result.Summary.Created+result.Summary.Updated == 0 {
		statusCode = http.StatusUnprocessableEntity
	}

	c.JSON(statusCode, result)
}

// ValidateImportPayload parses and validates the import payload without executing.
// Useful for dry-run / preview in the dashboard.
func ValidateImportPayload(c *gin.Context) {
	payload, err := parseImportPayload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to parse import payload: %v", err),
		})
		return
	}

	// Apply defaults and resolve to view models for preview
	resolved := resolveChecks(payload)

	validationErrors := validateChecks(resolved)

	c.JSON(http.StatusOK, gin.H{
		"valid":  len(validationErrors) == 0,
		"checks": resolved,
		"errors": validationErrors,
		"count":  len(resolved),
	})
}

// parseImportPayload reads the request body and parses it as YAML or JSON.
func parseImportPayload(c *gin.Context) (*models.CheckImportPayload, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer c.Request.Body.Close()

	if len(body) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	var payload models.CheckImportPayload

	// YAML parser also handles JSON, so use it universally
	if err := yaml.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse YAML/JSON: %w", err)
	}

	return &payload, nil
}

// resolveChecks applies payload-level defaults and project/environment to each check.
func resolveChecks(payload *models.CheckImportPayload) []models.CheckImportItem {
	resolved := make([]models.CheckImportItem, 0, len(payload.Checks))

	for _, check := range payload.Checks {
		// Apply project from payload if check doesn't specify its own
		if check.Project == "" && payload.Project != "" {
			check.Project = payload.Project
		}

		// Apply environment as group_name if check doesn't specify its own
		if check.GroupName == "" && payload.Environment != "" {
			check.GroupName = payload.Environment
		}

		// Apply defaults
		if check.Duration == "" && payload.Defaults.Duration != "" {
			check.Duration = payload.Defaults.Duration
		}
		if check.Timeout == "" && payload.Defaults.Timeout != "" {
			check.Timeout = payload.Defaults.Timeout
		}
		if check.ActorType == "" && payload.Defaults.ActorType != "" {
			check.ActorType = payload.Defaults.ActorType
		}
		if check.Severity == "" && payload.Defaults.Severity != "" {
			check.Severity = payload.Defaults.Severity
		}
		if len(check.AlertChannels) == 0 && len(payload.Defaults.AlertChannels) > 0 {
			check.AlertChannels = payload.Defaults.AlertChannels
		}
		if check.ReAlertInterval == "" && payload.Defaults.ReAlertInterval != "" {
			check.ReAlertInterval = payload.Defaults.ReAlertInterval
		}
		if check.RetryCount == 0 && payload.Defaults.RetryCount > 0 {
			check.RetryCount = payload.Defaults.RetryCount
		}
		if check.RetryInterval == "" && payload.Defaults.RetryInterval != "" {
			check.RetryInterval = payload.Defaults.RetryInterval
		}
		if check.Enabled == nil {
			if payload.Defaults.Enabled != nil {
				check.Enabled = payload.Defaults.Enabled
			} else {
				enabled := true
				check.Enabled = &enabled
			}
		}

		// Set fallback duration
		if check.Duration == "" {
			check.Duration = "1m"
		}

		resolved = append(resolved, check)
	}

	return resolved
}

// validateChecks returns validation errors for each check.
func validateChecks(checks []models.CheckImportItem) []models.CheckImportError {
	errors := make([]models.CheckImportError, 0)

	for i, check := range checks {
		if check.Name == "" {
			errors = append(errors, models.CheckImportError{
				Name:    check.Name,
				Index:   i,
				Message: "name is required",
			})
		}
		if check.Project == "" {
			errors = append(errors, models.CheckImportError{
				Name:    check.Name,
				Index:   i,
				Message: "project is required",
			})
		}
		if check.Type == "" {
			errors = append(errors, models.CheckImportError{
				Name:    check.Name,
				Index:   i,
				Message: "type is required",
			})
		}

		// Type-specific validation
		switch check.Type {
		case "http":
			if check.URL == "" {
				errors = append(errors, models.CheckImportError{
					Name:    check.Name,
					Index:   i,
					Message: "url is required for http checks",
				})
			}
		case "tcp":
			if check.Host == "" {
				errors = append(errors, models.CheckImportError{
					Name:    check.Name,
					Index:   i,
					Message: "host is required for tcp checks",
				})
			}
		case "icmp":
			if check.Host == "" {
				errors = append(errors, models.CheckImportError{
					Name:    check.Name,
					Index:   i,
					Message: "host is required for icmp checks",
				})
			}
		case "domain_expiry":
			if check.Domain == "" {
				errors = append(errors, models.CheckImportError{
					Name:    check.Name,
					Index:   i,
					Message: "domain is required for domain_expiry checks",
				})
			}
		}
	}

	return errors
}

// executeImport performs the actual import: upsert checks, optionally prune.
func executeImport(ctx context.Context, repo db.Repository, payload *models.CheckImportPayload) (*models.CheckImportResult, error) {
	resolved := resolveChecks(payload)
	validationErrors := validateChecks(resolved)

	result := &models.CheckImportResult{
		Created: []models.CheckImportResultItem{},
		Updated: []models.CheckImportResultItem{},
		Deleted: []models.CheckImportResultItem{},
		Errors:  []models.CheckImportError{},
	}

	// Add validation errors
	result.Errors = append(result.Errors, validationErrors...)

	// Build a set of valid check indices (skip ones with validation errors)
	errorIndices := make(map[int]bool)
	for _, e := range validationErrors {
		errorIndices[e.Index] = true
	}

	// Fetch existing checks for matching
	allDefs, err := repo.GetAllCheckDefinitions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing checks: %w", err)
	}

	// Build lookup: (project, group_name, name) -> existing check
	type checkKey struct {
		project   string
		groupName string
		name      string
	}
	existingByKey := make(map[checkKey]models.CheckDefinition)
	for _, def := range allDefs {
		key := checkKey{
			project:   def.Project,
			groupName: def.GroupName,
			name:      def.Name,
		}
		existingByKey[key] = def
	}

	// Track which existing checks in scope were seen (for pruning)
	seenUUIDs := make(map[string]bool)

	// Process each check
	for i, check := range resolved {
		if errorIndices[i] {
			continue
		}

		key := checkKey{
			project:   check.Project,
			groupName: check.GroupName,
			name:      check.Name,
		}

		// Convert import item directly to domain model
		def := importItemToCheckDefinition(check)

		if existing, found := existingByKey[key]; found {
			// Update existing check
			def.UUID = existing.UUID
			def.CreatedAt = existing.CreatedAt
			def.UpdatedAt = time.Now()
			def.ID = existing.ID
			def.LastRun = existing.LastRun
			def.IsHealthy = existing.IsHealthy
			def.LastMessage = existing.LastMessage
			def.LastAlertSent = existing.LastAlertSent

			if err := repo.UpdateCheckDefinition(ctx, def); err != nil {
				result.Errors = append(result.Errors, models.CheckImportError{
					Name:    check.Name,
					Index:   i,
					Message: fmt.Sprintf("failed to update: %v", err),
				})
			} else {
				result.Updated = append(result.Updated, models.CheckImportResultItem{
					Name:    check.Name,
					UUID:    def.UUID,
					Project: check.Project,
				})
				seenUUIDs[def.UUID] = true
			}
		} else {
			// Create new check
			def.UUID = uuid.New().String()
			def.CreatedAt = time.Now()
			def.UpdatedAt = time.Now()

			id, err := repo.CreateCheckDefinition(ctx, def)
			if err != nil {
				result.Errors = append(result.Errors, models.CheckImportError{
					Name:    check.Name,
					Index:   i,
					Message: fmt.Sprintf("failed to create: %v", err),
				})
			} else {
				// id is the UUID returned by CreateCheckDefinition
				if id != "" {
					def.UUID = id
				}
				result.Created = append(result.Created, models.CheckImportResultItem{
					Name:    check.Name,
					UUID:    def.UUID,
					Project: check.Project,
				})
				seenUUIDs[def.UUID] = true
			}
		}
	}

	// Prune: delete checks in scope not present in payload
	if payload.Prune && payload.Project != "" {
		for _, def := range allDefs {
			if def.Project != payload.Project {
				continue
			}
			// If environment is set, only prune within that group
			if payload.Environment != "" && def.GroupName != payload.Environment {
				continue
			}
			if !seenUUIDs[def.UUID] {
				if err := repo.DeleteCheckDefinition(ctx, def.UUID); err != nil {
					result.Errors = append(result.Errors, models.CheckImportError{
						Name:    def.Name,
						Index:   -1,
						Message: fmt.Sprintf("failed to prune: %v", err),
					})
				} else {
					result.Deleted = append(result.Deleted, models.CheckImportResultItem{
						Name:    def.Name,
						UUID:    def.UUID,
						Project: def.Project,
					})
				}
			}
		}
	}

	result.Summary = models.CheckImportSummary{
		Total:   len(resolved),
		Created: len(result.Created),
		Updated: len(result.Updated),
		Deleted: len(result.Deleted),
		Errors:  len(result.Errors),
	}

	logrus.Infof("Import completed: %d created, %d updated, %d deleted, %d errors (source: %s)",
		result.Summary.Created, result.Summary.Updated, result.Summary.Deleted, result.Summary.Errors,
		payload.Source)

	return result, nil
}

// importItemToCheckDefinition converts an import item directly to a CheckDefinition,
// creating the appropriate typed Config for each check type.
func importItemToCheckDefinition(item models.CheckImportItem) models.CheckDefinition {
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
		} else if item.Answer != "" {
			// Default to true when answer is specified but answer_present is not
			httpCfg.AnswerPresent = true
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
	case "passive":
		def.Config = &models.PassiveCheckConfig{
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
			cfg.DBName = item.MySQL.DBName
			cfg.Query = item.MySQL.Query
			cfg.ServerList = item.MySQL.ServerList
		}
		def.Config = cfg
	case "mongodb":
		def.Config = &models.MongoDBCheckConfig{
			URI:     item.MongoDBURI,
			Timeout: item.Timeout,
		}
	case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
		cfg := &models.PostgreSQLCheckConfig{
			Host:    item.Host,
			Port:    item.Port,
			Timeout: item.Timeout,
		}
		if item.PgSQL != nil {
			cfg.UserName = item.PgSQL.UserName
			cfg.DBName = item.PgSQL.DBName
			cfg.Query = item.PgSQL.Query
			cfg.ServerList = item.PgSQL.ServerList
		}
		def.Config = cfg
	case "domain_expiry":
		def.Config = &models.DomainExpiryCheckConfig{
			Domain:            item.Domain,
			Timeout:           item.Timeout,
			ExpiryWarningDays: item.ExpiryWarningDays,
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
	case "ssh":
		def.Config = &models.SSHCheckConfig{
			Host:         item.Host,
			Port:         item.Port,
			Timeout:      item.Timeout,
			ExpectBanner: item.ExpectBanner,
		}
	case "redis":
		def.Config = &models.RedisCheckConfig{
			Host:     item.Host,
			Port:     item.Port,
			Timeout:  item.Timeout,
			Password: item.RedisPassword,
			DB:       item.RedisDB,
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
	}

	return def
}

// ExportCheckDefinitions exports all check definitions as YAML.
// Supports optional project and environment query filters.
func ExportCheckDefinitions(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	projectFilter := c.Query("project")
	environmentFilter := c.Query("environment")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	defs, err := repo.GetAllCheckDefinitions(ctx)
	if err != nil {
		logrus.Errorf("Failed to get check definitions for export: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch check definitions",
		})
		return
	}

	// Filter
	var filtered []models.CheckDefinition
	for _, def := range defs {
		if projectFilter != "" && def.Project != projectFilter {
			continue
		}
		if environmentFilter != "" && def.GroupName != environmentFilter {
			continue
		}
		filtered = append(filtered, def)
	}

	// Build export payload
	payload := models.CheckImportPayload{
		Project:     projectFilter,
		Environment: environmentFilter,
		Source:      "export",
	}

	for _, def := range filtered {
		item := models.CheckImportItem{
			Name:             def.Name,
			Project:          def.Project,
			GroupName:        def.GroupName,
			Type:             def.Type,
			Description:      def.Description,
			Duration:         def.Duration,
			ActorType: def.ActorType,
			Severity:         def.Severity,
			AlertChannels:    def.AlertChannels,
			ReAlertInterval:  def.ReAlertInterval,
			RetryCount:       def.RetryCount,
			RetryInterval:    def.RetryInterval,
		}

		enabled := def.Enabled
		item.Enabled = &enabled

		// Extract config fields
		if def.Config != nil {
			switch cfg := def.Config.(type) {
			case *models.HTTPCheckConfig:
				item.URL = cfg.URL
				item.Timeout = cfg.Timeout
				item.Answer = cfg.Answer
				if cfg.AnswerPresent {
					ap := true
					item.AnswerPresent = &ap
				}
				item.Code = cfg.Code
				item.Headers = cfg.Headers
				item.Cookies = cfg.Cookies
				if cfg.SkipCheckSSL {
					sc := true
					item.SkipCheckSSL = &sc
				}
				item.SSLExpirationPeriod = cfg.SSLExpirationPeriod
				if cfg.StopFollowRedirects {
					sfr := true
					item.StopFollowRedirects = &sfr
				}
				if cfg.Auth.User != "" || cfg.Auth.Password != "" {
					item.Auth = &models.AuthImportConfig{
						User:     cfg.Auth.User,
						Password: cfg.Auth.Password,
					}
				}
			case *models.TCPCheckConfig:
				item.Host = cfg.Host
				item.Port = cfg.Port
				item.Timeout = cfg.Timeout
			case *models.ICMPCheckConfig:
				item.Host = cfg.Host
			case *models.MySQLCheckConfig:
				item.Host = cfg.Host
				item.Port = cfg.Port
				item.MySQL = &models.DBImportConfig{
					UserName:   cfg.UserName,
					DBName:     cfg.DBName,
					Query:      cfg.Query,
					ServerList: cfg.ServerList,
				}
			case *models.MongoDBCheckConfig:
				item.MongoDBURI = cfg.URI
				item.Timeout = cfg.Timeout
			case *models.PostgreSQLCheckConfig:
				item.Host = cfg.Host
				item.Port = cfg.Port
				item.PgSQL = &models.DBImportConfig{
					UserName:   cfg.UserName,
					DBName:     cfg.DBName,
					Query:      cfg.Query,
					ServerList: cfg.ServerList,
				}
			case *models.DomainExpiryCheckConfig:
				item.Domain = cfg.Domain
				item.Timeout = cfg.Timeout
				item.ExpiryWarningDays = cfg.ExpiryWarningDays
			case *models.DNSCheckConfig:
				item.Host = cfg.Host
				item.Domain = cfg.Domain
				item.RecordType = cfg.RecordType
				item.Timeout = cfg.Timeout
				item.Expected = cfg.Expected
			case *models.SSLCertCheckConfig:
				item.Host = cfg.Host
				item.Port = cfg.Port
				item.Timeout = cfg.Timeout
				item.ExpiryWarningDays = cfg.ExpiryWarningDays
				item.ValidateChain = cfg.ValidateChain
			case *models.PassiveCheckConfig:
				item.Timeout = cfg.Timeout
			case *models.SSHCheckConfig:
				item.Host = cfg.Host
				item.Port = cfg.Port
				item.Timeout = cfg.Timeout
				item.ExpectBanner = cfg.ExpectBanner
			case *models.RedisCheckConfig:
				item.Host = cfg.Host
				item.Port = cfg.Port
				item.Timeout = cfg.Timeout
				item.RedisPassword = cfg.Password
				item.RedisDB = cfg.DB
			case *models.SMTPCheckConfig:
				item.Host = cfg.Host
				item.Port = cfg.Port
				item.Timeout = cfg.Timeout
				item.StartTLS = cfg.StartTLS
				item.SMTPUsername = cfg.Username
				item.SMTPPassword = cfg.Password
			case *models.GRPCHealthCheckConfig:
				item.Host = cfg.Host
				item.Timeout = cfg.Timeout
				item.UseTLS = cfg.UseTLS
			case *models.WebSocketCheckConfig:
				item.URL = cfg.URL
				item.Timeout = cfg.Timeout
			}
		}

		payload.Checks = append(payload.Checks, item)
	}

	// Return as YAML or JSON based on Accept header
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/x-yaml") || strings.Contains(accept, "text/yaml") {
		yamlBytes, err := yaml.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal YAML"})
			return
		}
		c.Data(http.StatusOK, "application/x-yaml", yamlBytes)
		return
	}

	c.JSON(http.StatusOK, payload)
}
