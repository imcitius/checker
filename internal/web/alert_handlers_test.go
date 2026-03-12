package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"checker/internal/db"
	"checker/internal/models"
)

// alertStubRepo extends stubRepo with configurable behavior for alert/silence tests.
type alertStubRepo struct {
	stubRepo

	alertEvents    []models.AlertEvent
	alertTotal     int
	activeSilences []models.AlertSilence

	createSilenceErr      error
	deactivateByIDErr     error
	lastCreatedSilence    models.AlertSilence
	lastDeactivatedID     int
}

func (r *alertStubRepo) GetAlertHistory(_ context.Context, limit, offset int, _ models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	// Simulate pagination
	end := offset + limit
	if end > len(r.alertEvents) {
		end = len(r.alertEvents)
	}
	if offset >= len(r.alertEvents) {
		return []models.AlertEvent{}, r.alertTotal, nil
	}
	return r.alertEvents[offset:end], r.alertTotal, nil
}

func (r *alertStubRepo) GetActiveSilences(_ context.Context) ([]models.AlertSilence, error) {
	return r.activeSilences, nil
}

func (r *alertStubRepo) CreateSilence(_ context.Context, s models.AlertSilence) error {
	r.lastCreatedSilence = s
	return r.createSilenceErr
}

func (r *alertStubRepo) DeactivateSilenceByID(_ context.Context, id int) error {
	r.lastDeactivatedID = id
	return r.deactivateByIDErr
}

// --- Tests ---

func TestListAlerts_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{alertTotal: 0}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
	c.Set("repo", db.Repository(repo))

	ListAlerts(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	alerts, ok := resp["alerts"].([]interface{})
	if !ok {
		t.Fatal("expected alerts array in response")
	}
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(alerts))
	}
	if resp["total"].(float64) != 0 {
		t.Fatalf("expected total 0, got %v", resp["total"])
	}
}

func TestListAlerts_WithResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	events := []models.AlertEvent{
		{ID: 1, CheckUUID: "uuid-1", CheckName: "check-1", Project: "proj", CreatedAt: now, IsResolved: false},
		{ID: 2, CheckUUID: "uuid-2", CheckName: "check-2", Project: "proj", CreatedAt: now, IsResolved: true},
	}

	repo := &alertStubRepo{alertEvents: events, alertTotal: 2}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/alerts?limit=10&offset=0", nil)
	c.Set("repo", db.Repository(repo))

	ListAlerts(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	alerts := resp["alerts"].([]interface{})
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}
}

func TestListSilences_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/silences", nil)
	c.Set("repo", db.Repository(repo))

	ListSilences(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	silences := resp["silences"].([]interface{})
	if len(silences) != 0 {
		t.Fatalf("expected 0 silences, got %d", len(silences))
	}
}

func TestListSilences_WithResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	silences := []models.AlertSilence{
		{ID: 1, Scope: "check", Target: "uuid-1", SilencedBy: "user@test.com", SilencedAt: now, Active: true},
	}

	repo := &alertStubRepo{activeSilences: silences}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/silences", nil)
	c.Set("repo", db.Repository(repo))

	ListSilences(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	result := resp["silences"].([]interface{})
	if len(result) != 1 {
		t.Fatalf("expected 1 silence, got %d", len(result))
	}
}

func TestCreateSilence_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"scope":"check","target":"uuid-123","duration":"1h","reason":"maintenance"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/silences", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))
	c.Set("user_email", "admin@test.com")

	CreateSilence(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if repo.lastCreatedSilence.Scope != "check" {
		t.Fatalf("expected scope 'check', got %q", repo.lastCreatedSilence.Scope)
	}
	if repo.lastCreatedSilence.Target != "uuid-123" {
		t.Fatalf("expected target 'uuid-123', got %q", repo.lastCreatedSilence.Target)
	}
	if repo.lastCreatedSilence.SilencedBy != "admin@test.com" {
		t.Fatalf("expected silenced_by 'admin@test.com', got %q", repo.lastCreatedSilence.SilencedBy)
	}
	if repo.lastCreatedSilence.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set for 1h duration")
	}
	if repo.lastCreatedSilence.Reason != "maintenance" {
		t.Fatalf("expected reason 'maintenance', got %q", repo.lastCreatedSilence.Reason)
	}
}

func TestCreateSilence_Indefinite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"scope":"project","target":"my-project","duration":"indefinite"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/silences", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateSilence(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if repo.lastCreatedSilence.ExpiresAt != nil {
		t.Fatal("expected ExpiresAt to be nil for indefinite duration")
	}
	if repo.lastCreatedSilence.SilencedBy != "ui" {
		t.Fatalf("expected silenced_by 'ui' when no user context, got %q", repo.lastCreatedSilence.SilencedBy)
	}
}

func TestCreateSilence_InvalidScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"scope":"invalid","target":"uuid-123","duration":"1h"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/silences", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateSilence(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSilence_InvalidDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"scope":"check","target":"uuid-123","duration":"99h"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/silences", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateSilence(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSilence_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/silences/42", nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Set("repo", db.Repository(repo))

	DeleteSilence(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if repo.lastDeactivatedID != 42 {
		t.Fatalf("expected deactivated ID 42, got %d", repo.lastDeactivatedID)
	}
}

func TestDeleteSilence_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/silences/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	c.Set("repo", db.Repository(repo))

	DeleteSilence(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSilence_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &alertStubRepo{deactivateByIDErr: fmt.Errorf("silence not found or already inactive")}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/silences/999", nil)
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	c.Set("repo", db.Repository(repo))

	DeleteSilence(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
