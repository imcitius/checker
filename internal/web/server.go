package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"
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
func BroadcastChecksUpdate(mongoDB *db.MongoDB) {
	// If no clients, skip fetching data and broadcasting
	clientsMux.Lock()
	clientCount := len(clients)
	clientsMux.Unlock()

	if clientCount == 0 {
		return
	}

	checks, err := getAllCheckStatuses(mongoDB)
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

	logrus.Debugf("Broadcasting check updates to %d clients", len(clients))
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
func RunServer(ctx context.Context, cfg *config.Config, mongoDB *db.MongoDB) error {
	// Create a router with default middleware
	router := gin.Default()

	// Add MongoDB to context
	router.Use(func(c *gin.Context) {
		c.Set("mongodb", mongoDB)
		c.Next()
	})

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Failed to get working directory: %v", err)
	}
	logrus.Infof("Current working directory: %s", cwd)

	// Create absolute paths for templates
	templatesDir := filepath.Join(cwd, "internal", "web", "templates")
	staticDir := filepath.Join(cwd, "internal", "web", "static")

	// Log filesystem check (once at startup, not per request)
	logrus.Infof("Static directory: %s", staticDir)
	if stat, err := os.Stat(staticDir); err != nil {
		logrus.Errorf("Static directory issue: %v", err)
	} else {
		logrus.Infof("Static directory exists and is a directory: %v", stat.IsDir())

		// List files in the static directory
		files, err := os.ReadDir(staticDir)
		if err != nil {
			logrus.Errorf("Failed to read static directory: %v", err)
		} else {
			logrus.Info("Files in static directory:")
			for _, file := range files {
				logrus.Infof("  - %s", file.Name())
			}
		}
	}

	// Load HTML templates
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	// Serve static files (using Gin's built-in Static method)
	router.Static("/static", staticDir)

	// Main dashboard route
	router.GET("/", handleDashboard)

	// Check definitions management page
	router.GET("/check-definitions", func(c *gin.Context) {
		c.HTML(http.StatusOK, "check_management.html", nil)
	})

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		handleWebSocket(c, mongoDB)
	})

	// REST API routes
	// API for check definitions management
	checkDefinitionsGroup := router.Group("/api/check-definitions")
	{
		checkDefinitionsGroup.GET("", ListCheckDefinitions)
		checkDefinitionsGroup.GET("/:uuid", GetCheckDefinition)
		checkDefinitionsGroup.POST("", CreateCheckDefinition)
		checkDefinitionsGroup.PUT("/:uuid", UpdateCheckDefinition)
		checkDefinitionsGroup.DELETE("/:uuid", DeleteCheckDefinition)
		checkDefinitionsGroup.PATCH("/:uuid/toggle", ToggleCheckDefinitionStatus)
	}

	// Metadata endpoints
	metadataGroup := router.Group("/api/metadata")
	{
		metadataGroup.GET("/projects", GetCheckProjects)
		metadataGroup.GET("/check-types", GetCheckTypes)
	}

	// Admin endpoints for migrations (not for production use)
	if gin.Mode() != gin.ReleaseMode {
		router.POST("/api/admin/migrate-config", func(c *gin.Context) {
			// Convert config file to database entries
			err := mongoDB.ConvertConfigToCheckDefinitions(c.Request.Context(), cfg)
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
	router.GET("/api/checks", func(c *gin.Context) {
		statuses, err := getAllCheckStatuses(mongoDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		viewModels := convertToViewModels(statuses)
		c.JSON(http.StatusOK, viewModels)
	})

	// Enable/disable a check
	router.POST("/api/toggle-check", func(c *gin.Context) {
		uuid := c.PostForm("uuid")
		enabled := c.PostForm("enabled") == "true"

		logrus.Infof("Received toggle request for check %s to enabled=%v", uuid, enabled)

		// Try to toggle the check
		if err := toggleCheck(mongoDB, uuid, enabled); err != nil {
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
		check, err := getCheckByUUID(mongoDB, uuid)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// This shouldn't happen since toggleCheck should create the check if it doesn't exist
				// But just in case, create a placeholder check for the response
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
				// Continue anyway to return success to the client
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
	router.POST("/api/checks/:uuid/ping", updateLastPingStatus)

	// Debug endpoint for verifying static resources (removed in production)
	if gin.Mode() != gin.ReleaseMode {
		router.GET("/debug/static", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message":     "Static debug endpoint",
				"static_path": staticDir,
				"files":       []string{"styles.css", "script.js"},
				"time":        time.Now().String(),
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
				BroadcastChecksUpdate(mongoDB)
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

func getAllCheckStatuses(mongoDB *db.MongoDB) ([]models.CheckStatus, error) {
	logrus.Debug("Getting all check statuses from MongoDB")

	if mongoDB == nil {
		logrus.Error("MongoDB connection is nil")
		return nil, fmt.Errorf("database connection is nil")
	}

	if mongoDB.Database == nil {
		logrus.Error("MongoDB database is nil")
		return nil, fmt.Errorf("database is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := mongoDB.Database.Collection("check_definitions").Find(ctx, bson.M{})
	if err != nil {
		logrus.Errorf("Failed to query check_definitions collection: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.CheckStatus
	if err := cursor.All(ctx, &results); err != nil {
		logrus.Errorf("Failed to decode check statuses: %v", err)
		return nil, err
	}

	logrus.Debugf("Retrieved %d check statuses from database", len(results))

	return results, nil
}

func getCheckByUUID(mongoDB *db.MongoDB, uuid string) (models.CheckStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Debugf("Getting check by UUID: %s", uuid)

	var result models.CheckStatus
	err := mongoDB.Database.Collection("check_definitions").FindOne(ctx, bson.M{"uuid": uuid}).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			logrus.Warnf("Check %s not found in database, returning error", uuid)
			return result, err
		}

		logrus.Errorf("Error getting check by UUID %s: %v", uuid, err)
		return result, err
	}

	return result, nil
}

func toggleCheck(mongoDB *db.MongoDB, uuid string, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Infof("Toggling check %s to enabled=%v", uuid, enabled)

	filter := bson.M{
		"uuid": uuid,
	}

	update := bson.M{
		"$set": bson.M{
			"enabled":    enabled,
			"updated_at": time.Now(),
		},
	}

	result, err := mongoDB.Collection("check_definitions").UpdateOne(ctx, filter, update)
	if err != nil {
		logrus.Errorf("Error toggling check %s: %v", uuid, err)
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("check definition with UUID %s not found", uuid)
	}

	logrus.Infof("Successfully toggled check %s to enabled=%v (matched: %d, modified: %d)",
		uuid, enabled, result.MatchedCount, result.ModifiedCount)

	return nil
}

func handleDashboard(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)

	checks, err := getAllCheckStatuses(mongoDB)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	viewModels := convertToViewModels(checks)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"checks": viewModels,
	})
}

func convertToViewModel(check models.CheckStatus) models.CheckViewModel {
	return models.CheckViewModel{
		ID:          check.ID.Hex(),
		Name:        check.CheckName,
		Project:     check.Project,
		Healthcheck: check.CheckGroup,
		LastResult:  check.IsHealthy,
		LastExec:    check.LastRun.Format("2006-01-02 15:04:05"),
		LastPing:    check.LastAlertSent.Format("2006-01-02 15:04:05"),
		Enabled:     check.IsEnabled,
		UUID:        check.UUID,
		CheckType:   check.CheckType,
		Message:     check.Message,
		Host:        check.Host,
		Periodicity: check.Periodicity,
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
func handleWebSocket(c *gin.Context, mongoDB *db.MongoDB) {
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
	err = sendInitialData(conn, mongoDB)
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
						sendInitialData(conn, mongoDB)
					case "ack":
						// Just a keepalive, no need to do anything
					case "toggleCheck":
						// Handle toggle check
						uuid, uuidOk := request["uuid"].(string)
						enabled, enabledOk := request["enabled"].(bool)
						if uuidOk && enabledOk {
							err := toggleCheck(mongoDB, uuid, enabled)
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
func sendInitialData(conn *websocket.Conn, mongoDB *db.MongoDB) error {
	checks, err := getAllCheckStatuses(mongoDB)
	if err != nil {
		return fmt.Errorf("failed to get check statuses: %w", err)
	}

	// // If no data, add some fake checks for development
	// if len(checks) == 0 && gin.Mode() != gin.ReleaseMode {
	// 	logrus.Warn("No check statuses found in database. Adding test data.")

	// 	// Add test data for development only
	// 	checks = append(checks, models.CheckStatus{
	// 		Project:     "Test Project",
	// 		CheckName:   "Test Check 1",
	// 		CheckType:   "HTTP",
	// 		IsHealthy:   true,
	// 		IsEnabled:   true,
	// 		LastRun:     time.Now(),
	// 		UUID:        "test-uuid-1",
	// 		Host:        "example.com",
	// 		Periodicity: "1m",
	// 	})

	// 	checks = append(checks, models.CheckStatus{
	// 		Project:     "Test Project",
	// 		CheckName:   "Test Check 2",
	// 		CheckType:   "TCP",
	// 		IsHealthy:   false,
	// 		IsEnabled:   true,
	// 		LastRun:     time.Now(),
	// 		UUID:        "test-uuid-2",
	// 		Message:     "Connection refused",
	// 		Host:        "example.org:8080",
	// 		Periodicity: "5m",
	// 	})
	// }

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
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)

	uuid := c.Param("uuid")
	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UUID parameter is required"})
		return
	}

	logrus.Debugf("Updating last ping for check %s", uuid)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"uuid": uuid,
		"type": "passive", // Only allow updating passive checks
	}

	update := bson.M{
		"$set": bson.M{
			"last_run":   time.Now(),
			"is_healthy": true,
		},
	}

	result, err := mongoDB.Collection("check_definitions").UpdateOne(ctx, filter, update)
	if err != nil {
		logrus.Errorf("Error updating last ping for check %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("passive check with UUID %s not found", uuid)})
		return
	}

	logrus.Infof("Successfully updated last ping for check %s (matched: %d, modified: %d)",
		uuid, result.MatchedCount, result.ModifiedCount)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
