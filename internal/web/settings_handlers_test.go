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
	"github.com/stretchr/testify/assert"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// settingsStubRepo extends stubRepo with settings methods.
type settingsStubRepo struct {
	stubRepo
	defaults   models.CheckDefaults
	saved      models.CheckDefaults
	failGet    bool
	failSave   bool
}

func (r *settingsStubRepo) GetCheckDefaults(_ context.Context) (models.CheckDefaults, error) {
	if r.failGet {
		return models.CheckDefaults{}, fmt.Errorf("db error")
	}
	return r.defaults, nil
}

func (r *settingsStubRepo) SaveCheckDefaults(_ context.Context, defaults models.CheckDefaults) error {
	if r.failSave {
		return fmt.Errorf("db error")
	}
	r.saved = defaults
	r.defaults = defaults
	return nil
}

func TestGetCheckDefaults_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingsStubRepo{
		defaults: models.CheckDefaults{
			RetryCount:    3,
			RetryInterval: "30s",
			CheckInterval: "1m",
			Timeouts:      map[string]string{"http": "10s", "tcp": "5s"},
			Severity:      "critical",
			AlertChannels: []string{"slack", "pagerduty"},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/settings/check-defaults", nil)
	c.Set("repo", db.Repository(repo))

	GetCheckDefaults(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.CheckDefaults
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, 3, result.RetryCount)
	assert.Equal(t, "30s", result.RetryInterval)
	assert.Equal(t, "1m", result.CheckInterval)
	assert.Equal(t, "critical", result.Severity)
	assert.Len(t, result.AlertChannels, 2)
}

func TestGetCheckDefaults_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingsStubRepo{
		defaults: models.CheckDefaults{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/settings/check-defaults", nil)
	c.Set("repo", db.Repository(repo))

	GetCheckDefaults(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCheckDefaults_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingsStubRepo{failGet: true}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/settings/check-defaults", nil)
	c.Set("repo", db.Repository(repo))

	GetCheckDefaults(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateCheckDefaults_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingsStubRepo{}

	body := `{"retry_count":5,"retry_interval":"60s","check_interval":"2m","severity":"warning","alert_channels":["telegram"]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/settings/check-defaults", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	UpdateCheckDefaults(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 5, repo.saved.RetryCount)
	assert.Equal(t, "60s", repo.saved.RetryInterval)
	assert.Equal(t, "warning", repo.saved.Severity)

	var result models.CheckDefaults
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, 5, result.RetryCount)
}

func TestUpdateCheckDefaults_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingsStubRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/settings/check-defaults", bytes.NewBufferString(`{bad`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	UpdateCheckDefaults(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCheckDefaults_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingsStubRepo{failSave: true}

	body := `{"retry_count":3}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/settings/check-defaults", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	UpdateCheckDefaults(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateCheckDefaults_WithTimeouts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingsStubRepo{}

	body := `{"retry_count":2,"timeouts":{"http":"15s","tcp":"10s","dns":"5s"}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/settings/check-defaults", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	UpdateCheckDefaults(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "15s", repo.saved.Timeouts["http"])
	assert.Equal(t, "10s", repo.saved.Timeouts["tcp"])
	assert.Equal(t, "5s", repo.saved.Timeouts["dns"])
}
