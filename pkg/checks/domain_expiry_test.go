// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"testing"
	"time"
)

// TestParseExpiryDate_RFC3339 tests parsing of RFC3339 format dates.
func TestParseExpiryDate_RFC3339(t *testing.T) {
	input := `Domain Name: EXAMPLE.COM
Registry Domain ID: 123456
Registry Expiry Date: 2025-03-15T00:00:00Z
Updated Date: 2024-01-01T00:00:00Z
`
	expected := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	got, err := ParseExpiryDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestParseExpiryDate_ISODate tests parsing of plain ISO date format.
func TestParseExpiryDate_ISODate(t *testing.T) {
	input := `Domain Name: EXAMPLE.COM
Registry Expiry Date: 2025-03-15
`
	expected := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	got, err := ParseExpiryDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestParseExpiryDate_DayMonthYear tests parsing "15-Mar-2025" format.
func TestParseExpiryDate_DayMonthYear(t *testing.T) {
	input := `Domain Name: EXAMPLE.COM
Expiration Date: 15-Mar-2025
`
	expected := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	got, err := ParseExpiryDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestParseExpiryDate_ExpiryDateVariant tests "Expiry Date:" (without "Registry").
func TestParseExpiryDate_ExpiryDateVariant(t *testing.T) {
	input := `Domain Name: EXAMPLE.COM
Expiry Date: 2026-12-01T12:00:00Z
`
	expected := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	got, err := ParseExpiryDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestParseExpiryDate_PaidTill tests parsing "paid-till:" format (Russian registrars).
func TestParseExpiryDate_PaidTill(t *testing.T) {
	input := `domain:        EXAMPLE.RU
nserver:       ns1.example.ru.
paid-till:     2025-06-20
`
	expected := time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC)
	got, err := ParseExpiryDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestParseExpiryDate_SlashSeparated tests parsing "2025/03/15" format.
func TestParseExpiryDate_SlashSeparated(t *testing.T) {
	input := `Domain Name: EXAMPLE.JP
[Expires on] 2025/03/15
`
	expected := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	got, err := ParseExpiryDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestParseExpiryDate_NotFound tests that an error is returned when no date is found.
func TestParseExpiryDate_NotFound(t *testing.T) {
	input := `Domain Name: EXAMPLE.COM
No matching date here.
`
	_, err := ParseExpiryDate(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestParseExpiryDate_MultipleFormats tests that the first matching pattern wins.
func TestParseExpiryDate_MultipleFormats(t *testing.T) {
	input := `Domain Name: EXAMPLE.COM
Registry Expiry Date: 2025-03-15T00:00:00Z
Expiration Date: 16-Mar-2025
`
	// Should match the first pattern (RFC3339)
	expected := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	got, err := ParseExpiryDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestExtractTLD tests TLD extraction from domain names.
func TestExtractTLD(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"example.com", "com"},
		{"example.co.uk", "uk"},
		{"sub.example.io", "io"},
		{"example.com.", "com"},
		{"EXAMPLE.COM", "com"},
		{"test", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := extractTLD(tt.domain)
			if got != tt.expected {
				t.Errorf("extractTLD(%q) = %q, want %q", tt.domain, got, tt.expected)
			}
		})
	}
}

// TestDomainExpiryCheck_EmptyDomain tests that an empty domain returns an error.
func TestDomainExpiryCheck_EmptyDomain(t *testing.T) {
	check := &DomainExpiryCheck{
		Domain:  "",
		Timeout: "5s",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
	if err.Error() != ErrEmptyDomain {
		t.Errorf("expected error %q, got %q", ErrEmptyDomain, err.Error())
	}
}

// TestDomainExpiryCheck_InvalidTimeout tests that an invalid timeout returns an error.
func TestDomainExpiryCheck_InvalidTimeout(t *testing.T) {
	check := &DomainExpiryCheck{
		Domain:  "example.com",
		Timeout: "invalid",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

// TestDomainExpiryCheck_UnknownTLD tests that an unknown TLD returns an error.
func TestDomainExpiryCheck_UnknownTLD(t *testing.T) {
	check := &DomainExpiryCheck{
		Domain:  "example.invalidtld",
		Timeout: "5s",
	}
	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for unknown TLD")
	}
}

// TestDomainExpiryCheck_Integration performs a real WHOIS lookup.
// This test is skipped in short mode as it requires network access.
func TestDomainExpiryCheck_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	check := &DomainExpiryCheck{
		Domain:            "google.com",
		ExpiryWarningDays: 7,
		Timeout:           "10s",
	}

	duration, err := check.Run()
	if err != nil {
		t.Fatalf("unexpected error checking google.com: %v", err)
	}

	if duration <= 0 {
		t.Error("expected positive duration")
	}
}
