// SPDX-License-Identifier: BUSL-1.1

package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// GetCheckDefaults returns the current checker-wide default settings.
// GET /api/settings/check-defaults
func GetCheckDefaults(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	defaults, err := repo.GetCheckDefaults(ctx)
	if err != nil {
		logrus.Errorf("Failed to get check defaults: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch check defaults"})
		return
	}

	c.JSON(http.StatusOK, defaults)
}

// UpdateCheckDefaults saves checker-wide default settings.
// PUT /api/settings/check-defaults
func UpdateCheckDefaults(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var defaults models.CheckDefaults
	if err := c.ShouldBindJSON(&defaults); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := repo.SaveCheckDefaults(ctx, defaults); err != nil {
		logrus.Errorf("Failed to save check defaults: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save check defaults"})
		return
	}

	c.JSON(http.StatusOK, defaults)
}
