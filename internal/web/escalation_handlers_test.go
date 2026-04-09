// SPDX-License-Identifier: BUSL-1.1

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

// escalationStubRepo extends stubRepo with escalation policy CRUD.
type escalationStubRepo struct {
	stubRepo
	policies   []models.EscalationPolicy
	failGet    bool
	failCreate bool
	failUpdate bool
	failDelete bool
}

func (r *escalationStubRepo) GetAllEscalationPolicies(_ context.Context) ([]models.EscalationPolicy, error) {
	if r.failGet {
		return nil, fmt.Errorf("db error")
	}
	return r.policies, nil
}

func (r *escalationStubRepo) CreateEscalationPolicy(_ context.Context, policy models.EscalationPolicy) error {
	if r.failCreate {
		return fmt.Errorf("db error")
	}
	r.policies = append(r.policies, policy)
	return nil
}

func (r *escalationStubRepo) UpdateEscalationPolicy(_ context.Context, policy models.EscalationPolicy) error {
	if r.failUpdate {
		return fmt.Errorf("db error")
	}
	for i, p := range r.policies {
		if p.Name == policy.Name {
			r.policies[i] = policy
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func (r *escalationStubRepo) DeleteEscalationPolicy(_ context.Context, name string) error {
	if r.failDelete {
		return fmt.Errorf("db error")
	}
	for i, p := range r.policies {
		if p.Name == name {
			r.policies = append(r.policies[:i], r.policies[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func TestListEscalationPolicies_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{
		policies: []models.EscalationPolicy{
			{ID: 1, Name: "critical", Steps: []models.EscalationStep{{Channel: "slack", DelayMin: 0}}},
			{ID: 2, Name: "warning", Steps: []models.EscalationStep{{Channel: "email", DelayMin: 5}}},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/escalation-policies", nil)
	c.Set("repo", db.Repository(repo))

	ListEscalationPolicies(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.EscalationPolicy
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "critical", result[0].Name)
}

func TestListEscalationPolicies_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{policies: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/escalation-policies", nil)
	c.Set("repo", db.Repository(repo))

	ListEscalationPolicies(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.EscalationPolicy
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestListEscalationPolicies_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{failGet: true}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/escalation-policies", nil)
	c.Set("repo", db.Repository(repo))

	ListEscalationPolicies(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateEscalationPolicy_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{policies: []models.EscalationPolicy{}}

	body := `{"name":"critical","steps":[{"channel":"slack","delay_min":0},{"channel":"pagerduty","delay_min":15}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/escalation-policies", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateEscalationPolicy(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Len(t, repo.policies, 1)
	assert.Equal(t, "critical", repo.policies[0].Name)
	assert.Len(t, repo.policies[0].Steps, 2)
}

func TestCreateEscalationPolicy_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/escalation-policies", bytes.NewBufferString(`{invalid`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateEscalationPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEscalationPolicy_MissingName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{}

	body := `{"steps":[{"channel":"slack","delay_min":0}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/escalation-policies", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateEscalationPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEscalationPolicy_NoSteps(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{}

	body := `{"name":"critical","steps":[]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/escalation-policies", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateEscalationPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEscalationPolicy_StepMissingChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{}

	body := `{"name":"critical","steps":[{"delay_min":0}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/escalation-policies", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateEscalationPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEscalationPolicy_NegativeDelay(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{}

	body := `{"name":"critical","steps":[{"channel":"slack","delay_min":-1}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/escalation-policies", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateEscalationPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEscalationPolicy_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{failCreate: true}

	body := `{"name":"critical","steps":[{"channel":"slack","delay_min":0}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/escalation-policies", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	CreateEscalationPolicy(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateEscalationPolicy_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{
		policies: []models.EscalationPolicy{
			{ID: 1, Name: "critical", Steps: []models.EscalationStep{{Channel: "slack", DelayMin: 0}}},
		},
	}

	body := `{"steps":[{"channel":"pagerduty","delay_min":5}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/escalation-policies/critical", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))
	c.Params = []gin.Param{{Key: "name", Value: "critical"}}

	UpdateEscalationPolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pagerduty", repo.policies[0].Steps[0].Channel)
}

func TestUpdateEscalationPolicy_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/escalation-policies/critical", bytes.NewBufferString(`{bad`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))
	c.Params = []gin.Param{{Key: "name", Value: "critical"}}

	UpdateEscalationPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEscalationPolicy_NoSteps(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{}

	body := `{"steps":[]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/escalation-policies/critical", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))
	c.Params = []gin.Param{{Key: "name", Value: "critical"}}

	UpdateEscalationPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEscalationPolicy_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{failUpdate: true}

	body := `{"steps":[{"channel":"slack","delay_min":0}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/escalation-policies/critical", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))
	c.Params = []gin.Param{{Key: "name", Value: "critical"}}

	UpdateEscalationPolicy(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteEscalationPolicy_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{
		policies: []models.EscalationPolicy{
			{ID: 1, Name: "critical", Steps: []models.EscalationStep{{Channel: "slack", DelayMin: 0}}},
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("repo", db.Repository(repo))
		c.Next()
	})
	r.DELETE("/api/escalation-policies/:name", DeleteEscalationPolicy)

	req := httptest.NewRequest(http.MethodDelete, "/api/escalation-policies/critical", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, repo.policies, 0)
}

func TestDeleteEscalationPolicy_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{failDelete: true}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("repo", db.Repository(repo))
		c.Next()
	})
	r.DELETE("/api/escalation-policies/:name", DeleteEscalationPolicy)

	req := httptest.NewRequest(http.MethodDelete, "/api/escalation-policies/critical", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateEscalationPolicy_NameFromURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &escalationStubRepo{
		policies: []models.EscalationPolicy{
			{ID: 1, Name: "critical", Steps: []models.EscalationStep{{Channel: "slack", DelayMin: 0}}},
		},
	}

	// Body has a different name, but URL name should win
	body := `{"name":"other-name","steps":[{"channel":"pagerduty","delay_min":5}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/escalation-policies/critical", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))
	c.Params = []gin.Param{{Key: "name", Value: "critical"}}

	UpdateEscalationPolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.EscalationPolicy
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "critical", resp.Name, "URL name should override body name")
}
