package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"checker/internal/alerts"
	"checker/internal/db"
	"checker/internal/discord"
	"checker/internal/models"
)

// ListAlertChannels returns all alert channels with sensitive fields masked.
// GET /api/alert-channels
func ListAlertChannels(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	ctx := c.Request.Context()

	channels, err := repo.GetAllAlertChannels(ctx)
	if err != nil {
		logrus.Errorf("Failed to get alert channels: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alert channels"})
		return
	}

	if channels == nil {
		channels = []models.AlertChannel{}
	}

	// Mask sensitive fields in the response
	for i := range channels {
		channels[i].Config = models.MaskSensitiveConfig(channels[i].Type, channels[i].Config)
	}

	c.JSON(http.StatusOK, channels)
}

// CreateAlertChannel creates a new alert channel.
// POST /api/alert-channels
func CreateAlertChannel(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var channel models.AlertChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert channel data"})
		return
	}

	if channel.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel name is required"})
		return
	}
	if !alerts.IsRegisteredType(channel.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid channel type: %s", channel.Type)})
		return
	}
	if len(channel.Config) == 0 {
		channel.Config = json.RawMessage("{}")
	}

	ctx := c.Request.Context()
	if err := repo.CreateAlertChannel(ctx, channel); err != nil {
		logrus.Errorf("Failed to create alert channel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert channel"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Alert channel created", "name": channel.Name})
}

// UpdateAlertChannel updates an existing alert channel.
// PUT /api/alert-channels/:name
func UpdateAlertChannel(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel name is required"})
		return
	}

	var channel models.AlertChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert channel data"})
		return
	}

	// Ensure name from URL is used
	channel.Name = name

	if !alerts.IsRegisteredType(channel.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid channel type: %s", channel.Type)})
		return
	}

	ctx := c.Request.Context()

	// If config contains masked values, merge with existing config
	existing, err := repo.GetAlertChannelByName(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert channel not found"})
		return
	}

	channel.Config = mergeConfigWithExisting(channel.Type, channel.Config, existing.Config)

	if err := repo.UpdateAlertChannel(ctx, channel); err != nil {
		logrus.Errorf("Failed to update alert channel %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alert channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert channel updated", "name": name})
}

// DeleteAlertChannel deletes an alert channel.
// DELETE /api/alert-channels/:name
func DeleteAlertChannel(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel name is required"})
		return
	}

	ctx := c.Request.Context()
	if err := repo.DeleteAlertChannel(ctx, name); err != nil {
		logrus.Errorf("Failed to delete alert channel %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert channel deleted", "name": name})
}

