package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/imcitius/checker/pkg/checks"
	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// ListCheckDefinitions returns all check definitions
func ListCheckDefinitions(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	defs, err := repo.GetAllCheckDefinitions(ctx)
	if err != nil {
		logrus.Errorf("Failed to get check definitions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch check definitions",
		})
		return
	}

	// Convert to view models for the API
	viewModels := make([]models.CheckDefinitionViewModel, 0, len(defs))
	for _, def := range defs {
		viewModels = append(viewModels, convertToCheckDefViewModel(def))
	}

	c.JSON(http.StatusOK, viewModels)
}

// GetCheckDefinition returns a single check definition by UUID
func GetCheckDefinition(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	def, err := repo.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		logrus.Errorf("Failed to get check definition %s: %v", uuid, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Check definition not found",
		})
		return
	}

	c.JSON(http.StatusOK, convertToCheckDefViewModel(def))
}

// GetCheckRegionResults returns the latest per-region check results for multi-region consensus.
func GetCheckRegionResults(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UUID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	results, err := repo.GetLatestRegionResults(ctx, uuid)
	if err != nil {
		logrus.Errorf("Failed to get region results for %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get region results"})
		return
	}

	type regionResultView struct {
		Region    string `json:"region"`
		IsHealthy bool   `json:"is_healthy"`
		Message   string `json:"message"`
		CreatedAt string `json:"created_at"`
	}

	out := make([]regionResultView, 0, len(results))
	for _, r := range results {
		out = append(out, regionResultView{
			Region:    r.Region,
			IsHealthy: r.IsHealthy,
			Message:   r.Message,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, out)
}

// CreateCheckDefinition creates a new check definition
func CreateCheckDefinition(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var def models.CheckDefinition
	var vm models.CheckDefinitionViewModel
	if err := c.ShouldBindJSON(&vm); err != nil {
		logrus.Errorf("Failed to bind check definition: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid check definition data",
		})
		return
	}

	// Validate
	if vm.Name == "" || vm.Project == "" || vm.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name, project, and type are required fields",
		})
		return
	}

	// Convert VM -> Domain Model
	def = convertFromCheckDefViewModel(vm)

	// Generate UUID if not provided (required for PostgreSQL uuid column)
	if def.UUID == "" {
		def.UUID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}
	def.UpdatedAt = now

	// Default to enabled if not specified
	if !def.Enabled {
		def.Enabled = true
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Apply checker-wide defaults for fields not explicitly set
	if defaults, err := repo.GetCheckDefaults(ctx); err == nil {
		if def.RetryCount == 0 && def.RetryInterval == "" && defaults.RetryCount > 0 {
			def.RetryCount = defaults.RetryCount
			def.RetryInterval = defaults.RetryInterval
		}
		if def.Duration == "" && defaults.CheckInterval != "" {
			def.Duration = defaults.CheckInterval
		}
		if def.ReAlertInterval == "" && defaults.ReAlertInterval != "" {
			def.ReAlertInterval = defaults.ReAlertInterval
		}
		if def.Severity == "" && defaults.Severity != "" {
			def.Severity = defaults.Severity
		}
		if len(def.AlertChannels) == 0 && len(defaults.AlertChannels) > 0 {
			def.AlertChannels = defaults.AlertChannels
		}
		if def.EscalationPolicyName == "" && defaults.EscalationPolicy != "" {
			def.EscalationPolicyName = defaults.EscalationPolicy
		}
	}

	id, err := repo.CreateCheckDefinition(ctx, def)
	if err != nil {
		logrus.Errorf("Failed to create check definition: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create check definition",
		})
		return
	}

	if idObj, err := primitive.ObjectIDFromHex(id); err == nil {
		def.ID = idObj
	} else {
		// Since we switched to Postgres, ID might be UUID already in CreateCheckDefinition return
		// but primitive.ObjectIDFromHex is Mongo specific.
		// We should just use the returned 'id' assuming it is the UUID or something useful.
		// Actually CreateCheckDefinition returns UUID as string in our implementation.
		def.UUID = id
	}

	c.JSON(http.StatusCreated, convertToCheckDefViewModel(def))
}

