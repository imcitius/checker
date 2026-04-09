// SPDX-License-Identifier: BUSL-1.1

package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
)

// ListEscalationPolicies returns all escalation policies.
// GET /api/escalation-policies
func ListEscalationPolicies(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	ctx := c.Request.Context()
	policies, err := repo.GetAllEscalationPolicies(ctx)
	if err != nil {
		logrus.Errorf("Failed to get escalation policies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch escalation policies"})
		return
	}

	if policies == nil {
		policies = []models.EscalationPolicy{}
	}

	c.JSON(http.StatusOK, policies)
}

// CreateEscalationPolicy creates a new escalation policy.
// POST /api/escalation-policies
func CreateEscalationPolicy(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)

	var policy models.EscalationPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid escalation policy data"})
		return
	}

	if policy.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Policy name is required"})
		return
	}
	if len(policy.Steps) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one escalation step is required"})
		return
	}

	// Validate steps
	for i, step := range policy.Steps {
		if step.Channel == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Channel is required for each step", "step": i})
			return
		}
		if step.DelayMin < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delay_min must be >= 0", "step": i})
			return
		}
	}

	ctx := c.Request.Context()
	if err := repo.CreateEscalationPolicy(ctx, policy); err != nil {
		logrus.Errorf("Failed to create escalation policy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create escalation policy"})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// UpdateEscalationPolicy updates an existing escalation policy.
// PUT /api/escalation-policies/:name
func UpdateEscalationPolicy(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Policy name is required"})
		return
	}

	var policy models.EscalationPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid escalation policy data"})
		return
	}

	// Ensure name from URL is used
	policy.Name = name

	if len(policy.Steps) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one escalation step is required"})
		return
	}

	// Validate steps
	for i, step := range policy.Steps {
		if step.Channel == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Channel is required for each step", "step": i})
			return
		}
		if step.DelayMin < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delay_min must be >= 0", "step": i})
			return
		}
	}

	ctx := c.Request.Context()
	if err := repo.UpdateEscalationPolicy(ctx, policy); err != nil {
		logrus.Errorf("Failed to update escalation policy %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update escalation policy"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// DeleteEscalationPolicy deletes an escalation policy.
// DELETE /api/escalation-policies/:name
func DeleteEscalationPolicy(c *gin.Context) {
	repo := c.MustGet("repo").(db.Repository)
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Policy name is required"})
		return
	}

	ctx := c.Request.Context()
	if err := repo.DeleteEscalationPolicy(ctx, name); err != nil {
		logrus.Errorf("Failed to delete escalation policy %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete escalation policy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Escalation policy deleted",
		"name":    name,
	})
}
