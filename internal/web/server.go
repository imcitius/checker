package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"checker/internal/auth"
	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"
	"checker/internal/slack"
)

var (
	// WebSocket upgrader with more explicit configuration
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all connections for development
		},
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 10 * time.Second,
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			logrus.Errorf("WebSocket upgrade error: %v, status: %d", reason, status)
		},
	}

	// Manage connected WebSocket clients with a mutex for thread safety
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.Mutex
)

// BroadcastCheckUpdate sends check status updates to all connected WebSocket clients
func BroadcastCheckUpdate(checkStatus models.CheckViewModel) {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	// If no clients, skip broadcasting
	if len(clients) == 0 {
		return
	}

	disconnectedClients := []*websocket.Conn{}

	for client := range clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":  "update",
			"check": checkStatus,
		})
		if err != nil {
			// Client likely disconnected, mark for removal
			disconnectedClients = append(disconnectedClients, client)
			logrus.Debugf("Failed to send update to WebSocket client: %v", err)
		}
	}

	// Remove disconnected clients
	for _, client := range disconnectedClients {
		delete(clients, client)
		client.Close()
	}

	if len(disconnectedClients) > 0 {
		logrus.Debugf("Removed %d disconnected WebSocket clients", len(disconnectedClients))
	}
}

// BroadcastChecksUpdate sends all check statuses to all connected WebSocket clients
func BroadcastChecksUpdate(repo db.Repository) {
	// If no clients, skip fetching data and broadcasting
	clientsMux.Lock()
	clientCount := len(clients)
	clientsMux.Unlock()

	if clientCount == 0 {
		return
	}

	checks, err := getAllCheckStatuses(repo)
	if err != nil {
		logrus.Errorf("Failed to get check statuses for broadcast: %v", err)
		return
	}

	viewModels := convertToViewModels(checks)

	clientsMux.Lock()
	defer clientsMux.Unlock()

	// Check again if we still have clients after the data fetch
	if len(clients) == 0 {
		return
	}

	// logrus.Debugf("Broadcasting check updates to %d clients", len(clients))
	disconnectedClients := []*websocket.Conn{}

	for client := range clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":   "checks",
			"checks": viewModels,
		})
		if err != nil {
			// Client likely disconnected, mark for removal
			disconnectedClients = append(disconnectedClients, client)
			logrus.Debugf("Failed to send checks update to WebSocket client: %v", err)
		}
	}

	// Remove disconnected clients
	for _, client := range disconnectedClients {
		delete(clients, client)
		client.Close()
	}

	if len(disconnectedClients) > 0 {
		logrus.Debugf("Removed %d disconnected WebSocket clients", len(disconnectedClients))
	}
}

