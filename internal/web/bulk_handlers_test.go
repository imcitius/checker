package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imcitius/checker/pkg/db"

	"github.com/gin-gonic/gin"
)

func TestBulkEnableChecks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"uuids":["uuid-1","uuid-2","uuid-3"]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/checks/bulk-enable", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	BulkEnableChecks(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["success"] != true {
		t.Fatal("expected success to be true")
	}
	if int(resp["count"].(float64)) != 3 {
		t.Fatalf("expected count 3, got %v", resp["count"])
	}

	if len(repo.lastBulkUUIDs) != 3 {
		t.Fatalf("expected 3 UUIDs, got %d", len(repo.lastBulkUUIDs))
	}
	if !repo.lastBulkEnabled {
		t.Fatal("expected enabled to be true")
	}
}

func TestBulkDisableChecks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"uuids":["uuid-1","uuid-2"]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/checks/bulk-disable", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	BulkDisableChecks(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["success"] != true {
		t.Fatal("expected success to be true")
	}
	if int(resp["count"].(float64)) != 2 {
		t.Fatalf("expected count 2, got %v", resp["count"])
	}

	if repo.lastBulkEnabled {
		t.Fatal("expected enabled to be false")
	}
}

func TestBulkDeleteChecks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"uuids":["uuid-1","uuid-2","uuid-3","uuid-4"]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/checks/bulk-delete", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	BulkDeleteChecks(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["success"] != true {
		t.Fatal("expected success to be true")
	}
	if int(resp["count"].(float64)) != 4 {
		t.Fatalf("expected count 4, got %v", resp["count"])
	}

	if len(repo.lastBulkUUIDs) != 4 {
		t.Fatalf("expected 4 UUIDs, got %d", len(repo.lastBulkUUIDs))
	}
}

func TestBulkEnableChecks_EmptyUUIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"uuids":[]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/checks/bulk-enable", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	BulkEnableChecks(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBulkDeleteChecks_MissingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/api/checks/bulk-delete", strings.NewReader("{}"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	BulkDeleteChecks(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBulkDisableChecks_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/api/checks/bulk-disable", strings.NewReader("not json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("repo", db.Repository(repo))

	BulkDisableChecks(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}
