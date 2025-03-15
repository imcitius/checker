package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
)

// Config structure to match config.yaml
type Config struct {
	DB struct {
		Protocol string `yaml:"protocol"`
		Host     string `yaml:"host"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"db"`
}

func main() {
	// Load configuration from config.yaml
	fmt.Println("Loading configuration from config.yaml...")
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Construct MongoDB URI from config
	uri := fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority",
		cfg.DB.Username,
		cfg.DB.Password,
		cfg.DB.Host)

	fmt.Printf("Connecting to MongoDB at %s...", cfg.DB.Host)
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Ping the MongoDB server to verify the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	fmt.Println("Connected to MongoDB!")

	// Get the database
	database := client.Database(cfg.DB.Database)
	fmt.Printf("Using database: %s\n", cfg.DB.Database)

	// Count documents in check_definitions collection
	count, err := database.Collection("check_definitions").CountDocuments(ctx, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to count documents: %v", err)
	}
	fmt.Printf("Number of check_status documents: %d\n", count)

	// List collection names for debugging
	collections, err := database.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to list collections: %v", err)
	}
	fmt.Println("Collections in database:")
	for _, collection := range collections {
		fmt.Printf("- %s\n", collection)
	}
}

// Load configuration from file
func loadConfig(file string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse config file: %v", err)
	}

	return &config, nil
}
