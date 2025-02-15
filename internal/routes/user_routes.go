package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"checker/internal/db"
	"checker/internal/models"
)

type UserRoutes struct {
	DB *db.MongoDB
}

// RegisterUserRoutes attaches user-related endpoints to the router group
func RegisterUserRoutes(router *gin.RouterGroup, dbConn *db.MongoDB) {
	ur := &UserRoutes{DB: dbConn}

	router.POST("/", ur.CreateUser)
	router.GET("/", ur.GetAllUsers)
	router.GET("/:id", ur.GetUser)
	router.PUT("/:id", ur.UpdateUser)
	router.DELETE("/:id", ur.DeleteUser)
}

// CreateUser godoc
func (ur *UserRoutes) CreateUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logrus.Errorf("Failed to bind user data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	res, err := ur.DB.Database.Collection("users").InsertOne(ctx, user)
	if err != nil {
		logrus.Errorf("Failed to insert user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert user"})
		return
	}
	user.ID = res.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, user)
}

// GetAllUsers godoc
func (ur *UserRoutes) GetAllUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := ur.DB.Database.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		logrus.Errorf("Failed to find users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		logrus.Errorf("Failed to decode users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser godoc
func (ur *UserRoutes) GetUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		logrus.Errorf("Invalid user ID: %v", id)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	err = ur.DB.Database.Collection("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		logrus.Errorf("User not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
func (ur *UserRoutes) UpdateUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		logrus.Errorf("Invalid user ID: %v", id)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logrus.Errorf("Failed to bind user data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"name":  user.Name,
			"email": user.Email,
		},
	}

	_, err = ur.DB.Database.Collection("users").UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		logrus.Errorf("Failed to update user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser godoc
func (ur *UserRoutes) DeleteUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		logrus.Errorf("Invalid user ID: %v", id)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	_, err = ur.DB.Database.Collection("users").DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		logrus.Errorf("Failed to delete user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
