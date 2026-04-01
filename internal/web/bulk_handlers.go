package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/pkg/db"
)

// bulkUUIDsRequest is the common request body for bulk operations.
type bulkUUIDsRequest struct {
	UUIDs []string `json:"uuids" binding:"required"`
}

// BulkEnableChecks enables multiple check definitions at once.
// POST /api/checks/bulk-enable
func BulkEnableChecks(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var req bulkUUIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.UUIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'uuids' is required and must be a non-empty array"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	count, err := repo.BulkToggleCheckDefinitions(ctx, req.UUIDs, true)
	if err != nil {
		logrus.Errorf("Failed to bulk enable checks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable checks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "count": count})
}

// BulkDisableChecks disables multiple check definitions at once.
// POST /api/checks/bulk-disable
func BulkDisableChecks(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var req bulkUUIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.UUIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'uuids' is required and must be a non-empty array"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	count, err := repo.BulkToggleCheckDefinitions(ctx, req.UUIDs, false)
	if err != nil {
		logrus.Errorf("Failed to bulk disable checks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable checks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "count": count})
}

// BulkDeleteChecks deletes multiple check definitions at once.
// POST /api/checks/bulk-delete
func BulkDeleteChecks(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var req bulkUUIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.UUIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'uuids' is required and must be a non-empty array"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	count, err := repo.BulkDeleteCheckDefinitions(ctx, req.UUIDs)
	if err != nil {
		logrus.Errorf("Failed to bulk delete checks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete checks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "count": count})
}
