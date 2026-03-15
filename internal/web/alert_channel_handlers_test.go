package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"checker/internal/db"
	"checker/internal/models"
)

// alertChannelStubRepo extends stubRepo with alert channel CRUD.
type alertChannelStubRepo struct {
	stubRepo
	channels []models.AlertChannel
}

func (r *alertChannelStubRepo) GetAllAlertChannels(_ context.Context) ([]models.AlertChannel, error) {
	return r.channels, nil
}

func (r *alertChannelStubRepo) GetAlertChannelByName(_ context.Context, name string) (models.AlertChannel, error) {
	for _, ch := range r.channels {
		if ch.Name == name {
			return ch, nil
		}
	}
	return models.AlertChannel{}, fmt.Errorf("alert channel not found")
}

func (r *alertChannelStubRepo) CreateAlertChannel(_ context.Context, channel models.AlertChannel) error {
	r.channels = append(r.channels, channel)
	return nil
}

func (r *alertChannelStubRepo) UpdateAlertChannel(_ context.Context, channel models.AlertChannel) error {
	for i, ch := range r.channels {
		if ch.Name == channel.Name {
			r.channels[i] = channel
			return nil
		}
	}
	return fmt.Errorf("alert channel not found")
}

func (r *alertChannelStubRepo) DeleteAlertChannel(_ context.Context, name string) error {
	for i, ch := range r.channels {
		if ch.Name == name {
			r.channels = append(r.channels[:i], r.channels[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("alert channel not found")
}

func TestListAlertChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertChannelStubRepo{
		channels: []models.AlertChannel{
			{
				ID:     1,
				Name:   "test-slack",
				Type:   "slack_webhook",
				Config: json.RawMessage(`{"webhook_url":"https://hooks.slack.com/services/SECRET123"}`),
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/alert-channels", nil)
	c.Set("repo", db.Repository(repo))

	ListAlertChannels(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []models.AlertChannel
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(result))
	}

	// Verify sensitive field is masked
	var cfg map[string]interface{}
	json.Unmarshal(result[0].Config, &cfg)
	webhookURL, _ := cfg["webhook_url"].(string)
	if webhookURL == "https://hooks.slack.com/services/SECRET123" {
		t.Error("webhook_url should be masked but was returned in plain text")
	}
	if !containsMask(webhookURL) {
		t.Errorf("webhook_url should contain mask (****), got: %s", webhookURL)
	}
}

func TestListAlertChannels_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertChannelStubRepo{channels: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/alert-channels", nil)
	c.Set("repo", db.Repository(repo))

	ListAlertChannels(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []models.AlertChannel
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 channels, got %d", len(result))
	}
}

func TestCreateAlertChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid telegram channel",
			body:       `{"name":"my-telegram","type":"telegram","config":{"bot_token":"abc","chat_id":"123"}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid slack channel",
			body:       `{"name":"my-slack","type":"slack","config":{"webhook_url":"https://hooks.slack.com/test"}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing name",
			body:       `{"type":"slack","config":{}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid type",
			body:       `{"name":"test","type":"whatsapp","config":{}}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &alertChannelStubRepo{channels: []models.AlertChannel{}}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/alert-channels",
				bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("repo", db.Repository(repo))

			CreateAlertChannel(c)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestDeleteAlertChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertChannelStubRepo{
		channels: []models.AlertChannel{
			{ID: 1, Name: "test-channel", Type: "slack", Config: json.RawMessage(`{}`)},
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("repo", db.Repository(repo))
		c.Next()
	})
	r.DELETE("/api/alert-channels/:name", DeleteAlertChannel)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/alert-channels/test-channel", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(repo.channels) != 0 {
		t.Errorf("expected 0 channels after delete, got %d", len(repo.channels))
	}
}

func TestDeleteAlertChannel_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertChannelStubRepo{channels: []models.AlertChannel{}}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("repo", db.Repository(repo))
		c.Next()
	})
	r.DELETE("/api/alert-channels/:name", DeleteAlertChannel)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/alert-channels/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for not found, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAlertChannel_PreservesMaskedSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertChannelStubRepo{
		channels: []models.AlertChannel{
			{
				ID:     1,
				Name:   "my-slack",
				Type:   "slack_webhook",
				Config: json.RawMessage(`{"webhook_url":"https://hooks.slack.com/real-secret"}`),
			},
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("repo", db.Repository(repo))
		c.Next()
	})
	r.PUT("/api/alert-channels/:name", UpdateAlertChannel)

	// Update with masked value — should preserve the original
	body := `{"type":"slack_webhook","config":{"webhook_url":"ht****et"}}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/alert-channels/my-slack",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the secret was preserved
	var cfg map[string]interface{}
	json.Unmarshal(repo.channels[0].Config, &cfg)
	if cfg["webhook_url"] != "https://hooks.slack.com/real-secret" {
		t.Errorf("expected original webhook_url to be preserved, got: %v", cfg["webhook_url"])
	}
}

func TestMaskSensitiveConfig(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		config      string
		checkField  string
		shouldMask  bool
	}{
		{
			name:        "slack webhook masked",
			channelType: "slack_webhook",
			config:      `{"webhook_url":"https://hooks.slack.com/services/SECRET"}`,
			checkField:  "webhook_url",
			shouldMask:  true,
		},
		{
			name:        "telegram bot_token masked",
			channelType: "telegram",
			config:      `{"bot_token":"123456:ABC-DEF","chat_id":"-100123"}`,
			checkField:  "bot_token",
			shouldMask:  true,
		},
		{
			name:        "non-sensitive field not masked",
			channelType: "telegram",
			config:      `{"bot_token":"123456:ABC-DEF","chat_id":"-100123"}`,
			checkField:  "chat_id",
			shouldMask:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := models.MaskSensitiveConfig(tt.channelType, json.RawMessage(tt.config))
			var cfg map[string]interface{}
			json.Unmarshal(masked, &cfg)

			val, _ := cfg[tt.checkField].(string)
			hasMask := containsMask(val)
			if tt.shouldMask && !hasMask {
				t.Errorf("expected %s to be masked, got: %s", tt.checkField, val)
			}
			if !tt.shouldMask && hasMask {
				t.Errorf("expected %s NOT to be masked, got: %s", tt.checkField, val)
			}
		})
	}
}