// RunServer starts the web server and returns an error if it fails
func RunServer(ctx context.Context, cfg *config.Config, repo db.Repository, slackClient *slack.SlackClient, authMgr *auth.AuthManager) error {
	// Create a router with default middleware
	router := gin.Default()

	// Add Repository to context
	router.Use(func(c *gin.Context) {
		c.Set("repo", repo)
		c.Next()
	})

	// Load embedded HTML templates (strip "templates/" prefix so names match c.HTML calls)
	subTemplates, err := fs.Sub(templateFS, "templates")
	if err != nil {
		return fmt.Errorf("failed to access embedded templates: %w", err)
	}
	tmpl := template.Must(template.New("").ParseFS(subTemplates, "*.html"))
	router.SetHTMLTemplate(tmpl)

	// Serve embedded static files (public — CSS/JS must load for login redirects)
	subStatic, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("failed to access embedded static files: %w", err)
	}
	router.StaticFS("/static", http.FS(subStatic))

	// Serve the React SPA from embedded frontend/dist (built by Vite)
	spaRoot, spaErr := fs.Sub(spaFS, "spa")
	if spaErr != nil {
		logrus.Warnf("SPA frontend not embedded (run 'make build-frontend' first): %v", spaErr)
	}
	var spaHandler http.Handler
	if spaErr == nil {
		spaHandler = http.FileServer(http.FS(spaRoot))
	}

	// Health check endpoint (public — used by Railway/load balancers)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public auth routes (no auth required)
	router.GET("/auth/login", authMgr.HandleLogin)
	router.GET("/auth/callback", authMgr.HandleCallback)
	router.GET("/auth/logout", authMgr.HandleLogout)

	// Slack routes (exempt from OIDC — they have their own signature verification)
	if slackClient != nil {
		handler := NewSlackInteractiveHandler(slackClient.SigningSecret(), slackClient, repo)
		router.POST("/api/slack/interactive", gin.WrapF(handler.HandleInteraction))
		logrus.Info("Slack interactive endpoint registered at /api/slack/interactive")
		router.POST("/api/slack/commands", gin.WrapF(handler.HandleSlashCommand))
		logrus.Info("Slack slash command endpoint registered at /api/slack/commands")
	}

	// Protected routes (OIDC cookie or API key required)
	protected := router.Group("/")
	protected.Use(authMgr.Middleware())

	// Serve SPA for dashboard routes (if frontend is built), fall back to legacy templates
	if spaHandler != nil {
		protected.GET("/", serveSPA(spaRoot))
		protected.GET("/manage", serveSPA(spaRoot))
	} else {
		// Legacy template-based routes (fallback when SPA not built)
		protected.GET("/", handleDashboard)
		protected.GET("/check-definitions", func(c *gin.Context) {
			c.HTML(http.StatusOK, "check_management.html", gin.H{
				"user_email": c.GetString("user_email"),
				"user_name":  c.GetString("user_name"),
			})
		})
	}

	// Serve SPA static assets (JS, CSS, etc.)
	if spaHandler != nil {
		router.GET("/assets/*filepath", func(c *gin.Context) {
			spaHandler.ServeHTTP(c.Writer, c.Request)
		})
	}

	// WebSocket endpoint
	protected.GET("/ws", func(c *gin.Context) {
		handleWebSocket(c, repo)
	})

	// REST API routes
	checkDefinitionsGroup := protected.Group("/api/check-definitions")
	{
		checkDefinitionsGroup.GET("", ListCheckDefinitions)
		checkDefinitionsGroup.GET("/:uuid", GetCheckDefinition)
		checkDefinitionsGroup.POST("", CreateCheckDefinition)
		checkDefinitionsGroup.PUT("/:uuid", UpdateCheckDefinition)
		checkDefinitionsGroup.DELETE("/:uuid", DeleteCheckDefinition)
		checkDefinitionsGroup.PATCH("/:uuid/toggle", ToggleCheckDefinitionStatus)
	}

	// Bulk import/export endpoints (separate group to avoid /:uuid conflict)
	bulkGroup := protected.Group("/api/checks")
	{
		bulkGroup.POST("/import", ImportCheckDefinitions)
		bulkGroup.POST("/import/validate", ValidateImportPayload)
		bulkGroup.GET("/export", ExportCheckDefinitions)
	}

	// Metadata endpoints
	metadataGroup := protected.Group("/api/metadata")
	{
		metadataGroup.GET("/projects", GetCheckProjects)
		metadataGroup.GET("/check-types", GetCheckTypes)
		metadataGroup.GET("/default-timeouts", GetDefaultTimeouts)
	}

	// Admin endpoints for migrations (not for production use)
	if gin.Mode() != gin.ReleaseMode {
		protected.POST("/api/admin/migrate-config", func(c *gin.Context) {
			// Convert config file to database entries
			err := repo.ConvertConfigToCheckDefinitions(c.Request.Context(), cfg)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Failed to migrate config: %v", err),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "Configuration migrated to database successfully",
			})
		})
	}

	// Legacy API routes (for backward compatibility)
	protected.GET("/api/checks", func(c *gin.Context) {
		statuses, err := getAllCheckStatuses(repo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		viewModels := convertToViewModels(statuses)
		c.JSON(http.StatusOK, viewModels)
	})

	// Enable/disable a check
	protected.POST("/api/toggle-check", func(c *gin.Context) {
		uuid := c.PostForm("uuid")
		enabled := c.PostForm("enabled") == "true"

		logrus.Infof("Received toggle request for check %s to enabled=%v", uuid, enabled)

		// Try to toggle the check
		if err := toggleCheck(repo, uuid, enabled); err != nil {
			logrus.Errorf("Failed to toggle check %s: %v", uuid, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   err.Error(),
				"uuid":    uuid,
				"enabled": enabled,
				"success": false,
			})
			return
		}

		// Get updated check to broadcast
		check, err := getCheckByUUID(repo, uuid)
		if err != nil {
			if err.Error() == "check definition not found" {
				logrus.Warnf("Check %s still not found after toggle, using placeholder for response", uuid)

				check = models.CheckStatus{
					UUID:        uuid,
					Project:     "Unknown",
					CheckGroup:  "Unknown",
					CheckName:   fmt.Sprintf("Check %s", uuid[:8]),
					CheckType:   "Unknown",
					IsEnabled:   enabled,
					LastRun:     time.Now(),
					IsHealthy:   true,
					Message:     "Newly created check",
					Host:        "",
					Periodicity: "1m",
				}
			} else {
				logrus.Errorf("Failed to get check %s after toggle: %v", uuid, err)
			}
		}

		viewModel := convertToViewModel(check)
		logrus.Infof("Broadcasting check update for %s with enabled=%v", uuid, viewModel.Enabled)

		// Broadcast update to all WebSocket clients
		BroadcastCheckUpdate(viewModel)

		c.JSON(http.StatusOK, gin.H{
			"message": "Check status updated",
			"uuid":    uuid,
			"enabled": viewModel.Enabled,
			"success": true,
			"check":   viewModel,
		})
	})

	// Update last ping status for passive checks
	protected.POST("/api/checks/:uuid/ping", updateLastPingStatus)

	// Debug endpoint for verifying static resources (removed in production)
	if gin.Mode() != gin.ReleaseMode {
		protected.GET("/debug/static", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Static assets are embedded in binary",
				"files":   []string{"styles.css", "script.js"},
				"time":    time.Now().String(),
			})
		})
	}

	// Start a ticker for periodic broadcasts
	broadcastTicker := time.NewTicker(30 * time.Second)

	// Create a context for background goroutines
	bgCtx, cancel := context.WithCancel(ctx)

	// Start the broadcast goroutine
	go func() {
		defer broadcastTicker.Stop()

		for {
			select {
			case <-bgCtx.Done():
				logrus.Info("Broadcast ticker stopping due to context cancellation")
				return
			case <-broadcastTicker.C:
				BroadcastChecksUpdate(repo)
			}
		}
	}()

	// Set up server with graceful shutdown
	srv := &http.Server{
		Addr:    ":8080", // Default port
		Handler: router,
	}

	// Use PORT environment variable if available
	if port := os.Getenv("PORT"); port != "" {
		srv.Addr = ":" + port
	}

	// Channel to signal server shutdown completion
	serverShutdown := make(chan struct{})

	// Run the server in a goroutine
	go func() {
		logrus.Infof("Starting web server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Web server error: %v", err)
		}
		close(serverShutdown)
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Cancel background goroutines
	cancel()

	// Shutdown timeout context
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Close all WebSocket connections
	clientsMux.Lock()
	for client := range clients {
		client.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutting down"),
			time.Now().Add(time.Second),
		)
		client.Close()
		delete(clients, client)
	}
	clientsMux.Unlock()

	// Gracefully shutdown the server
	logrus.Info("Shutting down web server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("Web server shutdown error: %v", err)
		return err
	}

	// Wait for server to finish
	select {
	case <-serverShutdown:
		logrus.Info("Web server shutdown complete")
	case <-shutdownCtx.Done():
		logrus.Warn("Web server shutdown timed out")
		return shutdownCtx.Err()
	}

	return nil
}

