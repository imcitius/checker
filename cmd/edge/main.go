// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/imcitius/checker/internal/sentry"
	"github.com/imcitius/checker/pkg/edge"
	"github.com/sirupsen/logrus"
)

// Version is injected at build time via -ldflags.
// It defaults to "dev" so that local/unversioned builds are distinguishable.
var Version = "dev"

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logLevel := logrus.InfoLevel
	if levelStr := os.Getenv("CHECKER_LOG_LEVEL"); levelStr != "" {
		parsed, err := logrus.ParseLevel(levelStr)
		if err != nil {
			logrus.Warnf("EdgeClient: invalid CHECKER_LOG_LEVEL %q, falling back to info", levelStr)
		} else {
			logLevel = parsed
		}
	}
	logrus.SetLevel(logLevel)
	logrus.Infof("EdgeClient: log level set to %s", logLevel)

	if sentry.Init(Version) {
		defer sentry.Flush(2 * time.Second)
	}

	if v := os.Getenv("CHECKER_EDGE_HEARTBEAT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			edge.SetHeartbeatInterval(d)
		} else {
			logrus.Warnf("Invalid CHECKER_EDGE_HEARTBEAT_INTERVAL %q, using default: %v", v, err)
		}
	}

	cfg := edge.ClientConfig{
		SaaSURL:    envOrDefault("CHECKER_SAAS_URL", "wss://app.ensafely.com/ws/edge"),
		APIKey:     envOrFatal("CHECKER_API_KEY"),
		Region:     envOrDefault("CHECKER_EDGE_REGION", "edge-default"),
		MaxWorkers: envIntOrDefault("CHECKER_EDGE_WORKERS", 10),
		HealthPort: envOrDefault("CHECKER_EDGE_HEALTH_PORT", "9091"),
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
