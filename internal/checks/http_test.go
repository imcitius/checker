package checks

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHTTPCheck_Success tests a basic HTTP check success.
func TestHTTPCheck_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:     ts.URL,
		Timeout: "5s",
		Code:    []int{200},
	}
	duration, err := check.Run()
	if err != nil {
		t.Fatalf("Expected success but got failure: %s", err)
	}
	if duration <= 0 {
		t.Errorf("Expected positive duration, got: %v", duration)
	}
}

// TestHTTPCheck_MultipleAllowedCodes tests success with multiple allowed status codes.
func TestHTTPCheck_MultipleAllowedCodes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted) // 202
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:     ts.URL,
		Timeout: "5s",
		Code:    []int{200, 201, 202},
	}
	_, err := check.Run()
	if err != nil {
		t.Fatalf("Expected success with status 202, got error: %s", err)
	}
}

// TestHTTPCheck_FailureStatusCode tests that non-allowed response codes cause a failure.
func TestHTTPCheck_FailureStatusCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Code:        []int{200},
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("Expected failure due to incorrect status code, but got success")
	}
	expectedErr := "HTTP check failed with status 404"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing %q, got %q", expectedErr, err.Error())
	}
}

// TestHTTPCheck_AnswerPresent_Success tests that a regex answer is found when expected.
func TestHTTPCheck_AnswerPresent_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The quick brown fox."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "quick.*fox",
		AnswerPresent: true,
		ErrorHeader:   "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Fatalf("Expected answer pattern match to pass but got failure: %s", err)
	}
}

// TestHTTPCheck_AnswerPresent_Failure tests that missing expected answer results in failure.
func TestHTTPCheck_AnswerPresent_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The lazy dog."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "Hello",
		AnswerPresent: true,
		ErrorHeader:   "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("Expected failure because answer pattern was not found")
	}
	expectedErr := "expected pattern 'Hello' not found"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing %q, got %q", expectedErr, err.Error())
	}
}

// TestHTTPCheck_AnswerAbsent_Success tests that if Answer is empty, check succeeds
func TestHTTPCheck_AnswerAbsent_Success(t *testing.T) {
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
		t.Fatalf("Expected success because answer should be absent, got failure: %s", err)
	}
}

// TestHTTPCheck_InvalidRegex tests that an invalid regex pattern fails
func TestHTTPCheck_InvalidRegex(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The lazy dog."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "[invalid regex",
		AnswerPresent: true,
		ErrorHeader:   "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("Expected failure because the regex pattern is invalid")
	}
	expectedErr := "error processing answer regex"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing %q, got %q", expectedErr, err.Error())
	}
}

// TestHTTPCheck_AnswerAbsent_PatternNotFound_Success tests that when AnswerPresent is false
// and the pattern is NOT found in the response, the check succeeds.
func TestHTTPCheck_AnswerAbsent_PatternNotFound_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The lazy dog."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "fox",
		AnswerPresent: false,
		ErrorHeader:   "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Fatalf("Expected success because pattern should be absent, got failure: %s", err)
	}
}

// TestHTTPCheck_AnswerAbsent_PatternFound_Failure tests that when AnswerPresent is false
// and the pattern IS found in the response, the check fails.
func TestHTTPCheck_AnswerAbsent_PatternFound_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The quick brown fox."))
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:           ts.URL,
		Answer:        "fox",
		AnswerPresent: false,
		ErrorHeader:   "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("Expected failure because pattern was found but should be absent")
	}
	expectedErr := "unexpected pattern 'fox' found in response"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing %q, got %q", expectedErr, err.Error())
	}
}

// TestHTTPCheck_CustomHeaders tests that custom headers are correctly added.
func TestHTTPCheck_CustomHeaders(t *testing.T) {
	headerChecked := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("X-Test"); v != "HeaderValue" {
			t.Errorf("Expected header X-Test=HeaderValue, got %q", v)
		}
		if v := r.Header.Get("Authorization"); v != "Bearer token123" {
			t.Errorf("Expected header Authorization=Bearer token123, got %q", v)
		}
		headerChecked = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL: ts.URL,
		Headers: []map[string]string{
			{"X-Test": "HeaderValue"},
			{"Authorization": "Bearer token123"},
		},
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Fatalf("Expected success, got error: %s", err)
	}
	if !headerChecked {
		t.Error("Headers were not checked in the test server")
	}
}

// TestHTTPCheck_StopFollowRedirects tests that redirect following stops when requested.
func TestHTTPCheck_StopFollowRedirects(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:                 ts.URL,
		StopFollowRedirects: true,
		ErrorHeader:         "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("Expected failure because redirects are stopped, but got success")
	}
	expectedErr := "redirect not allowed"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing %q, got %q", expectedErr, err.Error())
	}
}

// TestHTTPCheck_RedirectFollow tests that redirects are properly followed when not stopped.
func TestHTTPCheck_RedirectFollow(t *testing.T) {
	redirectCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			redirectCount++
			http.Redirect(w, r, "/middle", http.StatusFound)
		} else if r.URL.Path == "/middle" {
			redirectCount++
			http.Redirect(w, r, "/final", http.StatusFound)
		} else if r.URL.Path == "/final" {
			redirectCount++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Final Destination"))
		}
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:                 ts.URL + "/start",
		StopFollowRedirects: false,
		ErrorHeader:         "TestError: ",
	}
	_, err := check.Run()
	if err != nil {
		t.Fatalf("Expected success, got error: %s", err)
	}
	if redirectCount != 3 {
		t.Errorf("Expected 3 redirects, got %d", redirectCount)
	}
}

// TestHTTPCheck_Timeout tests that the specified timeout is enforced.
func TestHTTPCheck_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:         ts.URL,
		Timeout:     "1s",
		ErrorHeader: "TestError: ",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("Expected failure due to timeout, but check succeeded")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %s", err)
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
		t.Fatal("Expected failure due to invalid timeout, but got success")
	}
	expectedErr := "invalid timeout value"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing %q, got %q", expectedErr, err.Error())
	}
}

// TestHTTPCheck_TLS_Success tests a successful TLS check.
func TestHTTPCheck_TLS_Success(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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
		t.Fatalf("Expected TLS check to succeed, got error: %s", err)
	}
}

// TestHTTPCheck_TLS_Skip tests that skipping the TLS check works.
func TestHTTPCheck_TLS_Skip(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:          ts.URL,
		SkipCheckSSL: true,
		ErrorHeader:  "TestError: ",
		TlsConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	_, err := check.Run()
	if err != nil {
		t.Fatalf("Expected TLS skip check to succeed, got error: %s", err)
	}
}

// TestHTTPCheck_TLS_ExpiredCert tests handling of expired certificates.
func TestHTTPCheck_TLS_ExpiredCert(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	check := HTTPCheck{
		URL:                 ts.URL,
		SkipCheckSSL:        false,
		SSLExpirationPeriod: "0s", // Immediate expiration
		ErrorHeader:         "TestError: ",
		TlsConfig:           ts.Client().Transport.(*http.Transport).TLSClientConfig,
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("Expected failure due to SSL expiration check, but got success")
	}
	if !strings.Contains(err.Error(), "SSL certificate will expire") {
		t.Errorf("Expected SSL expiration error, got: %s", err)
	}
}
