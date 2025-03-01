package checks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHTTPCheck_Success tests a basic HTTP check success.
func TestHTTPCheck_Success(t *testing.T) {
	// Create a server that always returns 200 and a simple body.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL: ts.URL,
		// Defaults will be applied (Timeout "5s", Code [200], etc).
	}
	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got failure: %s", err)
	}
	if duration <= 0 {
		t.Errorf("Expected non-zero duration, got: %v", duration)
	}
}

// TestHTTPCheck_FailureStatusCode tests that non-allowed response codes cause a failure.
func TestHTTPCheck_FailureStatusCode(t *testing.T) {
	// Create a server that returns 404.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		ErrorHeader: "TestError: ",
		// Allowed code defaults to 200.
	}
	_, err := check.Run()
	if err == nil {
		t.Errorf("Expected failure due to incorrect status code, but got success")
	}
	if !strings.Contains(err.Error(), "HTTP check failed with status 404") {
		t.Errorf("Unexpected error message for bad status: %s", err)
	}
}

// TestHTTPCheck_AnswerPresent_Success tests that a regex answer is found when expected.
func TestHTTPCheck_AnswerPresent_Success(t *testing.T) {
	// Create a server that returns a body that should match the regex.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The quick brown fox."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Answer:      "quick.*fox",
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected answer pattern match to pass but got failure: %s", err)
	}
}

// TestHTTPCheck_AnswerPresent_Failure tests that missing expected answer results in failure.
func TestHTTPCheck_AnswerPresent_Failure(t *testing.T) {
	// Create a server that returns a body that won't match the regex.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The lazy dog."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Answer:      "Hello",
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Errorf("Expected failure because answer pattern was not found")
	}
	if !strings.Contains(err.Error(), "expected pattern 'Hello' not found") {
		t.Errorf("Unexpected error message for answer check: %s", err)
	}
}

// TestHTTPCheck_AnswerAbsent_Success tests that if Answer is empty, check succeeds
func TestHTTPCheck_AnswerAbsent_Success(t *testing.T) {
	// Create a server that returns a body without the pattern.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The lazy dog."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Answer:      "",
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected success because answer should be absent, got failure: %s", err)
	}
}

// TestHTTPCheck_AnswerAbsent_Failure is actually not applicable if the HTTP check only checks for pattern presence
// This test is modified to test something else useful - checking that an invalid regex pattern fails
func TestHTTPCheck_AnswerAbsent_Failure(t *testing.T) {
	// Create a server that returns a simple body
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The lazy dog."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Answer:      "[invalid regex",  // This is an invalid regex pattern
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Errorf("Expected failure because the regex pattern is invalid")
	}
	if !strings.Contains(err.Error(), "error processing answer regex") {
		t.Errorf("Unexpected error message: %s", err)
	}
}

// TestHTTPCheck_CustomHeaders tests that custom headers are correctly added.
func TestHTTPCheck_CustomHeaders(t *testing.T) {
	// Create a server that checks for a specific header.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just check the header's value and return OK
		if r.Header.Get("X-Test") != "HeaderValue" {
			t.Errorf("Expected header X-Test with value HeaderValue, got: %s", r.Header.Get("X-Test"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Headers:     []map[string]string{{"X-Test": "HeaderValue"}},
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected success due to proper header set, but got failure: %s", err)
	}
}

// TestHTTPCheck_StopFollowRedirects tests that redirect following stops when requested.
func TestHTTPCheck_StopFollowRedirects(t *testing.T) {
	// Create a server that always issues a redirect.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/redirected")
		w.WriteHeader(http.StatusFound) // 302 redirect
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:                 ts.URL,
		StopFollowRedirects: true,
		ErrorHeader:         "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Errorf("Expected failure because redirects are stopped, but got success")
	}
	if !strings.Contains(err.Error(), "redirect not allowed") {
		t.Errorf("Unexpected error message for redirect stop: %s", err)
	}
}

// TestHTTPCheck_RedirectFollow tests that redirects are properly followed when not stopped.
func TestHTTPCheck_RedirectFollow(t *testing.T) {
	// Create a server with two endpoints: one that redirects and one final target.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
		} else if r.URL.Path == "/target" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Final Destination"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:                 ts.URL + "/redirect",
		StopFollowRedirects: false,
		ErrorHeader:         "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected success because redirect should be followed, but got failure: %s", err)
	}
}

// TestHTTPCheck_Timeout tests that the specified timeout is enforced.
func TestHTTPCheck_Timeout(t *testing.T) {
	// Create a server that delays its response.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Delayed response"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Timeout:     "1s", // Set a timeout of 1 second.
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Errorf("Expected failure due to timeout, but check succeeded")
	}
	if !strings.Contains(err.Error(), "HTTP error:") {
		t.Errorf("Unexpected error message for timeout: %s", err)
	}
}

// TestHTTPCheck_InvalidTimeout tests that an invalid timeout string produces an error.
func TestHTTPCheck_InvalidTimeout(t *testing.T) {
	check := HTTPCheck{
		URL:         "http://example.com",
		Timeout:     "notaduration",
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Errorf("Expected failure due to invalid timeout, but got success")
	}
	if !strings.Contains(err.Error(), "invalid timeout value") {
		t.Errorf("Unexpected error message for invalid timeout: %s", err)
	}
}

// TestHTTPCheck_InvalidSSLExpirationPeriod tests that an invalid SSL expiration period produces an error.
func TestHTTPCheck_InvalidSSLExpirationPeriod(t *testing.T) {
	// Use a TLS server so that SSL checks are performed.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TLS Test"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:                 ts.URL,
		SSLExpirationPeriod: "invalidPeriod",
		ErrorHeader:         "TestError: ",
		TlsConfig:           ts.Client().Transport.(*http.Transport).TLSClientConfig,
	}
	_, err := check.Run()
	if err == nil {
		t.Errorf("Expected failure due to invalid SSL expiration period, but check succeeded")
	}
	if !strings.Contains(err.Error(), "invalid SSL expiration period") {
		t.Errorf("Unexpected error message for invalid SSL period: %s", err)
	}
}

// TestHTTPCheck_TLS_Success tests a successful TLS check.
func TestHTTPCheck_TLS_Success(t *testing.T) {
	// Use a TLS server.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TLS Test Successful"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:                 ts.URL,
		SkipCheckSSL:        false,
		SSLExpirationPeriod: "720h",
		ErrorHeader:         "TestError: ",
		TlsConfig:           ts.Client().Transport.(*http.Transport).TLSClientConfig,
	}
	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected TLS check to succeed, but got failure: %s", err)
	}
}

// TestHTTPCheck_TLS_Skip tests that skipping the TLS check works.
func TestHTTPCheck_TLS_Skip(t *testing.T) {
	// Use a TLS server.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TLS Skip Test Successful"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:          ts.URL,
		SkipCheckSSL: true, // Skip TLS check.
		ErrorHeader:  "TestError: ",
		TlsConfig:    ts.Client().Transport.(*http.Transport).TLSClientConfig,
	}
	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected TLS skip check to succeed, but got failure: %s", err)
	}
}
