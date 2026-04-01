package sentry

import (
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
)

// Init initializes the Sentry SDK from environment variables.
// Returns true if Sentry was initialized (SENTRY_DSN is set), false otherwise.
func Init(release string) bool {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return false
	}

	environment := os.Getenv("SENTRY_ENVIRONMENT")
	if environment == "" {
		environment = "production"
	}

	tracesSampleRate := 0.1
	if v := os.Getenv("SENTRY_TRACES_SAMPLE_RATE"); v != "" {
		if rate, err := strconv.ParseFloat(v, 64); err == nil {
			tracesSampleRate = rate
		}
	}

	debug := false
	if v := os.Getenv("SENTRY_DEBUG"); v == "true" || v == "1" {
		debug = true
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		Release:          release,
		Debug:            debug,
		TracesSampleRate: tracesSampleRate,
	})
	if err != nil {
		logrus.Errorf("Failed to initialize Sentry: %v", err)
		return false
	}

	logrus.Infof("Sentry initialized (environment=%s, traces_sample_rate=%.2f)", environment, tracesSampleRate)
	return true
}

// Flush waits for buffered events to be sent to Sentry.
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// CaptureError reports an error to Sentry with optional tags.
// Safe to call even if Sentry is not initialized (no-ops gracefully).
func CaptureError(err error, tags map[string]string) {
	if err == nil {
		return
	}
	hub := sentry.CurrentHub()
	if hub.Client() == nil {
		return
	}
	hub.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		hub.CaptureException(err)
	})
}
