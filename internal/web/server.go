package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// Manage connected WebSocket clients
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.Mutex
)

// BroadcastCheckUpdate sends check status updates to all connected WebSocket clients
func BroadcastCheckUpdate(checkStatus models.CheckViewModel) {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	for client := range clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":  "update",
			"check": checkStatus,
		})
		if err != nil {
			// Client likely disconnected, remove it
			client.Close()
			delete(clients, client)
			logrus.Debugf("Removed disconnected WebSocket client: %v", err)
		}
	}
}

// BroadcastChecksUpdate sends all check statuses to all connected WebSocket clients
func BroadcastChecksUpdate(mongoDB *db.MongoDB) {
	checks, err := getAllCheckStatuses(mongoDB)
	if err != nil {
		logrus.Errorf("Failed to get check statuses for broadcast: %v", err)
		return
	}

	viewModels := convertToViewModels(checks)

	clientsMux.Lock()
	defer clientsMux.Unlock()

	// Only broadcast if we have active clients
	if len(clients) == 0 {
		return
	}
	logrus.Debugf("Broadcasting check updates to %d clients", len(clients))

	for client := range clients {
		err := client.WriteJSON(map[string]interface{}{
			"type":   "checks",
			"checks": viewModels,
		})
		if err != nil {
			// Client likely disconnected, remove it
			client.Close()
			delete(clients, client)
			logrus.Debugf("Removed disconnected WebSocket client: %v", err)
		}
	}
}