// UpdateCheckDefinition updates an existing check definition
func UpdateCheckDefinition(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	var vm models.CheckDefinitionViewModel
	if err := c.ShouldBindJSON(&vm); err != nil {
		logrus.Errorf("Failed to bind check definition: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid check definition data",
		})
		return
	}

	if vm.Name == "" || vm.Project == "" || vm.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name, project, and type are required fields",
		})
		return
	}

	def := convertFromCheckDefViewModel(vm)

	// Ensure UUID in URL matches body
	def.UUID = uuid

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get the existing definition to maintain ID
	existingDef, err := repo.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		logrus.Errorf("Failed to get check definition %s: %v", uuid, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Check definition not found",
		})
		return
	}

	// Preserve the ID and status fields that are not part of the edit view model
	def.ID = existingDef.ID
	def.CreatedAt = existingDef.CreatedAt
	def.LastRun = existingDef.LastRun
	def.IsHealthy = existingDef.IsHealthy
	def.LastMessage = existingDef.LastMessage
	def.LastAlertSent = existingDef.LastAlertSent
	def.MaintenanceUntil = existingDef.MaintenanceUntil

	// Update the definition
	if err := repo.UpdateCheckDefinition(ctx, def); err != nil {
		logrus.Errorf("Failed to update check definition %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update check definition",
		})
		return
	}

	c.JSON(http.StatusOK, convertToCheckDefViewModel(def))
}

// DeleteCheckDefinition deletes a check definition
func DeleteCheckDefinition(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := repo.DeleteCheckDefinition(ctx, uuid); err != nil {
		logrus.Errorf("Failed to delete check definition %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete check definition",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Check definition %s deleted", uuid),
		"uuid":    uuid,
	})
}

// ToggleCheckDefinitionStatus enables or disables a check definition.
// If the "enabled" query parameter is provided, the check is set to that value.
// Otherwise, the current enabled state is flipped (true toggle).
func ToggleCheckDefinitionStatus(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var enabled bool
	if enabledParam := c.Query("enabled"); enabledParam != "" {
		// Explicit enabled value provided — use it
		enabled = enabledParam == "true"
	} else {
		// No explicit value — fetch current state and flip it
		def, err := repo.GetCheckDefinitionByUUID(ctx, uuid)
		if err != nil {
			logrus.Errorf("Failed to get check definition %s for toggle: %v", uuid, err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Check definition not found",
			})
			return
		}
		enabled = !def.Enabled
	}

	if err := repo.ToggleCheckDefinition(ctx, uuid, enabled); err != nil {
		logrus.Errorf("Failed to toggle check definition %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to toggle check definition",
		})
		return
	}

	// Get the updated check definition
	def, err := repo.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		logrus.Warnf("Check definition %s toggled but could not be retrieved: %v", uuid, err)
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Check definition %s toggled to %v", uuid, enabled),
			"uuid":    uuid,
			"enabled": enabled,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Check definition %s toggled to %v", uuid, enabled),
		"uuid":    uuid,
		"enabled": enabled,
		"check":   convertToCheckDefViewModel(def),
	})
}

// Get all projects for check definitions
func GetCheckProjects(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	projects, err := repo.GetAllProjects(ctx)
	if err != nil {
		logrus.Errorf("Failed to get projects: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch projects",
		})
		return
	}

	if projects == nil {
		projects = []string{}
	}
	c.JSON(http.StatusOK, projects)
}

// Get all check types
func GetCheckTypes(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	types, err := repo.GetAllCheckTypes(ctx)
	if err != nil {
		logrus.Errorf("Failed to get check types: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch check types",
		})
		return
	}

	if types == nil {
		types = []string{}
	}
	c.JSON(http.StatusOK, types)
}

// GetDefaultTimeouts returns default timeouts for all check types
func GetDefaultTimeouts(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	defaultTimeouts := repo.GetAllDefaultTimeouts()
	c.JSON(http.StatusOK, defaultTimeouts)
}

