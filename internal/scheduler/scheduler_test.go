package scheduler

import (
	"testing"

	"checker/internal/checks"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestCheckerFactory_Http verifies that CheckerFactory returns an HTTPCheck for a valid HTTP configuration.
func TestCheckerFactory_Http(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_Http")
	checkDef := models.CheckDefinition{
		ID:                  primitive.NewObjectID(),
		UUID:                "test-uuid",
		Type:                "http",
		URL:                 "https://example.com",
		Timeout:             "5s",
		Answer:              "ok",
		AnswerPresent:       true,
		Code:                []int{200},
		Headers:             []map[string]string{{"Content-Type": "application/json"}},
		SkipCheckSSL:        false,
		SSLExpirationPeriod: "720h",
		StopFollowRedirects: false,
	}
	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for HTTP check")
	}

	// Assert the returned checker is of type *checks.HTTPCheck.
	httpCheck, ok := checker.(*checks.HTTPCheck)
	if !ok {
		t.Errorf("Expected *checks.HTTPCheck, got %T", checker)
	}

	// Verify fields were set correctly
	if httpCheck.URL != "https://example.com" {
		t.Errorf("Expected URL to be 'https://example.com', got '%s'", httpCheck.URL)
	}
	if httpCheck.Timeout != "5s" {
		t.Errorf("Expected Timeout to be '5s', got '%s'", httpCheck.Timeout)
	}
	if httpCheck.Answer != "ok" {
		t.Errorf("Expected Answer to be 'ok', got '%s'", httpCheck.Answer)
	}
	if !httpCheck.SkipCheckSSL == checkDef.SkipCheckSSL {
		t.Errorf("Expected SkipCheckSSL to be %v, got %v", checkDef.SkipCheckSSL, httpCheck.SkipCheckSSL)
	}
}

// TestCheckerFactory_TCP verifies that CheckerFactory returns a TCPCheck for a valid TCP configuration.
func TestCheckerFactory_TCP(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_TCP")
	checkDef := models.CheckDefinition{
		ID:      primitive.NewObjectID(),
		UUID:    "test-uuid",
		Type:    "tcp",
		Host:    "example.com",
		Port:    80,
		Timeout: "5s",
	}
	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for TCP check")
	}

	// Assert the returned checker is of type *checks.TCPCheck.
	tcpCheck, ok := checker.(*checks.TCPCheck)
	if !ok {
		t.Errorf("Expected *checks.TCPCheck, got %T", checker)
	}

	// Verify fields were set correctly
	if tcpCheck.Host != "example.com" {
		t.Errorf("Expected Host to be 'example.com', got '%s'", tcpCheck.Host)
	}
	if tcpCheck.Port != 80 {
		t.Errorf("Expected Port to be 80, got %d", tcpCheck.Port)
	}
	if tcpCheck.Timeout != "5s" {
		t.Errorf("Expected Timeout to be '5s', got '%s'", tcpCheck.Timeout)
	}
}

// TestCheckerFactory_ICMP verifies that CheckerFactory returns an ICMPCheck for a valid ICMP configuration.
func TestCheckerFactory_ICMP(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_ICMP")

	checkDef := models.CheckDefinition{
		ID:      primitive.NewObjectID(),
		UUID:    "test-uuid",
		Type:    "icmp",
		Host:    "example.com",
		Count:   4,
		Timeout: "5s",
	}

	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for ICMP check")
	}

	// Verify the type
	icmpCheck, ok := checker.(*checks.ICMPCheck)
	if !ok {
		t.Fatal("CheckerFactory returned wrong type")
	}

	// Verify the fields
	if icmpCheck.Host != "example.com" {
		t.Errorf("Expected Host to be 'example.com', got '%s'", icmpCheck.Host)
	}
	if icmpCheck.Count != 4 {
		t.Errorf("Expected Count to be 4, got %d", icmpCheck.Count)
	}
	if icmpCheck.Timeout != "5s" {
		t.Errorf("Expected Timeout to be '5s', got '%s'", icmpCheck.Timeout)
	}
}

// TestCheckerFactory_Unknown verifies that CheckerFactory returns nil for an unknown check type.
func TestCheckerFactory_Unknown(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_Unknown")
	checkDef := models.CheckDefinition{
		ID:   primitive.NewObjectID(),
		UUID: "test-uuid",
		Type: "unsupported",
	}
	checker := CheckerFactory(checkDef, logger)
	if checker != nil {
		t.Errorf("Expected nil for unknown check type, got %T", checker)
	}
}

