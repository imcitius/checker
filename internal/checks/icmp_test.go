package checks

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// skipIfNoICMPPrivilege skips the test if the error indicates missing raw socket permissions.
func skipIfNoICMPPrivilege(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), "permission denied") {
		t.Skip("Skipping: ICMP raw socket not permitted (requires root or CAP_NET_RAW)")
	}
}

// TestICMPCheck_EmptyHost tests handling of empty host.
func TestICMPCheck_EmptyHost(t *testing.T) {
	check := ICMPCheck{
		Host:    "",
		Count:   3,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "TestICMPCheck_EmptyHost"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for empty host but got success")
	}
	if err.Error() != ErrEmptyHost {
		t.Errorf("Expected error message '%s', got '%s'", ErrEmptyHost, err.Error())
	}
}

// TestICMPCheck_InvalidHost tests handling of invalid host names.
func TestICMPCheck_InvalidHost(t *testing.T) {
	check := ICMPCheck{
		Host:    "invalid.host.name.that.does.not.exist",
		Count:   3,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "TestICMPCheck_InvalidHost"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid hostname but got success")
	}
}

// TestICMPCheck_LocalhostSuccess tests a successful ping to localhost.
func TestICMPCheck_LocalhostSuccess(t *testing.T) {
	check := ICMPCheck{
		Host:    "localhost",
		Count:   1,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestICMPCheck_LocalhostSuccess"),
	}

	duration, err := check.Run()
	skipIfNoICMPPrivilege(t, err)
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestICMPCheck_PacketLoss tests handling of packet loss.
func TestICMPCheck_PacketLoss(t *testing.T) {
	// This test uses a non-existent host that should result in 100% packet loss
	check := ICMPCheck{
		Host:    "240.0.0.0", // Reserved for future use, guaranteed to be unreachable
		Count:   1,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestICMPCheck_PacketLoss"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error due to packet loss but got success")
	}
}

// TestICMPCheck_InvalidTimeout tests handling of invalid timeout values.
func TestICMPCheck_InvalidTimeout(t *testing.T) {
	check := ICMPCheck{
		Host:    "localhost",
		Count:   3,
		Timeout: "invalid",
		Logger:  logrus.WithField("test", "TestICMPCheck_InvalidTimeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid timeout but got success")
	}
}

// TestICMPCheck_ZeroCount tests handling of zero count.
func TestICMPCheck_ZeroCount(t *testing.T) {
	check := ICMPCheck{
		Host:    "localhost",
		Count:   0,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestICMPCheck_ZeroCount"),
	}

	_, err := check.Run()
	skipIfNoICMPPrivilege(t, err)
	if err != nil {
		t.Errorf("Expected success with default count but got error: %v", err)
	}
}

// TestICMPCheck_Timeout tests that pings timeout as expected.
func TestICMPCheck_Timeout(t *testing.T) {
	check := ICMPCheck{
		Host:    "8.8.8.8", // Using Google's DNS server
		Count:   10,        // High count
		Timeout: "1ms",     // Extremely short timeout
		Logger:  logrus.WithField("test", "TestICMPCheck_Timeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected timeout error but got success")
	}
}