func RunServer(cfg *config.Config, mongoDB *db.MongoDB) error {
	router := gin.Default()

	// Add MongoDB to context
	router.Use(func(c *gin.Context) {
		c.Set("mongodb", mongoDB)
		c.Next()
	})

	// Template loading handled with LoadHTMLGlob above
	// router.SetHTMLTemplate(template.Must(template.ParseFiles("internal/web/templates/dashboard.html")))

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Failed to get working directory: %v", err)
	}
	logrus.Infof("Current working directory: %s", cwd)

	// Create absolute paths for static files and templates
	staticDir := filepath.Join(cwd, "internal", "web", "static")
	templatesDir := filepath.Join(cwd, "internal", "web", "templates")

	// Log filesystem check
	logrus.Infof("Checking if static directory exists: %s", staticDir)
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

	// Use absolute path for templates
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	// Serve static files with verbose logging for debugging
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			logrus.Infof("Static file request: %s", c.Request.URL.Path)
		}
		c.Next()
	})

	// Handle static files with custom handler for more control
	router.GET("/static/*filepath", func(c *gin.Context) {
		relPath := c.Param("filepath")
		logrus.Infof("Serving static file: %s", relPath)

		// Construct the full file path
		filePath := filepath.Join(staticDir, relPath)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			logrus.Errorf("Static file not found: %s", filePath)
			c.String(http.StatusNotFound, "File not found: "+relPath)
			return
		}

		// Set appropriate Content-Type based on file extension
		extension := filepath.Ext(relPath)
		switch extension {
		case ".css":
			c.Header("Content-Type", "text/css")
		case ".js":
			c.Header("Content-Type", "application/javascript")
		case ".html":
			c.Header("Content-Type", "text/html")
		case ".png":
			c.Header("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			c.Header("Content-Type", "image/jpeg")
		case ".svg":
			c.Header("Content-Type", "image/svg+xml")
		}

		// Set cache control for all static assets
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")

		// Serve the file
		c.File(filePath)
	})

	// Direct file endpoint for emergencies
	router.GET("/direct-file/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		filePath := filepath.Join(staticDir, filename)

		logrus.Infof("Direct file request for: %s (full path: %s)", filename, filePath)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			logrus.Errorf("File not found: %s", filePath)
			c.String(http.StatusNotFound, "File not found")
			return
		}

		c.File(filePath)
	})

	// WebSocket endpoint with enhanced logging and debugging
	router.GET("/ws", func(c *gin.Context) {
		// Detailed connection logging
		logrus.Infof("WebSocket connection attempt from %s | Protocol: %s | Origin: %s",
			c.ClientIP(), c.GetHeader("Sec-WebSocket-Protocol"), c.GetHeader("Origin"))

		// More comprehensive header logging
		logrus.Debug("WebSocket request headers:")
		for name, values := range c.Request.Header {
			logrus.Debugf("  %s: %v", name, values)
		}

		// Set CORS headers for WebSocket handshake
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "content-type, content-length, accept-encoding, x-csrf-token, authorization, accept, origin, cache-control, x-requested-with")

		// Check all necessary conditions for a successful WebSocket upgrade
		if c.Request.Method != "GET" {
			logrus.Errorf("WebSocket connection attempt with non-GET method: %s", c.Request.Method)
			c.String(http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		if c.GetHeader("Connection") == "" || !strings.Contains(strings.ToLower(c.GetHeader("Connection")), "upgrade") {
			logrus.Errorf("WebSocket connection attempt missing 'Connection: Upgrade' header")
			// Continue anyway, the WebSocket library will handle this
		}

		if c.GetHeader("Upgrade") == "" || strings.ToLower(c.GetHeader("Upgrade")) != "websocket" {
			logrus.Errorf("WebSocket connection attempt missing 'Upgrade: websocket' header")
			// Continue anyway, the WebSocket library will handle this
		}

		// Retrieve MongoDB from context - with error handling
		mongoDBValue, exists := c.Get("mongodb")
		if !exists {
			logrus.Error("MongoDB not found in context")
			c.String(http.StatusInternalServerError, "Internal Server Error")
			return
		}

		mongoDB, ok := mongoDBValue.(*db.MongoDB)
		if !ok {
			logrus.Error("MongoDB context value is not of the expected type")
			c.String(http.StatusInternalServerError, "Internal Server Error")
			return
		}

		logrus.Info("WebSocket upgrading connection - all checks passed")
		handleWebSocket(c, mongoDB)
	})

	// New Web UI routes
	router.GET("/", handleDashboard)

	// REST API routes
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

	// Debug endpoint for verifying static resources
	router.GET("/debug/static", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":     "Static debug endpoint",
			"static_path": "internal/web/static",
			"files":       []string{"styles.css", "script.js"},
			"time":        time.Now().String(),
		})
	})

	// WebSocket endpoint is registered above with enhanced logging

	// Periodically broadcast updates (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			BroadcastChecksUpdate(mongoDB)
		}
	}()

	// Use PORT environment variable if available, otherwise default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	address := ":" + port
	logrus.Infof("Starting web server on %s", address)
	return router.Run(address)
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

	logrus.Debugf("Using database: %s", mongoDB.Database.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Debug("Executing Find query on check_statuses collection")

	cursor, err := mongoDB.Database.Collection("check_statuses").Find(ctx, bson.M{})
	if err != nil {
		logrus.Errorf("Failed to query check_statuses collection: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	logrus.Debug("Query executed successfully, decoding results")

	var results []models.CheckStatus
	if err := cursor.All(ctx, &results); err != nil {
		logrus.Errorf("Failed to decode check statuses: %v", err)
		return nil, err
	}

	logrus.Debugf("Retrieved %d check statuses from database", len(results))

	// Log some details about the checks if available
	if len(results) > 0 {
		projects := make(map[string]int)
		types := make(map[string]int)
		healthy := 0

		for _, check := range results {
			projects[check.Project]++
			types[check.CheckType]++
			if check.IsHealthy {
				healthy++
			}
		}

		logrus.Debugf("Check statistics - Projects: %d, Types: %d, Healthy: %d/%d",
			len(projects), len(types), healthy, len(results))
	} else {
		logrus.Warn("No check statuses found in database")
	}

	return results, nil
}

func getCheckByUUID(mongoDB *db.MongoDB, uuid string) (models.CheckStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Debugf("Getting check by UUID: %s", uuid)

	var result models.CheckStatus
	err := mongoDB.Database.Collection("check_statuses").FindOne(ctx, bson.M{"uuid": uuid}).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			logrus.Warnf("Check %s not found in database, returning error", uuid)
			// We'll let the caller handle this case
			return result, err
		}

		logrus.Errorf("Error getting check by UUID %s: %v", uuid, err)
		return result, err
	}

	logrus.Debugf("Successfully retrieved check %s (Name: %s, IsEnabled: %v)",
		uuid, result.CheckName, result.IsEnabled)

	return result, nil
}