// Helper to convert a CheckDefinition to a CheckDefinitionViewModel
func convertToCheckDefViewModel(def models.CheckDefinition) models.CheckDefinitionViewModel {
	vm := models.CheckDefinitionViewModel{
		ID:               def.ID.Hex(),
		UUID:             def.UUID,
		Name:             def.Name,
		Project:          def.Project,
		GroupName:        def.GroupName,
		Type:             def.Type,
		Description:      def.Description,
		Enabled:          def.Enabled,
		CreatedAt:        def.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        def.UpdatedAt.Format(time.RFC3339),
		Duration:         def.Duration,
		ReAlertInterval:  def.ReAlertInterval,
		ActorType:            def.ActorType,
		RetryCount:           def.RetryCount,
		RetryInterval:       def.RetryInterval,
		EscalationPolicyName: def.EscalationPolicyName,
		AlertChannels:        def.AlertChannels,
		Severity:             def.Severity,
	}

	// Set maintenance window
	if def.MaintenanceUntil != nil {
		formatted := def.MaintenanceUntil.Format(time.RFC3339)
		vm.MaintenanceUntil = &formatted
	}

	// Populate config fields
	if def.Config != nil {
		switch c := def.Config.(type) {
		case *models.HTTPCheckConfig:
			vm.URL = c.URL
			vm.Timeout = c.Timeout
			vm.Answer = c.Answer
			vm.AnswerPresent = c.AnswerPresent
			vm.Code = c.Code
			vm.Headers = c.Headers
			vm.Cookies = c.Cookies
			vm.SkipCheckSSL = c.SkipCheckSSL
			vm.SSLExpirationPeriod = c.SSLExpirationPeriod
			vm.StopFollowRedirects = c.StopFollowRedirects
			vm.Auth.User = c.Auth.User
			vm.Auth.Password = c.Auth.Password
		case *models.TCPCheckConfig:
			vm.Host = c.Host
			vm.Port = c.Port
			vm.Timeout = c.Timeout
		case *models.ICMPCheckConfig:
			vm.Host = c.Host
			vm.Count = c.Count
			vm.Timeout = c.Timeout
		case *models.MySQLCheckConfig:
			vm.Host = c.Host
			vm.Port = c.Port
			vm.Timeout = c.Timeout
			vm.MySQL.UserName = c.UserName
			vm.MySQL.Password = c.Password
			vm.MySQL.DBName = c.DBName
			vm.MySQL.Query = c.Query
			vm.MySQL.Response = c.Response
			vm.MySQL.Difference = c.Difference
			vm.MySQL.TableName = c.TableName
			vm.MySQL.Lag = c.Lag
			vm.MySQL.ServerList = c.ServerList
		case *models.PostgreSQLCheckConfig:
			vm.Host = c.Host
			vm.Port = c.Port
			vm.Timeout = c.Timeout
			vm.PgSQL.UserName = c.UserName
			vm.PgSQL.Password = c.Password
			vm.PgSQL.DBName = c.DBName
			vm.PgSQL.SSLMode = c.SSLMode
			vm.PgSQL.Query = c.Query
			vm.PgSQL.Response = c.Response
			vm.PgSQL.Difference = c.Difference
			vm.PgSQL.TableName = c.TableName
			vm.PgSQL.Lag = c.Lag
			vm.PgSQL.ServerList = c.ServerList
			vm.PgSQL.AnalyticReplicas = c.AnalyticReplicas
		case *models.DomainExpiryCheckConfig:
			vm.Domain = c.Domain
			vm.Timeout = c.Timeout
			vm.ExpiryWarningDays = c.ExpiryWarningDays
		case *models.MongoDBCheckConfig:
			vm.MongoDBURI = c.URI
			vm.Timeout = c.Timeout
		case *models.DNSCheckConfig:
			vm.Host = c.Host
			vm.Domain = c.Domain
			vm.RecordType = c.RecordType
			vm.Timeout = c.Timeout
			vm.Expected = c.Expected
		case *models.SSHCheckConfig:
			vm.Host = c.Host
			vm.Port = c.Port
			vm.Timeout = c.Timeout
			vm.ExpectBanner = c.ExpectBanner
		case *models.RedisCheckConfig:
			vm.Host = c.Host
			vm.Port = c.Port
			vm.Timeout = c.Timeout
			vm.RedisPassword = c.Password
			vm.RedisDB = c.DB
		case *models.SSLCertCheckConfig:
			vm.Host = c.Host
			vm.Port = c.Port
			vm.Timeout = c.Timeout
			vm.ExpiryWarningDays = c.ExpiryWarningDays
			vm.ValidateChain = c.ValidateChain
		case *models.SMTPCheckConfig:
			vm.Host = c.Host
			vm.Port = c.Port
			vm.Timeout = c.Timeout
			vm.StartTLS = c.StartTLS
			vm.SMTPUsername = c.Username
			vm.SMTPPassword = c.Password
		case *models.GRPCHealthCheckConfig:
			vm.Host = c.Host
			vm.Timeout = c.Timeout
			vm.UseTLS = c.UseTLS
		case *models.WebSocketCheckConfig:
			vm.URL = c.URL
			vm.Timeout = c.Timeout
		}
	}

	return vm
}

