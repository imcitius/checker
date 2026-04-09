// SPDX-License-Identifier: BUSL-1.1

package alerts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsgenieClient_Trigger(t *testing.T) {
	var receivedPayload opsgenieAlertPayload
	var receivedAuth string
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"result":"Request will be processed"}`))
	}))
	defer server.Close()

	client := &OpsgenieClient{
		APIKey: "test-api-key",
		Region: "us",
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			// Rewrite URL to point to test server
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(server.URL, "http://")
			return http.DefaultClient.Do(req)
		},
	}

	err := client.Trigger("my-check", "uuid-1234", "connection refused", "critical")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify authorization header
	if receivedAuth != "GenieKey test-api-key" {
		t.Errorf("expected auth 'GenieKey test-api-key', got %q", receivedAuth)
	}

	// Verify path
	if receivedPath != "/v2/alerts" {
		t.Errorf("expected path '/v2/alerts', got %q", receivedPath)
	}

	// Verify payload
	if receivedPayload.Message != "my-check is DOWN" {
		t.Errorf("expected message 'my-check is DOWN', got %q", receivedPayload.Message)
	}
	if receivedPayload.Alias != "uuid-1234" {
		t.Errorf("expected alias 'uuid-1234', got %q", receivedPayload.Alias)
	}
	if receivedPayload.Description != "Error: connection refused" {
		t.Errorf("expected description 'Error: connection refused', got %q", receivedPayload.Description)
	}
	if receivedPayload.Priority != "P1" {
		t.Errorf("expected priority 'P1', got %q", receivedPayload.Priority)
	}
}

func TestOpsgenieClient_Resolve(t *testing.T) {
	var receivedPayload opsgenieClosePayload
	var receivedAuth string
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"result":"Request will be processed"}`))
	}))
	defer server.Close()

	client := &OpsgenieClient{
		APIKey: "test-api-key",
		Region: "us",
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(server.URL, "http://")
			return http.DefaultClient.Do(req)
		},
	}

	err := client.Resolve("uuid-1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "GenieKey test-api-key" {
		t.Errorf("expected auth 'GenieKey test-api-key', got %q", receivedAuth)
	}

	if receivedPath != "/v2/alerts/uuid-1234/close" {
		t.Errorf("expected path '/v2/alerts/uuid-1234/close', got %q", receivedPath)
	}

	if receivedPayload.Note != "Resolved by Checker" {
		t.Errorf("expected note 'Resolved by Checker', got %q", receivedPayload.Note)
	}
}

func TestOpsgenieClient_RegionRouting(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		wantHost string
	}{
		{"US region", "us", "api.opsgenie.com"},
		{"EU region", "eu", "api.eu.opsgenie.com"},
		{"empty defaults to US", "", "api.opsgenie.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedHost string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			}))
			defer server.Close()

			client := &OpsgenieClient{
				APIKey: "test-key",
				Region: tt.region,
				HTTPDo: func(req *http.Request) (*http.Response, error) {
					receivedHost = req.URL.Host
					// Redirect to test server
					req.URL.Scheme = "http"
					req.URL.Host = strings.TrimPrefix(server.URL, "http://")
					return http.DefaultClient.Do(req)
				},
			}

			err := client.Trigger("check", "alias", "err", "critical")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedHost != tt.wantHost {
				t.Errorf("expected host %q, got %q", tt.wantHost, receivedHost)
			}
		})
	}
}

func TestMapPriority(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "P1"},
		{"warning", "P2"},
		{"info", "P3"},
		{"unknown", "P3"},
		{"", "P3"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := MapPriority(tt.severity)
			if got != tt.want {
				t.Errorf("MapPriority(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestOpsgenieClient_TriggerHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"API key is invalid"}`))
	}))
	defer server.Close()

	client := &OpsgenieClient{
		APIKey: "bad-key",
		Region: "us",
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(server.URL, "http://")
			return http.DefaultClient.Do(req)
		},
	}

	err := client.Trigger("check", "alias", "err", "critical")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to contain '403', got %q", err.Error())
	}
}

func TestNewOpsgenieAlerter_Valid(t *testing.T) {
	cfg := json.RawMessage(`{"api_key":"test-key-123","region":"eu"}`)
	a, err := NewAlerter("opsgenie", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oa, ok := a.(*OpsgenieAlerter)
	if !ok {
		t.Fatalf("expected *OpsgenieAlerter, got %T", a)
	}
	if oa.APIKey != "test-key-123" {
		t.Errorf("unexpected APIKey: %q", oa.APIKey)
	}
	if oa.Region != "eu" {
		t.Errorf("unexpected Region: %q", oa.Region)
	}
	if oa.Type() != "opsgenie" {
		t.Errorf("expected Type() 'opsgenie', got %q", oa.Type())
	}
}

func TestNewOpsgenieAlerter_MissingAPIKey(t *testing.T) {
	_, err := NewAlerter("opsgenie", json.RawMessage(`{"region":"us"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewOpsgenieAlerter_InvalidJSON(t *testing.T) {
	_, err := NewAlerter("opsgenie", json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestNewOpsgenieAlerter_DefaultRegion(t *testing.T) {
	cfg := json.RawMessage(`{"api_key":"test-key"}`)
	a, err := NewAlerter("opsgenie", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oa := a.(*OpsgenieAlerter)
	if oa.Region != "" {
		t.Errorf("expected empty Region (defaults to US), got %q", oa.Region)
	}
}

func TestOpsgenieClient_ResolveHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Alert not found"}`))
	}))
	defer server.Close()

	client := &OpsgenieClient{
		APIKey: "test-key",
		Region: "us",
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(server.URL, "http://")
			return http.DefaultClient.Do(req)
		},
	}

	err := client.Resolve("non-existent-alias")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got %q", err.Error())
	}
}
