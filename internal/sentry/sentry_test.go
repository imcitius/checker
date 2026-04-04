package sentry

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit_NoDSN(t *testing.T) {
	os.Unsetenv("SENTRY_DSN")
	result := Init("v1.0.0")
	assert.False(t, result, "Init should return false when SENTRY_DSN is not set")
}

func TestInit_WithDSN(t *testing.T) {
	// Use a valid-looking DSN format so the Sentry SDK accepts it.
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	t.Setenv("SENTRY_ENVIRONMENT", "testing")
	t.Setenv("SENTRY_TRACES_SAMPLE_RATE", "0.5")
	t.Setenv("SENTRY_DEBUG", "false")

	result := Init("v1.0.0-test")
	assert.True(t, result, "Init should return true with a valid DSN")
}

func TestInit_DefaultEnvironment(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	os.Unsetenv("SENTRY_ENVIRONMENT")

	result := Init("v1.0.0-test")
	assert.True(t, result, "Init should succeed with default environment")
}

func TestInit_InvalidTracesSampleRate(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	t.Setenv("SENTRY_TRACES_SAMPLE_RATE", "not-a-number")

	// Should still init successfully, falling back to default rate.
	result := Init("v1.0.0-test")
	assert.True(t, result)
}

func TestInit_DebugModes(t *testing.T) {
	tests := []struct {
		name     string
		debugVal string
	}{
		{"debug_true", "true"},
		{"debug_one", "1"},
		{"debug_false", "false"},
		{"debug_empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
			if tt.debugVal != "" {
				t.Setenv("SENTRY_DEBUG", tt.debugVal)
			} else {
				os.Unsetenv("SENTRY_DEBUG")
			}
			result := Init("v1.0.0-test")
			assert.True(t, result)
		})
	}
}

func TestInit_InvalidDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "not-a-valid-dsn")

	result := Init("v1.0.0-test")
	assert.False(t, result, "Init should return false for an invalid DSN")
}

func TestCaptureError_NilError(t *testing.T) {
	// Should not panic when called with nil error.
	CaptureError(nil, nil)
}

func TestCaptureError_NoClient(t *testing.T) {
	// CaptureError should gracefully no-op when no Sentry client is initialized.
	// Ensure no DSN is set so hub has no client.
	os.Unsetenv("SENTRY_DSN")
	Init("v1.0.0-test")

	CaptureError(errors.New("test error"), map[string]string{"key": "value"})
}

func TestCaptureError_WithClient(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	Init("v1.0.0-test")

	// Should not panic; we can't easily verify the event was sent without
	// a transport mock, but we verify no crash with tags.
	CaptureError(errors.New("test error"), map[string]string{
		"check_id": "abc-123",
		"severity": "critical",
	})
}

func TestCaptureError_NilTags(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	Init("v1.0.0-test")

	CaptureError(errors.New("test error"), nil)
}

func TestFlush(t *testing.T) {
	// Flush should not panic even without initialization.
	Flush(0)
}
