package alerts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendPagerDutyTrigger(t *testing.T) {
	var received PagerDutyEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"success","dedup_key":"test-uuid-123"}`))
	}))
	defer server.Close()

	// Override the PagerDuty URL to point to our test server
	originalURL := PagerDutyEventsURL
	PagerDutyEventsURL = server.URL
	defer func() { PagerDutyEventsURL = originalURL }()

	err := SendPagerDutyTrigger("test-routing-key", "test-uuid-123", "My Check", "connection refused", "critical")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the event
	if received.RoutingKey != "test-routing-key" {
		t.Errorf("expected routing_key 'test-routing-key', got '%s'", received.RoutingKey)
	}
	if received.EventAction != "trigger" {
		t.Errorf("expected event_action 'trigger', got '%s'", received.EventAction)
	}
	if received.DedupKey != "test-uuid-123" {
		t.Errorf("expected dedup_key 'test-uuid-123', got '%s'", received.DedupKey)
	}
	if received.Payload == nil {
		t.Fatal("expected payload to be present")
	}
	if received.Payload.Summary != "My Check is DOWN: connection refused" {
		t.Errorf("expected summary 'My Check is DOWN: connection refused', got '%s'", received.Payload.Summary)
	}
	if received.Payload.Source != "checker" {
		t.Errorf("expected source 'checker', got '%s'", received.Payload.Source)
	}
	if received.Payload.Severity != "critical" {
		t.Errorf("expected severity 'critical', got '%s'", received.Payload.Severity)
	}
}

func TestSendPagerDutyResolve(t *testing.T) {
	var received PagerDutyEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"success","dedup_key":"test-uuid-456"}`))
	}))
	defer server.Close()

	originalURL := PagerDutyEventsURL
	PagerDutyEventsURL = server.URL
	defer func() { PagerDutyEventsURL = originalURL }()

	err := SendPagerDutyResolve("test-routing-key", "test-uuid-456", "My Check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify resolve event
	if received.RoutingKey != "test-routing-key" {
		t.Errorf("expected routing_key 'test-routing-key', got '%s'", received.RoutingKey)
	}
	if received.EventAction != "resolve" {
		t.Errorf("expected event_action 'resolve', got '%s'", received.EventAction)
	}
	if received.DedupKey != "test-uuid-456" {
		t.Errorf("expected dedup_key 'test-uuid-456', got '%s'", received.DedupKey)
	}
	// Resolve events should not have a payload
	if received.Payload != nil {
		t.Errorf("expected no payload for resolve event, got %+v", received.Payload)
	}
}

func TestSendPagerDutyTrigger_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status":"invalid event","message":"routing key not found"}`))
	}))
	defer server.Close()

	originalURL := PagerDutyEventsURL
	PagerDutyEventsURL = server.URL
	defer func() { PagerDutyEventsURL = originalURL }()

	err := SendPagerDutyTrigger("bad-key", "uuid-1", "Check", "err", "critical")
	if err == nil {
		t.Fatal("expected error for bad request, got nil")
	}
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"critical", "critical"},
		{"warning", "warning"},
		{"degraded", "warning"},
		{"info", "info"},
		{"", "critical"},
		{"unknown", "critical"},
	}

	for _, tc := range tests {
		result := MapSeverity(tc.input)
		if result != tc.expected {
			t.Errorf("MapSeverity(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestNewPagerDutyAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"routing_key":"R0123456789"}`)
	a, err := NewAlerter("pagerduty", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pa, ok := a.(*PagerDutyAlerter)
	if !ok {
		t.Fatalf("expected *PagerDutyAlerter, got %T", a)
	}
	if pa.RoutingKey != "R0123456789" {
		t.Errorf("unexpected RoutingKey: %q", pa.RoutingKey)
	}
	if pa.Type() != "pagerduty" {
		t.Errorf("expected Type() 'pagerduty', got %q", pa.Type())
	}
}

func TestNewPagerDutyAlerter_MissingKey(t *testing.T) {
	_, err := NewAlerter("pagerduty", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewPagerDutyAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("pagerduty", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestSendPagerDutyTrigger_SeverityMapping(t *testing.T) {
	tests := []struct {
		severity         string
		expectedSeverity string
	}{
		{"critical", "critical"},
		{"warning", "warning"},
		{"degraded", "warning"},
		{"info", "info"},
	}

	for _, tc := range tests {
		t.Run(tc.severity, func(t *testing.T) {
			var received PagerDutyEvent

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&received)
				w.WriteHeader(http.StatusAccepted)
			}))
			defer server.Close()

			originalURL := PagerDutyEventsURL
			PagerDutyEventsURL = server.URL
			defer func() { PagerDutyEventsURL = originalURL }()

			err := SendPagerDutyTrigger("key", "uuid", "check", "msg", tc.severity)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if received.Payload.Severity != tc.expectedSeverity {
				t.Errorf("expected severity %q, got %q", tc.expectedSeverity, received.Payload.Severity)
			}
		})
	}
}
