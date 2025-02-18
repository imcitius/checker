package scheduler

import (
	"testing"

	"checker/internal/config"
	"checker/internal/checks"
)

// TestCheckerFactory_Http verifies that CheckerFactory returns an HTTPCheck for a valid HTTP configuration.
func TestCheckerFactory_Http(t *testing.T) {
	cfg := config.CheckConfig{
		Type:          "http",
		URL:           "https://example.com",
		Timeout:       "5s",
		Answer:        "ok",
		AnswerPresent: true,
		Code:          []int{200},
		Headers:       []map[string]string{{"Content-Type": "application/json"}},
		SkipCheckSSL:        false,
		SSLExpirationPeriod: "720h",
		StopFollowRedirects: false,
	}
	checker := CheckerFactory(cfg)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for HTTP check")
	}

	// Assert the returned checker is of type *checks.HTTPCheck.
	if _, ok := checker.(*checks.HTTPCheck); !ok {
		t.Errorf("Expected *checks.HTTPCheck, got %T", checker)
	}
}

// TestCheckerFactory_Unknown verifies that CheckerFactory returns nil for an unknown check type.
func TestCheckerFactory_Unknown(t *testing.T) {
	cfg := config.CheckConfig{
		Type: "unsupported",
	}
	checker := CheckerFactory(cfg)
	if checker != nil {
		t.Errorf("Expected nil for unknown check type, got %T", checker)
	}
}

// TestMergeHeaders verifies that mergeHeaders aggregates headers correctly.
func TestMergeHeaders(t *testing.T) {
	headersSlice := []map[string]string{
		{"A": "ValueA"},
		{"B": "ValueB"},
	}
	merged := mergeHeaders(headersSlice)
	if merged["A"] != "ValueA" || merged["B"] != "ValueB" {
		t.Errorf("mergeHeaders failed, got: %v", merged)
	}
} 