func toggleCheck(mongoDB *db.MongoDB, uuid string, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Infof("Toggling check %s to enabled=%v", uuid, enabled)

	// First check if the document exists
	var existingCheck models.CheckStatus
	err := mongoDB.Database.Collection("check_statuses").FindOne(ctx, bson.M{"uuid": uuid}).Decode(&existingCheck)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Check doesn't exist, create a new one
			logrus.Warnf("Check %s not found, creating a new one with enabled=%v", uuid, enabled)

			// Create a placeholder check with reasonable defaults
			newCheck := models.CheckStatus{
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

			_, err := mongoDB.Database.Collection("check_statuses").InsertOne(ctx, newCheck)
			if err != nil {
				logrus.Errorf("Failed to create new check %s: %v", uuid, err)
				return fmt.Errorf("failed to create new check: %w", err)
			}

			logrus.Infof("Successfully created new check %s with enabled=%v", uuid, enabled)
			return nil
		}

		// Some other error occurred
		logrus.Errorf("Error checking if check %s exists: %v", uuid, err)
		return fmt.Errorf("error checking if check exists: %w", err)
	}

	// Check exists, update it
	filter := bson.M{"uuid": uuid}
	update := bson.M{"$set": bson.M{"is_enabled": enabled}}

	result, err := mongoDB.Database.Collection("check_statuses").UpdateOne(ctx, filter, update)
	if err != nil {
		logrus.Errorf("Error toggling check %s: %v", uuid, err)
		return err
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

	// Log headers for debugging
	logrus.Debug("Request Headers:")
	for k, v := range c.Request.Header {
		logrus.Debugf("  %s: %v", k, v)
	}

	// Don't override CheckOrigin here - it's already set in the upgrader initialization

	// Log more details about the request
	logrus.Debugf("WebSocket protocol versions: %v", c.Request.Header["Sec-Websocket-Version"])

	// Debug the route
	logrus.Debugf("WebSocket route path: %s, full URL: %s",
		c.Request.URL.Path, c.Request.URL.String())

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	logrus.Debugf("Successfully upgraded to WebSocket connection for: %s", c.Request.RemoteAddr)

	// Register new client
	clientsMux.Lock()
	clients[conn] = true
	clientsMux.Unlock()

	// Set read and write deadlines
	err = conn.SetWriteDeadline(time.Time{}) // No deadline
	if err != nil {
		logrus.Errorf("Failed to set write deadline: %v", err)
	}

	err = conn.SetReadDeadline(time.Time{}) // No deadline
	if err != nil {
		logrus.Errorf("Failed to set read deadline: %v", err)
	}

	// Add a small delay before sending data (wait for client to be ready)
	time.Sleep(200 * time.Millisecond)

	// Set up ping/pong handlers
	conn.SetPingHandler(func(data string) error {
		logrus.Debug("Received ping from client, sending pong")
		err := conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(5*time.Second))
		if err != nil {
			logrus.Errorf("Failed to send pong: %v", err)
		}
		return nil
	})

	conn.SetPongHandler(func(data string) error {
		logrus.Debug("Received pong from client")
		return nil
	})

	// Start a regular pinger for keepalive
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
				if err != nil {
					logrus.Errorf("Failed to send ping: %v", err)
					return
				}
				logrus.Debug("Sent ping to client")
			}
		}
	}()

	// Handle incoming messages and connection lifecycle
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Recovered from panic in WebSocket handler: %v", r)
			}
			logrus.Debug("Closing WebSocket connection")
			conn.Close()
			clientsMux.Lock()
			delete(clients, conn)
			clientsMux.Unlock()
			logrus.Info("WebSocket client disconnected")
		}()

		// Send initial data first with a small delay
		time.Sleep(100 * time.Millisecond)
		logrus.Debug("Sending initial data to new WebSocket client")
		err = sendInitialData(conn, mongoDB)
		if err != nil {
			logrus.Errorf("Failed to send initial data: %v", err)
			return
		}
		logrus.Debug("Initial data sent successfully")

		// Start a ping ticker
		pingTicker := time.NewTicker(30 * time.Second)
		defer pingTicker.Stop()

		// Start a goroutine for ping
		go func() {
			for range pingTicker.C {
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
				if err != nil {
					logrus.Errorf("Failed to send ping: %v", err)
					return
				}
				logrus.Debug("Sent ping to client")
			}
		}()

		for {
			// Read message
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				logrus.Debugf("WebSocket read error: %v", err)

				// Log more details about the connection error
				if closeError, ok := err.(*websocket.CloseError); ok {
					logrus.Debugf("WebSocket close error code: %d, text: %s", closeError.Code, closeError.Text)

					// Check for normal closure (1000) vs abnormal
					if closeError.Code == websocket.CloseNormalClosure {
						logrus.Debug("WebSocket closed normally by client")
					} else if closeError.Code == websocket.CloseNoStatusReceived {
						logrus.Debug("WebSocket closed without status code (1005), likely browser navigation or refresh")
					} else {
						logrus.Warnf("WebSocket closed with abnormal code: %d", closeError.Code)
					}
				} else {
					logrus.Warnf("WebSocket connection error (not a close frame): %v", err)
				}

				break
			}

			// Handle client messages
			if msgType == websocket.TextMessage {
				// logrus.Debugf("Received WebSocket message: %s", string(msg))
				var request map[string]interface{}
				if err := json.Unmarshal(msg, &request); err != nil {
					// logrus.Errorf("Failed to parse WebSocket message: %v", err)
					continue
				}

				// Check the action requested
				action, ok := request["action"].(string)
				if !ok {
					logrus.Warn("WebSocket message missing 'action' field")
					continue
				}

				// logrus.Debugf("Processing WebSocket action: %s", action)

				switch action {
				case "getChecks":
					// Send all checks
					logrus.Debug("Client requested all checks data")
					sendInitialData(conn, mongoDB)

				case "ack":
					// Client acknowledging data receipt
					_, _ = request["received"].(bool)
					// logrus.Debugf("Client acknowledged receipt of data: received=%v", received)
					// Send a quick response to keep connection alive
					err := conn.WriteJSON(map[string]interface{}{
						"type":   "ack",
						"status": "ok",
					})
					if err != nil {
						logrus.Errorf("Failed to send ack response: %v", err)
					}

				case "toggleCheck":
					// Handle toggle check
					uuid, uuidOk := request["uuid"].(string)
					enabled, enabledOk := request["enabled"].(bool)
					if uuidOk && enabledOk {
						logrus.Debugf("Toggling check %s to %v", uuid, enabled)
						err := toggleCheck(mongoDB, uuid, enabled)
						if err != nil {
							logrus.Errorf("Failed to toggle check: %v", err)
						} else {
							logrus.Debugf("Successfully toggled check %s", uuid)
						}
					} else {
						logrus.Warn("Invalid toggleCheck parameters")
					}
				}
			}
		}
	}()
}