func getAllCheckStatuses(repo db.Repository) ([]models.CheckStatus, error) {
	if repo == nil {
		logrus.Error("Repository is nil")
		return nil, fmt.Errorf("repository is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Using GetAllCheckDefinitions which fetches all checks (enabled and disabled)
	// Actually for getAllCheckStatuses used in dashboard, we probably want all checks too.
	definitions, err := repo.GetAllCheckDefinitions(ctx)
	if err != nil {
		logrus.Errorf("Failed to query check definitions: %v", err)
		return nil, err
	}

	// Fetch active silences to mark silenced checks in the UI
	silences, silenceErr := repo.GetActiveSilences(ctx)
	if silenceErr != nil {
		logrus.Errorf("Failed to get active silences: %v", silenceErr)
	}

	// Build lookup sets for silenced checks and projects
	silencedChecks := make(map[string]bool)
	silencedProjects := make(map[string]bool)
	for _, s := range silences {
		switch s.Scope {
		case "check":
			silencedChecks[s.Target] = true
		case "project":
			silencedProjects[s.Target] = true
		}
	}

	// Convert CheckDefinition to CheckStatus
	results := make([]models.CheckStatus, len(definitions))
	for i, def := range definitions {
		host := ""
		url := ""
		if def.Config != nil {
			host = def.Config.GetTarget()
			if httpConf, ok := def.Config.(*models.HTTPCheckConfig); ok {
				url = httpConf.URL
			}
		}

		results[i] = models.CheckStatus{
			ID:            def.ID,
			UUID:          def.UUID,
			Project:       def.Project,
			CheckGroup:    def.GroupName,
			CheckName:     def.Name,
			CheckType:     def.Type,
			LastRun:       def.LastRun,
			IsHealthy:     def.IsHealthy,
			Message:       def.LastMessage,
			IsEnabled:     def.Enabled,
			LastAlertSent: def.LastAlertSent,
			Host:          host,
			Periodicity:   def.Duration,
			URL:           url,
			IsSilenced:    silencedChecks[def.UUID] || silencedProjects[def.Project],
		}
	}

	return results, nil
}

func getCheckByUUID(repo db.Repository, uuid string) (models.CheckStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Debugf("Getting check by UUID: %s", uuid)

	def, err := repo.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		// We can't easily check for "no documents" error type generically without custom error package
		// But for now we just return the error.
		return models.CheckStatus{}, err
	}

	host := ""
	url := ""
	if def.Config != nil {
		host = def.Config.GetTarget()
		if httpConf, ok := def.Config.(*models.HTTPCheckConfig); ok {
			url = httpConf.URL
		}
	}

	// Convert CheckDefinition to CheckStatus
	result := models.CheckStatus{
		ID:            def.ID,
		UUID:          def.UUID,
		Project:       def.Project,
		CheckGroup:    def.GroupName,
		CheckName:     def.Name,
		CheckType:     def.Type,
		LastRun:       def.LastRun,
		IsHealthy:     def.IsHealthy,
		Message:       def.LastMessage,
		IsEnabled:     def.Enabled,
		LastAlertSent: def.LastAlertSent,
		Host:          host,
		Periodicity:   def.Duration,
		URL:           url,
	}

	return result, nil
}

func toggleCheck(repo db.Repository, uuid string, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Infof("Toggling check %s to enabled=%v", uuid, enabled)

	if err := repo.ToggleCheckDefinition(ctx, uuid, enabled); err != nil {
		logrus.Errorf("Error toggling check %s: %v", uuid, err)
		return err
	}

	return nil
}

// serveSPA returns a gin handler that serves the SPA index.html for client-side routing
func serveSPA(spaRoot fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		indexHTML, err := fs.ReadFile(spaRoot, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "SPA index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}
}

func handleDashboard(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	checks, err := getAllCheckStatuses(repo)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	viewModels := convertToViewModels(checks)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"checks":     viewModels,
		"user_email": c.GetString("user_email"),
		"user_name":  c.GetString("user_name"),
	})
}

