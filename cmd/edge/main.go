package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/imcitius/checker/pkg/edge"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logrus.SetLevel(logrus.InfoLevel)

	cfg := edge.ClientConfig{
		SaaSURL:    envOrFatal("CHECKER_SAAS_URL"),
		APIKey:     envOrFatal("CHECKER_API_KEY"),
		Region:     envOrDefault("CHECKER_EDGE_REGION", "edge-default"),
		MaxWorkers: envIntOrDefault("CHECKER_EDGE_WORKERS", 10),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	client := edge.NewClient(cfg)
	if err := client.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}

// envOrFatal returns the value of the environment variable or exits if not set.
func envOrFatal(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

// envOrDefault returns the value of the environment variable or a default.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// envIntOrDefault returns the integer value of the environment variable or a default.
func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			logrus.Warnf("Invalid value for %s: %q — using default %d", key, v, defaultVal)
			return defaultVal
		}
		return n
	}
	return defaultVal
}