// Send initial checks data to a WebSocket client with enhanced logging and retry logic
func sendInitialData(conn *websocket.Conn, mongoDB *db.MongoDB) error {
	logrus.Info("Fetching all check statuses from database")

	// Create some fake data for development/testing if database is empty
	var checks []models.CheckStatus
	checks, err := getAllCheckStatuses(mongoDB)

	if err != nil {
		logrus.Errorf("Failed to get check statuses: %v", err)
		return fmt.Errorf("failed to get check statuses: %w", err)
	}

	logrus.Infof("Retrieved %d check statuses from database", len(checks))

	// If no data, add some fake checks for development and debugging
	if len(checks) == 0 {
		logrus.Warn("No check statuses found in database. Adding test data.")

		// Add some fake data for testing UI
		checks = append(checks, models.CheckStatus{
			Project:     "Test Project",
			CheckName:   "Test Check 1",
			CheckType:   "HTTP",
			IsHealthy:   true,
			IsEnabled:   true,
			LastRun:     time.Now(),
			UUID:        "test-uuid-1",
			Host:        "example.com",
			Periodicity: "1m",
		})

		checks = append(checks, models.CheckStatus{
			Project:     "Test Project",
			CheckName:   "Test Check 2",
			CheckType:   "TCP",
			IsHealthy:   false,
			IsEnabled:   true,
			LastRun:     time.Now(),
			UUID:        "test-uuid-2",
			Message:     "Connection refused",
			Host:        "example.org:8080",
			Periodicity: "5m",
		})
	}

	// Log the actual data we have before conversion
	logrus.Info("Check data sample (first 2 items if available):")
	for i, check := range checks {
		if i < 2 {
			logrus.Infof("Check %d - Project: %s, Name: %s, Type: %s, Healthy: %v",
				i, check.Project, check.CheckName, check.CheckType, check.IsHealthy)
		}
	}

	// Convert to view models
	viewModels := convertToViewModels(checks)
	logrus.Infof("Converted %d check statuses to view models, sending to client", len(viewModels))

	// Create message with additional information
	message := map[string]interface{}{
		"type":      "checks",
		"checks":    viewModels,
		"count":     len(viewModels),
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
	}

	// Log full message for debugging
	logrus.Info("Sending WebSocket message with check data")

	// Set a modest write deadline
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	// Send the message with retry logic
	var sendErr error
	for attempt := 1; attempt <= 3; attempt++ {
		sendErr = conn.WriteJSON(message)
		if sendErr == nil {
			logrus.Info("Successfully sent check data to client")
			break
		}

		logrus.Warnf("Failed to send data (attempt %d/3): %v", attempt, sendErr)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}
	}

	// Reset write deadline
	conn.SetWriteDeadline(time.Time{})

	if sendErr != nil {
		logrus.Errorf("Failed to send data to WebSocket client: %v", sendErr)
		return fmt.Errorf("failed to write data to WebSocket: %w", sendErr)
	}

	logrus.Debug("Successfully sent check data to WebSocket client")
	return nil
}