func formatTimeOrNever(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func convertToViewModel(check models.CheckStatus) models.CheckViewModel {
	return models.CheckViewModel{
		ID:          check.ID.Hex(),
		Name:        check.CheckName,
		Project:     check.Project,
		Healthcheck: check.CheckGroup,
		LastResult:  check.IsHealthy,
		LastExec:    formatTimeOrNever(check.LastRun),
		LastPing:    formatTimeOrNever(check.LastAlertSent),
		Enabled:     check.IsEnabled,
		UUID:        check.UUID,
		CheckType:   check.CheckType,
		Message:     check.Message,
		Host:        check.Host,
		Periodicity: check.Periodicity,
		URL:         check.URL,
		IsSilenced:  check.IsSilenced,
	}
}

func convertToViewModels(checks []models.CheckStatus) []models.CheckViewModel {
	viewModels := make([]models.CheckViewModel, 0, len(checks))
	for _, check := range checks {
		viewModels = append(viewModels, convertToViewModel(check))
	}
	return viewModels
}

// Handle WebSocket connections
func handleWebSocket(c *gin.Context, repo db.Repository) {
	logrus.Debugf("Received WebSocket connection request from: %s", c.Request.RemoteAddr)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Register new client
	clientsMux.Lock()
	clients[conn] = true
	clientCount := len(clients)
	clientsMux.Unlock()

	logrus.Debugf("WebSocket client connected from %s (total clients: %d)",
		c.Request.RemoteAddr, clientCount)

	// Set up ping handler (single one)
	conn.SetPingHandler(func(data string) error {
		err := conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(5*time.Second))
		if err != nil {
			logrus.Debugf("Failed to send pong: %v", err)
		}
		return nil
	})

	// Start a single ping ticker for this connection
	pingTicker := time.NewTicker(15 * time.Second)

	// Send initial data
	err = sendInitialData(conn, repo)
	if err != nil {
		logrus.Errorf("Failed to send initial data: %v", err)
		conn.Close()
		clientsMux.Lock()
		delete(clients, conn)
		clientsMux.Unlock()
		return
	}

	// Handle incoming messages in a goroutine
	go func() {
		defer func() {
			// Stop the ping ticker
			pingTicker.Stop()

			// Close the connection
			conn.Close()

			// Remove from clients map
			clientsMux.Lock()
			delete(clients, conn)
			remainingClients := len(clients)
			clientsMux.Unlock()

			logrus.Debugf("WebSocket client disconnected from %s (remaining clients: %d)",
				c.Request.RemoteAddr, remainingClients)
		}()

		// Channel for done signal from read loop
		done := make(chan struct{})

		// Start a read loop in a separate goroutine
		go func() {
			defer close(done)

			for {
				msgType, msg, err := conn.ReadMessage()
				if err != nil {
					// Normal close or error
					if websocket.IsUnexpectedCloseError(err,
						websocket.CloseGoingAway,
						websocket.CloseNormalClosure,
						websocket.CloseNoStatusReceived) {
						logrus.Debugf("WebSocket read error: %v", err)
					}
					return
				}

				// Process message if it's text
				if msgType == websocket.TextMessage {
					var request map[string]interface{}
					if err := json.Unmarshal(msg, &request); err != nil {
						logrus.Debugf("Failed to parse WebSocket message: %v", err)
						continue
					}

					// Handle different message types
					action, ok := request["action"].(string)
					if !ok {
						continue
					}

					switch action {
					case "getChecks":
						sendInitialData(conn, repo)
					case "ack":
						// Just a keepalive, no need to do anything
					case "toggleCheck":
						// Handle toggle check
						uuid, uuidOk := request["uuid"].(string)
						enabled, enabledOk := request["enabled"].(bool)
						if uuidOk && enabledOk {
							err := toggleCheck(repo, uuid, enabled)
							if err != nil {
								logrus.Errorf("Failed to toggle check: %v", err)
							}
						}
					}
				}
			}
		}()

		// Main event loop
		for {
			select {
			case <-done:
				// Read loop exited
				return
			case <-pingTicker.C:
				// Send ping
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
				if err != nil {
					logrus.Debugf("Failed to send ping, closing connection: %v", err)
					return
				}
			}
		}
	}()
}