// Helper to convert CheckDefinitionViewModel to CheckDefinition
func convertFromCheckDefViewModel(vm models.CheckDefinitionViewModel) models.CheckDefinition {
	def := models.CheckDefinition{
		UUID:             vm.UUID,
		Name:             vm.Name,
		Project:          vm.Project,
		GroupName:        vm.GroupName,
		Type:             vm.Type,
		Description:      vm.Description,
		Enabled:          vm.Enabled,
		Duration:         vm.Duration,
		ReAlertInterval:  vm.ReAlertInterval,
		ActorType:            vm.ActorType,
		RetryCount:           vm.RetryCount,
		RetryInterval:       vm.RetryInterval,
		EscalationPolicyName: vm.EscalationPolicyName,
		AlertChannels:        vm.AlertChannels,
		Severity:             vm.Severity,
	}

	// Create ID if present (parsed later usually)

	// Populate Config
	switch vm.Type {
	case "http":
		def.Config = &models.HTTPCheckConfig{
			URL:                 vm.URL,
			Timeout:             vm.Timeout,
			Answer:              vm.Answer,
			AnswerPresent:       vm.AnswerPresent,
			Code:                vm.Code,
			Headers:             vm.Headers,
			Cookies:             vm.Cookies,
			SkipCheckSSL:        vm.SkipCheckSSL,
			SSLExpirationPeriod: vm.SSLExpirationPeriod,
			StopFollowRedirects: vm.StopFollowRedirects,
			Auth: models.AuthConfig{
				User:     vm.Auth.User,
				Password: vm.Auth.Password,
			},
		}
	case "tcp":
		def.Config = &models.TCPCheckConfig{
			Host:    vm.Host,
			Port:    vm.Port,
			Timeout: vm.Timeout,
		}
	case "icmp":
		def.Config = &models.ICMPCheckConfig{
			Host:    vm.Host,
			Count:   vm.Count,
			Timeout: vm.Timeout,
		}
	case "mysql_query", "mysql_query_unixtime", "mysql_replication":
		def.Config = &models.MySQLCheckConfig{
			Host:       vm.Host,
			Port:       vm.Port,
			Timeout:    vm.Timeout,
			UserName:   vm.MySQL.UserName,
			Password:   vm.MySQL.Password,
			DBName:     vm.MySQL.DBName,
			Query:      vm.MySQL.Query,
			Response:   vm.MySQL.Response,
			Difference: vm.MySQL.Difference,
			TableName:  vm.MySQL.TableName,
			Lag:        vm.MySQL.Lag,
			ServerList: vm.MySQL.ServerList,
		}
	case "mongodb":
		def.Config = &models.MongoDBCheckConfig{
			URI:     vm.MongoDBURI,
			Timeout: vm.Timeout,
		}
	case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
		def.Config = &models.PostgreSQLCheckConfig{
			Host:             vm.Host,
			Port:             vm.Port,
			Timeout:          vm.Timeout,
			UserName:         vm.PgSQL.UserName,
			Password:         vm.PgSQL.Password,
			DBName:           vm.PgSQL.DBName,
			SSLMode:          vm.PgSQL.SSLMode,
			Query:            vm.PgSQL.Query,
			Response:         vm.PgSQL.Response,
			Difference:       vm.PgSQL.Difference,
			TableName:        vm.PgSQL.TableName,
			Lag:              vm.PgSQL.Lag,
			ServerList:       vm.PgSQL.ServerList,
			AnalyticReplicas: vm.PgSQL.AnalyticReplicas,
		}
	case "domain_expiry":
		def.Config = &models.DomainExpiryCheckConfig{
			Domain:            vm.Domain,
			Timeout:           vm.Timeout,
			ExpiryWarningDays: vm.ExpiryWarningDays,
		}
	case "dns":
		def.Config = &models.DNSCheckConfig{
			Host:       vm.Host,
			Domain:     vm.Domain,
			RecordType: vm.RecordType,
			Timeout:    vm.Timeout,
			Expected:   vm.Expected,
		}
	case "ssh":
		def.Config = &models.SSHCheckConfig{
			Host:         vm.Host,
			Port:         vm.Port,
			Timeout:      vm.Timeout,
			ExpectBanner: vm.ExpectBanner,
		}
	case "redis":
		def.Config = &models.RedisCheckConfig{
			Host:     vm.Host,
			Port:     vm.Port,
			Timeout:  vm.Timeout,
			Password: vm.RedisPassword,
			DB:       vm.RedisDB,
		}
	case "ssl_cert":
		def.Config = &models.SSLCertCheckConfig{
			Host:              vm.Host,
			Port:              vm.Port,
			Timeout:           vm.Timeout,
			ExpiryWarningDays: vm.ExpiryWarningDays,
			ValidateChain:     vm.ValidateChain,
		}
	case "smtp":
		def.Config = &models.SMTPCheckConfig{
			Host:     vm.Host,
			Port:     vm.Port,
			Timeout:  vm.Timeout,
			StartTLS: vm.StartTLS,
			Username: vm.SMTPUsername,
			Password: vm.SMTPPassword,
		}
	case "grpc_health":
		def.Config = &models.GRPCHealthCheckConfig{
			Host:    vm.Host,
			Timeout: vm.Timeout,
			UseTLS:  vm.UseTLS,
		}
	case "websocket":
		def.Config = &models.WebSocketCheckConfig{
			URL:     vm.URL,
			Timeout: vm.Timeout,
		}
	}

	return def
}

