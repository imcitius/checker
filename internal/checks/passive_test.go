package checks

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestPassiveCheck_Success tests a successful passive check within timeout.
func TestPassiveCheck_Success(t *testing.T) {
	check := PassiveCheck{
		LastPing:    time.Now(),
		Timeout:     "5m",
		ErrorHeader: "TestPassiveCheck",
		Logger:      logrus.WithField("test", "TestPassiveCheck_Success"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration < 0 {
		t.Error("Expected non-negative duration")
	}
}

// TestPassiveCheck_Timeout tests handling of timeout condition.
func TestPassiveCheck_Timeout(t *testing.T) {
	check := PassiveCheck{
		LastPing:    time.Now().Add(-10 * time.Minute), // Last ping was 10 minutes ago
		Timeout:     "5m",                              // 5 minute timeout
		ErrorHeader: "TestPassiveCheck",
		Logger:      logrus.WithField("test", "TestPassiveCheck_Timeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected timeout error but got success")
	}
}

// TestPassiveCheck_InvalidTimeout tests handling of invalid timeout values.
func TestPassiveCheck_InvalidTimeout(t *testing.T) {
	check := PassiveCheck{
		LastPing:    time.Now(),
		Timeout:     "invalid",
		ErrorHeader: "TestPassiveCheck",
		Logger:      logrus.WithField("test", "TestPassiveCheck_InvalidTimeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid timeout but got success")
	}
}

// TestPassiveCheck_EmptyTimeout tests that empty timeout uses the default (15m) and succeeds
// when the last ping is recent.
func TestPassiveCheck_EmptyTimeout(t *testing.T) {
	check := PassiveCheck{
		LastPing:    time.Now(),
		Timeout:     "",
		ErrorHeader: "TestPassiveCheck",
		Logger:      logrus.WithField("test", "TestPassiveCheck_EmptyTimeout"),
	}

	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected success with default timeout, got error: %v", err)
	}
}

// TestPassiveCheck_ZeroTimeout tests handling of zero timeout.
func TestPassiveCheck_ZeroTimeout(t *testing.T) {
	check := PassiveCheck{
		LastPing:    time.Now(),
		Timeout:     "0s",
		ErrorHeader: "TestPassiveCheck",
		Logger:      logrus.WithField("test", "TestPassiveCheck_ZeroTimeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for zero timeout but got success")
	}
}

// TestPassiveCheck_FutureLastPing tests handling of future last ping time.
func TestPassiveCheck_FutureLastPing(t *testing.T) {
	check := PassiveCheck{
		LastPing:    time.Now().Add(1 * time.Hour), // Last ping is in the future
		Timeout:     "5m",
		ErrorHeader: "TestPassiveCheck",
		Logger:      logrus.WithField("test", "TestPassiveCheck_FutureLastPing"),
	}

	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected success for future last ping but got error: %v", err)
	}
}

// TestPassiveCheck_EdgeCaseTimeout tests timeout at exact boundary.
func TestPassiveCheck_EdgeCaseTimeout(t *testing.T) {
	timeout := "5m"
	timeoutDuration, _ := time.ParseDuration(timeout)

	check := PassiveCheck{
		LastPing:    time.Now().Add(-timeoutDuration), // Last ping exactly at timeout boundary
		Timeout:     timeout,
		ErrorHeader: "TestPassiveCheck",
		Logger:      logrus.WithField("test", "TestPassiveCheck_EdgeCaseTimeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected timeout error at boundary but got success")
	}
}
