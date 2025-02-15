package web

import (
    "context"
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
    router.PUT("/checks/enable/:id", func(c *gin.Context) {
        checkID := c.Param("id")
        if err := setCheckEnabled(mongoDB, checkID, true); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "Check enabled"})
    })

    router.PUT("/checks/disable/:id", func(c *gin.Context) {
        checkID := c.Param("id")
        if err := setCheckEnabled(mongoDB, checkID, false); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "Check disabled"})
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

func setCheckEnabled(mongoDB *db.MongoDB, checkID string, enabled bool) error {
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