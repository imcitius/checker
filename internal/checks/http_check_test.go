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
	ok, msg := check.Run()
	if !ok {
		t.Errorf("Expected success but got failure: %s", msg)
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
	ok, msg := check.Run()
	if ok {
		t.Errorf("Expected failure due to incorrect status code, but got success: %s", msg)
	}
	if !strings.Contains(msg, "HTTP check failed with status 404") {
		t.Errorf("Unexpected error message for bad status: %s", msg)
	}
}

// TestHTTPCheck_AnswerPresent_Success tests that a regex answer is found when expected.
func TestHTTPCheck_AnswerPresent_Success(t *testing.T) {
	// Create server returning a body containing "Hello".
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello world"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "Hello",
		AnswerPresent: true,
		ErrorHeader:   "TestError: ",
	}
	ok, msg := check.Run()
	if !ok {
		t.Errorf("Expected answer pattern match to pass but got failure: %s", msg)
	}
}

// TestHTTPCheck_AnswerPresent_Failure tests that missing expected answer results in failure.
func TestHTTPCheck_AnswerPresent_Failure(t *testing.T) {
	// Server returns a body without the expected pattern.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Goodbye world"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "Hello",
		AnswerPresent: true,
		ErrorHeader:   "TestError: ",
	}
	ok, msg := check.Run()
	if ok {
		t.Errorf("Expected failure because answer pattern was not found")
	}
	if !strings.Contains(msg, "expected pattern 'Hello' not found") {
		t.Errorf("Unexpected error message for answer check: %s", msg)
	}
}

// TestHTTPCheck_AnswerAbsent_Success tests that if AnswerPresent is false and the pattern is not found then success.
func TestHTTPCheck_AnswerAbsent_Success(t *testing.T) {
	// Server returns a body that does not include the pattern.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Goodbye world"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "Hello",
		AnswerPresent: false,
		ErrorHeader:   "TestError: ",
	}
	ok, msg := check.Run()
	if !ok {
		t.Errorf("Expected success because answer should be absent, got failure: %s", msg)
	}
}

// TestHTTPCheck_AnswerAbsent_Failure tests that if AnswerPresent is false but pattern is found then failure.
func TestHTTPCheck_AnswerAbsent_Failure(t *testing.T) {
	// Server returns a body that includes the pattern.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello world"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "Hello",
		AnswerPresent: false,
		ErrorHeader:   "TestError: ",
	}
	ok, msg := check.Run()
	if ok {
		t.Errorf("Expected failure because answer should not be present, but check succeeded: %s", msg)
	}
	if !strings.Contains(msg, "was found in response but should be absent") {
		t.Errorf("Unexpected error message: %s", msg)
	}
}

// TestHTTPCheck_CustomHeaders tests that custom headers are correctly added.
func TestHTTPCheck_CustomHeaders(t *testing.T) {
	// Create server that expects a custom header "X-Test" with value "HeaderValue".
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") == "HeaderValue" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Header OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Missing header"))
		}
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Headers:     map[string]string{"X-Test": "HeaderValue"},
		ErrorHeader: "TestError: ",
	}
	ok, msg := check.Run()
	if !ok {
		t.Errorf("Expected success due to proper header set, but got failure: %s", msg)
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
	ok, msg := check.Run()
	if ok {
		t.Errorf("Expected failure because redirects are stopped, but got success: %s", msg)
	}
	if !strings.Contains(msg, "redirect not allowed") {
		t.Errorf("Unexpected error message for redirect stop: %s", msg)
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
	ok, msg := check.Run()
	if !ok {
		t.Errorf("Expected success because redirect should be followed, but got failure: %s", msg)
	}
	if !strings.Contains(msg, "HTTP check passed") {
		t.Errorf("Unexpected success message: %s", msg)
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
	ok, msg := check.Run()
	if ok {
		t.Errorf("Expected failure due to timeout, but check succeeded: %s", msg)
	}
	if !strings.Contains(msg, "HTTP error:") {
		t.Errorf("Unexpected error message for timeout: %s", msg)
	}
}

// TestHTTPCheck_InvalidTimeout tests that an invalid timeout string produces an error.
func TestHTTPCheck_InvalidTimeout(t *testing.T) {
	check := HTTPCheck{
		URL:         "http://example.com",
		Timeout:     "notaduration",
		ErrorHeader: "TestError: ",
	}
	ok, msg := check.Run()
	if ok {
		t.Errorf("Expected failure due to invalid timeout, but got success")
	}
	if !strings.Contains(msg, "invalid timeout value") {
		t.Errorf("Unexpected error message for invalid timeout: %s", msg)
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
	ok, msg := check.Run()
	if ok {
		t.Errorf("Expected failure due to invalid SSL expiration period, but check succeeded: %s", msg)
	}
	if !strings.Contains(msg, "invalid SSL expiration period") {
		t.Errorf("Unexpected error message for invalid SSL period: %s", msg)
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
	ok, msg := check.Run()
	if !ok {
		t.Errorf("Expected TLS check to succeed, but got failure: %s", msg)
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
	ok, msg := check.Run()
	if !ok {
		t.Errorf("Expected TLS skip check to succeed, but got failure: %s", msg)
	}
}