// TestAlertChannel sends a test notification through the specified channel.
// POST /api/alert-channels/:name/test
func TestAlertChannel(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel name is required"})
		return
	}

	ctx := c.Request.Context()
	channel, err := repo.GetAlertChannelByName(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert channel not found"})
		return
	}

	testErr := sendTestNotification(channel)
	if testErr != nil {
		logrus.Errorf("Test notification failed for channel %s: %v", name, testErr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   fmt.Sprintf("Test notification failed: %v", testErr),
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Test notification sent successfully",
		"success":   true,
		"tested_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// sendTestNotification dispatches a test message based on channel type.
func sendTestNotification(channel models.AlertChannel) error {
	var cfg map[string]interface{}
	if err := json.Unmarshal(channel.Config, &cfg); err != nil {
		return fmt.Errorf("invalid channel config: %w", err)
	}

	testMessage := fmt.Sprintf("🧪 Test notification from Checker — channel: %s (%s)", channel.Name, channel.Type)

	switch channel.Type {
	case "telegram":
		botToken, _ := cfg["bot_token"].(string)
		chatID, _ := cfg["chat_id"].(string)
		if botToken == "" || chatID == "" {
			return fmt.Errorf("telegram requires bot_token and chat_id")
		}
		return alerts.SendTelegramAlert(botToken, chatID, testMessage)

	case "slack":
		botToken, _ := cfg["bot_token"].(string)
		defaultChannel, _ := cfg["default_channel"].(string)
		if botToken == "" || defaultChannel == "" {
			return fmt.Errorf("slack app requires bot_token and default_channel")
		}
		return alerts.SendSlackAppTest(botToken, defaultChannel, testMessage)

	case "slack_webhook":
		webhookURL, _ := cfg["webhook_url"].(string)
		if webhookURL == "" {
			return fmt.Errorf("slack webhook requires webhook_url")
		}
		return alerts.SendSlackAlert(webhookURL, testMessage)

	case "email":
		smtpHost, _ := cfg["smtp_host"].(string)
		smtpPortF, _ := cfg["smtp_port"].(float64)
		smtpUser, _ := cfg["smtp_user"].(string)
		smtpPassword, _ := cfg["smtp_password"].(string)
		from, _ := cfg["from"].(string)
		toRaw, _ := cfg["to"].([]interface{})
		useTLS, _ := cfg["use_tls"].(bool)

		var to []string
		for _, t := range toRaw {
			if s, ok := t.(string); ok {
				to = append(to, s)
			}
		}

		if smtpHost == "" || from == "" || len(to) == 0 {
			return fmt.Errorf("email requires smtp_host, from, and to")
		}

		emailCfg := alerts.EmailConfig{
			SMTPHost:     smtpHost,
			SMTPPort:     int(smtpPortF),
			SMTPUser:     smtpUser,
			SMTPPassword: smtpPassword,
			From:         from,
			To:           to,
			UseTLS:       useTLS,
		}
		data := alerts.EmailData{
			Subject:      "Checker Test Notification",
			HeaderClass:  "info",
			CheckName:    "Test Check",
			Project:      "Test Project",
			CheckType:    "test",
			ErrorMessage: testMessage,
			Timestamp:    time.Now().Format(time.RFC3339),
		}
		return alerts.SendEmailAlert(emailCfg, data)

	case "discord":
		webhookURL, _ := cfg["webhook_url"].(string)
		if webhookURL == "" {
			return fmt.Errorf("discord requires webhook_url")
		}
		payload := alerts.BuildDiscordPayload(alerts.DiscordAlertParams{
			CheckName: "Test Check",
			Project:   "Test Project",
			CheckType: "test",
			Message:   testMessage,
			IsDown:    false,
		})
		return alerts.SendDiscordAlert(webhookURL, payload)

	case "discord_bot":
		botToken, _ := cfg["bot_token"].(string)
		appID, _ := cfg["app_id"].(string)
		defaultChannel, _ := cfg["default_channel"].(string)
		if botToken == "" || defaultChannel == "" {
			return fmt.Errorf("discord_bot requires bot_token and default_channel")
		}
		client := discord.NewDiscordClient(botToken, appID, defaultChannel)
		payload := discord.BuildAlertMessage(discord.CheckAlertInfo{
			Name:      "Test Check",
			Project:   "Test Project",
			CheckType: "test",
			Message:   testMessage,
		})
		_, err := client.SendMessage(context.Background(), defaultChannel, payload)
		return err

	case "teams":
		webhookURL, _ := cfg["webhook_url"].(string)
		if webhookURL == "" {
			return fmt.Errorf("teams requires webhook_url")
		}
		return alerts.SendTeamsAlert(webhookURL, alerts.TeamsAlertParams{
			CheckName:   "Test Check",
			ProjectName: "Test Project",
			Status:      "RESOLVED",
			Error:       testMessage,
			Time:        time.Now(),
		})

	case "pagerduty":
		routingKey, _ := cfg["routing_key"].(string)
		if routingKey == "" {
			return fmt.Errorf("pagerduty requires routing_key")
		}
		return alerts.SendPagerDutyTrigger(routingKey, "checker-test", "Test Check", testMessage, "info")

	case "opsgenie":
		apiKey, _ := cfg["api_key"].(string)
		region, _ := cfg["region"].(string)
		if apiKey == "" {
			return fmt.Errorf("opsgenie requires api_key")
		}
		if region == "" {
			region = "us"
		}
		client := &alerts.OpsgenieClient{
			APIKey: apiKey,
			Region: region,
		}
		return client.Trigger("Test Check", "checker-test", testMessage, "info")

	case "ntfy":
		serverURL, _ := cfg["server_url"].(string)
		topic, _ := cfg["topic"].(string)
		token, _ := cfg["token"].(string)
		username, _ := cfg["username"].(string)
		password, _ := cfg["password"].(string)
		clickURL, _ := cfg["click_url"].(string)
		insecure, _ := cfg["insecure"].(bool)
		if topic == "" {
			return fmt.Errorf("ntfy requires topic")
		}
		return alerts.SendNtfyTest(serverURL, topic, token, username, password, testMessage, clickURL, insecure)

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// mergeConfigWithExisting takes the new config and replaces any masked values
// with the original values from the existing config.
func mergeConfigWithExisting(channelType string, newConfig, existingConfig json.RawMessage) json.RawMessage {
	sensitiveFields, ok := models.SensitiveFields[channelType]
	if !ok || len(sensitiveFields) == 0 {
		return newConfig
	}

	var newCfg, existCfg map[string]interface{}
	if err := json.Unmarshal(newConfig, &newCfg); err != nil {
		return newConfig
	}
	if err := json.Unmarshal(existingConfig, &existCfg); err != nil {
		return newConfig
	}

	for _, field := range sensitiveFields {
		if val, exists := newCfg[field]; exists {
			if s, ok := val.(string); ok {
				// If the value looks masked (contains ****), use the existing value
				if len(s) > 0 && containsMask(s) {
					if origVal, origExists := existCfg[field]; origExists {
						newCfg[field] = origVal
					}
				}
			}
		}
	}

	merged, err := json.Marshal(newCfg)
	if err != nil {
		return newConfig
	}
	return merged
}

func containsMask(s string) bool {
	for i := 0; i < len(s)-3; i++ {
		if s[i] == '*' && s[i+1] == '*' && s[i+2] == '*' && s[i+3] == '*' {
			return true
		}
	}
	return false
}
