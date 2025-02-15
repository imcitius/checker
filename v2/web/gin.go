package web

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CheckViewModel struct {
	UUID        string
	Name        string
	Project     string
	Healthcheck string
	LastResult  bool
	LastExec    string
	LastPing    string
	Enabled     bool
}

func Listen() {
	router := gin.Default()

	// Load templates
	router.SetHTMLTemplate(template.Must(template.ParseFiles("web/templates/dashboard.html")))

	// Serve static files
	router.Static("/static", "web/static")

	// Existing API endpoints
	router.GET("/ping", handlePing)
	router.GET("/ping/check/:uuid", handlePingCheck)
	router.GET("/list", handleList)

	// New Web UI routes
	router.GET("/", handleDashboard)
	router.POST("/api/toggle-check", handleToggleCheck)

	// uses 8080 by default, or PORT environment variable
	err := router.Run()
	if err != nil {
		logger.Fatalf("Failed to start web server: %v", err)
	}
}

func handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func handlePingCheck(c *gin.Context) {
	check, err := configurer.Ping(c.Param("uuid"))
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"pong": check.LastPing.String(),
		})
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err,
		})
	}
}

func handleList(c *gin.Context) {
	res, err := configurer.ListChecks()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"list": res,
	})
}

func handleDashboard(c *gin.Context) {
    checks, err := configurer.GetAllChecks()  // Use GetAllChecks() instead of ListChecks()
    if err != nil {
        c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
            "error": err.Error(),
        })
        return
    }

    viewModels := make([]CheckViewModel, 0)
    for _, check := range checks {
        viewModels = append(viewModels, CheckViewModel{
            UUID:        check.UUID,
            Name:        check.Name,
            Project:     check.Project,
            Healthcheck: check.Healthcheck,
            LastResult:  check.LastResult,
            LastExec:    check.LastExec.Format("2006-01-02 15:04:05"),
            LastPing:    check.LastPing.Format("2006-01-02 15:04:05"),
            Enabled:     check.Enabled,  // Now using the actual Enabled state
        })
    }

    c.HTML(http.StatusOK, "dashboard.html", viewModels)
}

func handleToggleCheck(c *gin.Context) {
	uuid := c.PostForm("uuid")
	enabled := c.PostForm("enabled") == "true"

	err := configurer.ToggleCheck(uuid, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}
