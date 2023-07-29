package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Listen() {
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	router.GET("/ping/check/:uuid", func(c *gin.Context) {
		res, err := configurer.GetCheckByUUid(c.Param("uuid"))
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"pong": res.Name,
			})
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err,
			})
		}
	})
	router.GET("/list", func(c *gin.Context) {
		res, err := configurer.ListChecks()
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"list": res,
			})
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err,
			})
		}
	})

	// uses 8080 by default, or PORT environment variable
	err := router.Run()
	if err != nil {
		logger.Fatalf("Failed to start web server: %v", err)
	}
}
