package checks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCheckTimeout_ValidDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"seconds", "5s", 5 * time.Second},
		{"minutes", "2m", 2 * time.Minute},
		{"milliseconds", "500ms", 500 * time.Millisecond},
		{"hours", "1h", time.Hour},
		{"mixed", "1m30s", 90 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseCheckTimeout(tt.input, 10*time.Second)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, d)
		})
	}
}

func TestParseCheckTimeout_EmptyReturnsDefault(t *testing.T) {
	defaultDur := 30 * time.Second
	d, err := parseCheckTimeout("", defaultDur)
	require.NoError(t, err)
	assert.Equal(t, defaultDur, d)
}

func TestParseCheckTimeout_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"garbage", "notaduration"},
		{"number without unit", "42"},
		{"just letters", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCheckTimeout(tt.input, 10*time.Second)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid timeout")
		})
	}
}

func TestParseCheckTimeout_NegativeDuration(t *testing.T) {
	_, err := parseCheckTimeout("-5s", 10*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be positive")
}

func TestParseCheckTimeout_ZeroDuration(t *testing.T) {
	_, err := parseCheckTimeout("0s", 10*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be positive")
}

func TestParseCheckTimeout_DefaultZero(t *testing.T) {
	// Empty string with zero default should return zero.
	d, err := parseCheckTimeout("", 0)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), d)
}
