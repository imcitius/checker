package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"checker/internal/config"
	"checker/internal/db"
	"checker/internal/routes"
)

type Server struct {
	config *config.Config
	db     *db.MongoDB
	engine *gin.Engine
}

// NewServer returns a new instance of the server
func NewServer(cfg *config.Config) *Server {
	// Initialize DB connection
	mongoDB, err := db.NewMongoDB(cfg)
	if err != nil {
		logrus.Fatalf("Could not initialize MongoDB: %v", err)
	}

	// Create a Gin engine
	engine := gin.Default()

	return &Server{
		config: cfg,
		db:     mongoDB,
		engine: engine,
	}
}

// Run starts the server
func (s *Server) Run() error {
	// Setup routes
	api := s.engine.Group("/api")
	users := api.Group("/users")
	routes.RegisterUserRoutes(users, s.db)

	address := fmt.Sprintf(":%s", s.config.ServerPort)
	logrus.Infof("Server listening on %s", address)
	return s.engine.Run(address)
}
