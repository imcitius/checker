package checks

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestDNSCheck_ARecord tests A record lookup for google.com.
func TestDNSCheck_ARecord(t *testing.T) {
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "A",
		Timeout:    "10s",
		Logger:     logrus.WithField("test", "TestDNSCheck_ARecord"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestDNSCheck_ARecordDefault tests that record type defaults to A.
func TestDNSCheck_ARecordDefault(t *testing.T) {
	check := DNSCheck{
		Domain:  "google.com",
		Timeout: "10s",
		Logger:  logrus.WithField("test", "TestDNSCheck_ARecordDefault"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestDNSCheck_MXRecord tests MX record lookup.
func TestDNSCheck_MXRecord(t *testing.T) {
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "MX",
		Timeout:    "10s",
		Logger:     logrus.WithField("test", "TestDNSCheck_MXRecord"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestDNSCheck_NonExistentDomain tests failure when domain doesn't exist.
func TestDNSCheck_NonExistentDomain(t *testing.T) {
	check := DNSCheck{
		Domain:     "this-domain-absolutely-does-not-exist-12345.invalid",
		RecordType: "A",
		Timeout:    "5s",
		Logger:     logrus.WithField("test", "TestDNSCheck_NonExistentDomain"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for non-existent domain but got success")
	}
	if !strings.Contains(err.Error(), "dns lookup failed") {
		t.Errorf("Expected DNS lookup failed error, got: %v", err)
	}
}

// TestDNSCheck_ExpectedMatch tests the Expected field matching.
func TestDNSCheck_ExpectedMatch(t *testing.T) {
	// google.com MX records contain "google"
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "MX",
		Timeout:    "10s",
		Expected:   "google",
		Logger:     logrus.WithField("test", "TestDNSCheck_ExpectedMatch"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestDNSCheck_ExpectedMismatch tests that Expected field causes failure when not matched.
func TestDNSCheck_ExpectedMismatch(t *testing.T) {
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "A",
		Timeout:    "10s",
		Expected:   "this-value-will-not-match-any-ip",
		Logger:     logrus.WithField("test", "TestDNSCheck_ExpectedMismatch"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for mismatched expected value but got success")
	}
	if !strings.Contains(err.Error(), "expected value") {
		t.Errorf("Expected 'expected value' error, got: %v", err)
	}
}

// TestDNSCheck_EmptyDomain tests handling of empty domain.
func TestDNSCheck_EmptyDomain(t *testing.T) {
	check := DNSCheck{
		Domain:     "",
		RecordType: "A",
		Timeout:    "5s",
		Logger:     logrus.WithField("test", "TestDNSCheck_EmptyDomain"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for empty domain but got success")
	}
	if err.Error() != ErrDNSEmptyDomain {
		t.Errorf("Expected error message '%s', got '%s'", ErrDNSEmptyDomain, err.Error())
	}
}

// TestDNSCheck_InvalidTimeout tests handling of invalid timeout values.
func TestDNSCheck_InvalidTimeout(t *testing.T) {
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "A",
		Timeout:    "invalid",
		Logger:     logrus.WithField("test", "TestDNSCheck_InvalidTimeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid timeout but got success")
	}
}

// TestDNSCheck_InvalidRecordType tests handling of unsupported record types.
func TestDNSCheck_InvalidRecordType(t *testing.T) {
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "INVALID",
		Timeout:    "5s",
		Logger:     logrus.WithField("test", "TestDNSCheck_InvalidRecordType"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid record type but got success")
	}
	if !strings.Contains(err.Error(), "unsupported DNS record type") {
		t.Errorf("Expected unsupported record type error, got: %v", err)
	}
}

// TestDNSCheck_TXTRecord tests TXT record lookup.
func TestDNSCheck_TXTRecord(t *testing.T) {
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "TXT",
		Timeout:    "10s",
		Logger:     logrus.WithField("test", "TestDNSCheck_TXTRecord"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestDNSCheck_NSRecord tests NS record lookup.
func TestDNSCheck_NSRecord(t *testing.T) {
	check := DNSCheck{
		Domain:     "google.com",
		RecordType: "NS",
		Timeout:    "10s",
		Logger:     logrus.WithField("test", "TestDNSCheck_NSRecord"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}
