package web

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/models"
)

func RunServer(cfg *config.Config, mongoDB *db.MongoDB) error {
	router := gin.Default()

	// Add MongoDB to context
	router.Use(func(c *gin.Context) {
		c.Set("mongodb", mongoDB)
		c.Next()
	})

	router.SetHTMLTemplate(template.Must(template.ParseFiles("internal/web/templates/dashboard.html")))

	// Serve static files
	router.Static("/static", "internal/web/static")

	// New Web UI routes
	router.GET("/", handleDashboard)

	// Web UI routes
	router.GET("/checks", func(c *gin.Context) {
		statuses, err := getAllCheckStatuses(mongoDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, statuses)
	})

	// Enable/disable a check
	router.POST("/api/toggle-check", func(c *gin.Context) {
		uuid := c.PostForm("uuid")
		enabled := c.PostForm("enabled") == "true"

		if err := toggleCheck(mongoDB, uuid, enabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Check status updated"})
	})

	address := ":8080"
	logrus.Infof("Starting web server on %s", address)
	return router.Run(address)
}

func getAllCheckStatuses(mongoDB *db.MongoDB) ([]models.CheckStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := mongoDB.Database.Collection("check_statuses").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.CheckStatus
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func toggleCheck(mongoDB *db.MongoDB, checkID string, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(checkID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": bson.M{"is_enabled": enabled}}
	_, err = mongoDB.Database.Collection("check_statuses").UpdateOne(ctx, filter, update)
	return err
}

func handleDashboard(c *gin.Context) {
	mongoDB := c.MustGet("mongodb").(*db.MongoDB)

	checks, err := getAllCheckStatuses(mongoDB)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	viewModels := make([]models.CheckViewModel, 0)
	for _, check := range checks {
		viewModels = append(viewModels, models.CheckViewModel{
			ID:          check.ID.Hex(),
			Name:        check.CheckName,
			Project:     check.Project,
			Healthcheck: check.CheckGroup,
			LastResult:  check.IsHealthy,
			LastExec:    check.LastRun.Format("2006-01-02 15:04:05"),
			LastPing:    check.LastAlertSent.Format("2006-01-02 15:04:05"),
			Enabled:     check.IsEnabled,
		})
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"checks": viewModels,
	})
}
