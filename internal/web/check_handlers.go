package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"checker/internal/db"
	"checker/internal/models"
)

// ListCheckDefinitions returns all check definitions
func ListCheckDefinitions(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	defs, err := mongoDB.GetAllCheckDefinitions(ctx)
	if err != nil {
		logrus.Errorf("Failed to get check definitions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch check definitions",
		})
		return
	}

	// Convert to view models for the API
	viewModels := make([]models.CheckDefinitionViewModel, 0, len(defs))
	for _, def := range defs {
		viewModels = append(viewModels, convertToCheckDefViewModel(def))
	}

	c.JSON(http.StatusOK, viewModels)
}

// GetCheckDefinition returns a single check definition by UUID
func GetCheckDefinition(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	def, err := mongoDB.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		logrus.Errorf("Failed to get check definition %s: %v", uuid, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Check definition not found",
		})
		return
	}

	c.JSON(http.StatusOK, convertToCheckDefViewModel(def))
}

// CreateCheckDefinition creates a new check definition
func CreateCheckDefinition(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)

	var def models.CheckDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		logrus.Errorf("Failed to bind check definition: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid check definition data",
		})
		return
	}

	// Validate the check definition
	if def.Name == "" || def.Project == "" || def.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name, project, and type are required fields",
		})
		return
	}

	// Default to enabled if not specified
	if !def.Enabled {
		def.Enabled = true
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := mongoDB.CreateCheckDefinition(ctx, def)
	if err != nil {
		logrus.Errorf("Failed to create check definition: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create check definition",
		})
		return
	}

	// Update the ID and return the created check definition
	if idObj, err := primitive.ObjectIDFromHex(id); err == nil {
		def.ID = idObj
	}

	c.JSON(http.StatusCreated, convertToCheckDefViewModel(def))
}

// UpdateCheckDefinition updates an existing check definition
func UpdateCheckDefinition(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	var def models.CheckDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		logrus.Errorf("Failed to bind check definition: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid check definition data",
		})
		return
	}

	// Ensure UUID in URL matches body
	def.UUID = uuid

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get the existing definition to maintain ID
	existingDef, err := mongoDB.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		logrus.Errorf("Failed to get check definition %s: %v", uuid, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Check definition not found",
		})
		return
	}

	// Preserve the ID
	def.ID = existingDef.ID

	// Update the definition
	if err := mongoDB.UpdateCheckDefinition(ctx, def); err != nil {
		logrus.Errorf("Failed to update check definition %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update check definition",
		})
		return
	}

	c.JSON(http.StatusOK, convertToCheckDefViewModel(def))
}

// DeleteCheckDefinition deletes a check definition
func DeleteCheckDefinition(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := mongoDB.DeleteCheckDefinition(ctx, uuid); err != nil {
		logrus.Errorf("Failed to delete check definition %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete check definition",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Check definition %s deleted", uuid),
		"uuid":    uuid,
	})
}

// ToggleCheckDefinitionStatus enables or disables a check definition
func ToggleCheckDefinitionStatus(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UUID is required",
		})
		return
	}

	enabled := c.Query("enabled") == "true"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := mongoDB.ToggleCheckDefinition(ctx, uuid, enabled); err != nil {
		logrus.Errorf("Failed to toggle check definition %s: %v", uuid, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to toggle check definition",
		})
		return
	}

	// Get the updated check definition
	def, err := mongoDB.GetCheckDefinitionByUUID(ctx, uuid)
	if err != nil {
		logrus.Warnf("Check definition %s toggled but could not be retrieved: %v", uuid, err)
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Check definition %s toggled to %v", uuid, enabled),
			"uuid":    uuid,
			"enabled": enabled,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Check definition %s toggled to %v", uuid, enabled),
		"uuid":    uuid,
		"enabled": enabled,
		"check":   convertToCheckDefViewModel(def),
	})
}

// Get all projects for check definitions
func GetCheckProjects(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	projects, err := mongoDB.GetAllProjects(ctx)
	if err != nil {
		logrus.Errorf("Failed to get projects: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch projects",
		})
		return
	}

	c.JSON(http.StatusOK, projects)
}

// Get all check types
func GetCheckTypes(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	types, err := mongoDB.GetAllCheckTypes(ctx)
	if err != nil {
		logrus.Errorf("Failed to get check types: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch check types",
		})
		return
	}

	c.JSON(http.StatusOK, types)
}

// Helper to convert a CheckDefinition to a CheckDefinitionViewModel
func convertToCheckDefViewModel(def models.CheckDefinition) models.CheckDefinitionViewModel {
	return models.CheckDefinitionViewModel{
		ID:               def.ID.Hex(),
		UUID:             def.UUID,
		Name:             def.Name,
		Project:          def.Project,
		GroupName:        def.GroupName,
		Type:             def.Type,
		Description:      def.Description,
		Enabled:          def.Enabled,
		CreatedAt:        def.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        def.UpdatedAt.Format(time.RFC3339),
		Duration:         def.Duration,
		URL:              def.URL,
		Timeout:          def.Timeout,
		Host:             def.Host,
		Port:             def.Port,
		ActorType:        def.ActorType,
		AlertType:        def.AlertType,
		AlertDestination: def.AlertDestination,
	}
}
