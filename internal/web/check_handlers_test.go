package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"
	"context"
)

// stubRepo implements db.Repository with minimal stubs for testing CreateCheckDefinition.
type stubRepo struct {
	db.Repository
	lastCreated models.CheckDefinition
}

func (s *stubRepo) CreateCheckDefinition(_ context.Context, def models.CheckDefinition) (string, error) {
	s.lastCreated = def
	return def.UUID, nil
}

func (s *stubRepo) GetAllCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (s *stubRepo) GetEnabledCheckDefinitions(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (s *stubRepo) GetCheckDefinitionByUUID(_ context.Context, _ string) (models.CheckDefinition, error) {
	return models.CheckDefinition{}, nil
}
func (s *stubRepo) UpdateCheckDefinition(_ context.Context, _ models.CheckDefinition) error {
	return nil
}
func (s *stubRepo) DeleteCheckDefinition(_ context.Context, _ string) error { return nil }
func (s *stubRepo) ToggleCheckDefinition(_ context.Context, _ string, _ bool) error {
	return nil
}
func (s *stubRepo) UpdateCheckStatus(_ context.Context, _ models.CheckStatus) error { return nil }
func (s *stubRepo) GetAllProjects(_ context.Context) ([]string, error)   { return nil, nil }
func (s *stubRepo) GetAllCheckTypes(_ context.Context) ([]string, error) { return nil, nil }
func (s *stubRepo) ConvertConfigToCheckDefinitions(_ context.Context, _ *config.Config) error {
	return nil
}
func (s *stubRepo) GetAllDefaultTimeouts() map[string]string { return nil }
func (s *stubRepo) CreateSlackThread(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (s *stubRepo) GetUnresolvedThread(_ context.Context, _ string) (models.SlackAlertThread, error) {
	return models.SlackAlertThread{}, nil
}
func (s *stubRepo) ResolveThread(_ context.Context, _ string) error { return nil }
func (s *stubRepo) UpdateSlackThread(_ context.Context, _, _, _ string) error {
	return nil
}
func (s *stubRepo) CreateSilence(_ context.Context, _ models.AlertSilence) error { return nil }
func (s *stubRepo) IsCheckSilenced(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (s *stubRepo) DeactivateSilence(_ context.Context, _, _ string) error { return nil }
func (s *stubRepo) DeactivateSilenceByID(_ context.Context, _ int) error   { return nil }
func (s *stubRepo) GetActiveSilences(_ context.Context) ([]models.AlertSilence, error) {
	return nil, nil
}
func (s *stubRepo) GetUnhealthyChecks(_ context.Context) ([]models.CheckDefinition, error) {
	return nil, nil
}
func (s *stubRepo) CreateAlertEvent(_ context.Context, _ models.AlertEvent) error { return nil }
func (s *stubRepo) ResolveAlertEvent(_ context.Context, _ string) error           { return nil }
func (s *stubRepo) GetAlertHistory(_ context.Context, _, _ int, _ models.AlertHistoryFilters) ([]models.AlertEvent, int, error) {
	return nil, 0, nil
}

func TestCreateCheckDefinition_GeneratesUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"name":"test-check","project":"proj","type":"http","url":"https://example.com"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/check-definitions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateCheckDefinition(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify UUID was generated
	if repo.lastCreated.UUID == "" {
		t.Fatal("expected UUID to be generated, got empty string")
	}

	// Verify it's a valid UUID
	if _, err := uuid.Parse(repo.lastCreated.UUID); err != nil {
		t.Fatalf("generated UUID is not valid: %s", repo.lastCreated.UUID)
	}

	// Verify timestamps were set
	if repo.lastCreated.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if repo.lastCreated.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}

	// Verify response contains UUID
	var resp models.CheckDefinitionViewModel
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.UUID == "" {
		t.Fatal("expected UUID in response, got empty string")
	}
}

func TestCreateCheckDefinition_PreservesProvidedUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	providedUUID := "550e8400-e29b-41d4-a716-446655440000"
	body := `{"name":"test-check","project":"proj","type":"http","url":"https://example.com","uuid":"` + providedUUID + `"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/check-definitions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateCheckDefinition(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if repo.lastCreated.UUID != providedUUID {
		t.Fatalf("expected UUID %s, got %s", providedUUID, repo.lastCreated.UUID)
	}
}
