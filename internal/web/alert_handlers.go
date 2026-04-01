package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// ListAlerts returns paginated alert history with optional filters.
// GET /api/alerts?limit=50&offset=0&project=&check_uuid=&status=all
func ListAlerts(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	// Parse pagination params
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build filters
	filters := models.AlertHistoryFilters{
		Project:   c.Query("project"),
		CheckUUID: c.Query("check_uuid"),
	}

	// Handle status filter: active, resolved, all (default)
	status := c.DefaultQuery("status", "all")
	switch status {
	case "active":
		f := false
		filters.IsResolved = &f
	case "resolved":
		f := true
		filters.IsResolved = &f
	// "all" leaves IsResolved as nil (no filter)
	}

	ctx := c.Request.Context()
	alerts, total, err := repo.GetAlertHistory(ctx, limit, offset, filters)
	if err != nil {
		logrus.Errorf("Failed to get alert history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve alert history"})
		return
	}

	// Ensure we return an empty array rather than null
	if alerts == nil {
		alerts = []models.AlertEvent{}
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  total,
	})
}

// ListSilences returns all active silences.
// GET /api/silences
func ListSilences(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	ctx := c.Request.Context()
	silences, err := repo.GetActiveSilences(ctx)
	if err != nil {
		logrus.Errorf("Failed to get active silences: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve silences"})
		return
	}

	if silences == nil {
		silences = []models.AlertSilence{}
	}

	c.JSON(http.StatusOK, gin.H{
		"silences": silences,
	})
}

// createSilenceRequest represents the JSON body for creating a silence.
type createSilenceRequest struct {
	Scope    string `json:"scope" binding:"required,oneof=check project"`
	Target   string `json:"target" binding:"required"`
	Channel  string `json:"channel"`  // optional — empty = all channels
	Duration string `json:"duration" binding:"required,oneof=30m 1h 4h 8h 24h indefinite"`
	Reason   string `json:"reason"`
}

// CreateSilence creates a new alert silence.
// POST /api/silences
func CreateSilence(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var req createSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine who created the silence
	silencedBy := c.GetString("user_email")
	if silencedBy == "" {
		silencedBy = c.GetString("user_name")
	}
	if silencedBy == "" {
		silencedBy = "ui"
	}

	// Compute expires_at from duration
	var expiresAt *time.Time
	if req.Duration != "indefinite" {
		d, err := time.ParseDuration(req.Duration)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration"})
			return
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	silence := models.AlertSilence{
		Scope:      req.Scope,
		Target:     req.Target,
		Channel:    req.Channel,
		SilencedBy: silencedBy,
		ExpiresAt:  expiresAt,
		Reason:     req.Reason,
		Active:     true,
	}

	ctx := c.Request.Context()
	if err := repo.CreateSilence(ctx, silence); err != nil {
		logrus.Errorf("Failed to create silence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create silence"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Silence created",
		"silence": silence,
	})
}

// DeleteSilence deactivates a silence by ID.
// DELETE /api/silences/:id
func DeleteSilence(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid silence ID"})
		return
	}

	ctx := c.Request.Context()
	if err := repo.DeactivateSilenceByID(ctx, id); err != nil {
		logrus.Errorf("Failed to deactivate silence %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate silence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Silence deactivated",
	})
}