// TestActorFactory_Log verifies that ActorFactory returns a LogActor for a valid Log configuration.
func TestActorFactory_Log(t *testing.T) {
	checkDef := models.CheckDefinition{
		ID:        primitive.NewObjectID(),
		UUID:      "test-uuid",
		ActorType: "log",
	}
	actor, err := ActorFactory(checkDef)
	if err != nil {
		t.Fatalf("ActorFactory returned error for Log actor: %v", err)
	}
	if actor == nil {
		t.Fatal("ActorFactory returned nil for Log actor")
	}
}

// TestActorFactory_Alert verifies that ActorFactory handles alert actors correctly.
func TestActorFactory_Alert(t *testing.T) {
	testCases := []struct {
		name      string
		alertType string
		wantErr   bool
	}{
		{
			name:      "Telegram Alert",
			alertType: "telegram",
			wantErr:   true, // Currently not implemented
		},
		{
			name:      "Slack Alert",
			alertType: "slack",
			wantErr:   true, // Currently not implemented
		},
		{
			name:      "Unknown Alert Type",
			alertType: "unknown",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checkDef := models.CheckDefinition{
				ID:        primitive.NewObjectID(),
				UUID:      "test-uuid",
				ActorType: "alert",
				AlertType: tc.alertType,
			}
			actor, err := ActorFactory(checkDef)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if actor != nil {
					t.Errorf("Expected nil actor, got %T", actor)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if actor == nil {
					t.Error("Expected non-nil actor, got nil")
				}
			}
		})
	}
}

// TestActorFactory_Unknown verifies that ActorFactory returns an error for an unknown actor type.
func TestActorFactory_Unknown(t *testing.T) {
	checkDef := models.CheckDefinition{
		ID:        primitive.NewObjectID(),
		UUID:      "test-uuid",
		ActorType: "unsupported",
	}
	actor, err := ActorFactory(checkDef)
	if err == nil {
		t.Error("Expected error for unknown actor type, got nil")
	}
	if actor != nil {
		t.Errorf("Expected nil actor for unknown actor type, got %T", actor)
	}
}

// TestMergeHeaders verifies that mergeHeaders aggregates headers correctly.
func TestMergeHeaders(t *testing.T) {
	testCases := []struct {
		name        string
		headers     []map[string]string
		expectedLen int
		expectedVal string
		expectedKey string
		description string
	}{
		{
			name: "Single Header",
			headers: []map[string]string{
				{"Content-Type": "application/json"},
			},
			expectedLen: 1,
			expectedKey: "Content-Type",
			expectedVal: "application/json",
			description: "Single header should be preserved",
		},
		{
			name: "Multiple Headers",
			headers: []map[string]string{
				{"Content-Type": "application/json"},
				{"Authorization": "Bearer token"},
			},
			expectedLen: 2,
			expectedKey: "Authorization",
			expectedVal: "Bearer token",
			description: "Multiple headers should be merged",
		},
		{
			name: "Overlapping Headers",
			headers: []map[string]string{
				{"Content-Type": "text/plain"},
				{"Content-Type": "application/json"},
			},
			expectedLen: 1,
			expectedKey: "Content-Type",
			expectedVal: "application/json",
			description: "Last value should win for duplicate keys",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			merged := mergeHeaders(tc.headers)

			if len(merged) != tc.expectedLen {
				t.Errorf("Expected %d headers, got %d", tc.expectedLen, len(merged))
			}

			if val, ok := merged[tc.expectedKey]; !ok || val != tc.expectedVal {
				t.Errorf("Expected %s=%s, got %s", tc.expectedKey, tc.expectedVal, val)
			}
		})
	}
}

// TestDurationToSeconds verifies that durationToSeconds converts durations correctly.
func TestDurationToSeconds(t *testing.T) {
	testCases := []struct {
		name     string
		duration string
		want     int64
		wantErr  bool
	}{
		{"1 Minute", "1m", 60, false},
		{"1 Hour", "1h", 3600, false},
		{"Invalid Duration", "invalid", 0, true},
		{"Empty Duration", "", 0, true},
		{"Complex Duration", "1h30m", 5400, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := durationToSeconds(tc.duration)
			if (err != nil) != tc.wantErr {
				t.Errorf("durationToSeconds(%q) error = %v, wantErr %v", tc.duration, err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("durationToSeconds(%q) = %v, want %v", tc.duration, got, tc.want)
			}
		})
	}
}

// TestSecondsFromDayStart verifies that secondsFromDayStart returns a reasonable value.
func TestSecondsFromDayStart(t *testing.T) {
	seconds := secondsFromDayStart()

	// Should be between 0 and 24*60*60 (seconds in a day)
	if seconds < 0 || seconds >= 24*60*60 {
		t.Errorf("secondsFromDayStart() = %v, want value between 0 and %v", seconds, 24*60*60)
	}
}