// Send initial checks data to a WebSocket client
func sendInitialData(conn *websocket.Conn, repo db.Repository) error {
	checks, err := getAllCheckStatuses(repo)
	if err != nil {
		return fmt.Errorf("failed to get check statuses: %w", err)
	}

	// Convert to view models
	viewModels := convertToViewModels(checks)

	// Create message
	message := map[string]interface{}{
		"type":      "checks",
		"checks":    viewModels,
		"count":     len(viewModels),
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
	}

	// Set a write deadline
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	// Send the message with retry
	var sendErr error
	for attempt := 1; attempt <= 3; attempt++ {
		sendErr = conn.WriteJSON(message)
		if sendErr == nil {
			break
		}

		if attempt < 3 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}
	}

	// Reset write deadline
	conn.SetWriteDeadline(time.Time{})

	return sendErr
}

// http route to upodate LastPing status of Passive Checks
func updateLastPingStatus(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	uuid := c.Param("uuid")
	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UUID parameter is required"})
		return
	}

	logrus.Debugf("Updating last ping for check %s", uuid)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Passive check logic requires fetching the check first to ensure it is passive?
	// Let's use GetCheckDefinitionByUUID first.
	def, err := repo.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Check not found"})
		return
	}

	if def.Type != "passive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Check is not passive"})
		return
	}

	// Update status
	status := models.CheckStatus{
		UUID:          uuid,
		LastRun:       time.Now(),
		IsHealthy:     true,
		LastAlertSent: def.LastAlertSent, // preserve
		Message:       "Passive check ping received",
	}

	if err := repo.UpdateCheckStatus(ctx, status); err != nil {
		logrus.Errorf("Failed to update last ping for %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update check status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pong received"})
}