// TestCheckDefinition executes a check definition without saving it (dry-run).
// POST /api/check-definitions/test
func TestCheckDefinition(c *gin.Context) {
	var vm models.CheckDefinitionViewModel
	if err := c.ShouldBindJSON(&vm); err != nil {
		logrus.Errorf("Failed to bind check definition for test: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid check definition data",
		})
		return
	}

	// Validate that check type is set
	if vm.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Check type is required",
		})
		return
	}

	// Convert VM -> Domain Model
	def := convertFromCheckDefViewModel(vm)

	// Create the checker via the factory
	logger := logrus.WithField("handler", "TestCheckDefinition")
	checker := checks.CheckerFactory(def, logger)
	if checker == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("Unsupported or misconfigured check type: %s", vm.Type),
		})
		return
	}

	// Execute with a 30-second hard timeout
	type checkResult struct {
		duration time.Duration
		err      error
	}
	resultCh := make(chan checkResult, 1)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	go func() {
		d, err := checker.Run()
		resultCh <- checkResult{duration: d, err: err}
	}()

	select {
	case res := <-resultCh:
		durationMs := res.duration.Milliseconds()
		if res.err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success":     false,
				"duration_ms": durationMs,
				"message":     res.err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success":     true,
				"duration_ms": durationMs,
				"message":     "OK",
			})
		}
	case <-ctx.Done():
		c.JSON(http.StatusOK, gin.H{
			"success":     false,
			"duration_ms": 30000,
			"message":     "Check timed out after 30 seconds",
		})
	}
}

// SetMaintenanceWindow sets a maintenance window for a check definition.
// PUT /api/check-definitions/:uuid/maintenance
// Body: {"until": "2024-03-15T15:00:00Z"}
func SetMaintenanceWindow(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	checkUUID := c.Param("uuid")

	if checkUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UUID is required"})
		return
	}

	var body struct {
		Until string `json:"until"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Until == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'until' is required (RFC3339 datetime)"})
		return
	}

	until, err := time.Parse(time.RFC3339, body.Until)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid datetime format, use RFC3339 (e.g. 2024-03-15T15:00:00Z)"})
		return
	}

	if until.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maintenance window must be in the future"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := repo.SetMaintenanceWindow(ctx, checkUUID, &until); err != nil {
		logrus.Errorf("Failed to set maintenance window for %s: %v", checkUUID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set maintenance window"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           fmt.Sprintf("Maintenance window set for check %s until %s", checkUUID, until.Format(time.RFC3339)),
		"uuid":              checkUUID,
		"maintenance_until": until.Format(time.RFC3339),
	})
}

// ClearMaintenanceWindow clears the maintenance window for a check definition.
// DELETE /api/check-definitions/:uuid/maintenance
func ClearMaintenanceWindow(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	checkUUID := c.Param("uuid")

	if checkUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UUID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := repo.SetMaintenanceWindow(ctx, checkUUID, nil); err != nil {
		logrus.Errorf("Failed to clear maintenance window for %s: %v", checkUUID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear maintenance window"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Maintenance window cleared for check %s", checkUUID),
		"uuid":    checkUUID,
	})
}
