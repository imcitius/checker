package checks

import (
	"net"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestTCPCheck_Success tests a successful TCP connection.
func TestTCPCheck_Success(t *testing.T) {
	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer listener.Close()

	// Accept connections in a goroutine
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
	}()

	check := TCPCheck{
		Host:    "127.0.0.1",
		Port:    listener.Addr().(*net.TCPAddr).Port,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "TestTCPCheck_Success"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestTCPCheck_ConnectionRefused tests handling of refused connections.
func TestTCPCheck_ConnectionRefused(t *testing.T) {
	check := TCPCheck{
		Host:    "127.0.0.1",
		Port:    54321, // Using a port that's likely not in use
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestTCPCheck_ConnectionRefused"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for refused connection but got success")
	}
}

// TestTCPCheck_InvalidHost tests handling of invalid host names.
func TestTCPCheck_InvalidHost(t *testing.T) {
	check := TCPCheck{
		Host:    "invalid.host.name.that.does.not.exist",
		Port:    80,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestTCPCheck_InvalidHost"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid hostname but got success")
	}
}

// TestTCPCheck_EmptyHost tests handling of empty host.
func TestTCPCheck_EmptyHost(t *testing.T) {
	check := TCPCheck{
		Host:    "",
		Port:    80,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestTCPCheck_EmptyHost"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for empty host but got success")
	}
	if err.Error() != ErrEmptyHost {
		t.Errorf("Expected error message '%s', got '%s'", ErrEmptyHost, err.Error())
	}
}

// TestTCPCheck_EmptyPort tests handling of empty port.
func TestTCPCheck_EmptyPort(t *testing.T) {
	check := TCPCheck{
		Host:    "localhost",
		Port:    0,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestTCPCheck_EmptyPort"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for empty port but got success")
	}
	if err.Error() != ErrEmptyPort {
		t.Errorf("Expected error message '%s', got '%s'", ErrEmptyPort, err.Error())
	}
}

// TestTCPCheck_InvalidTimeout tests handling of invalid timeout values.
func TestTCPCheck_InvalidTimeout(t *testing.T) {
	check := TCPCheck{
		Host:    "localhost",
		Port:    80,
		Timeout: "invalid",
		Logger:  logrus.WithField("test", "TestTCPCheck_InvalidTimeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid timeout but got success")
	}
}

// TestTCPCheck_Timeout tests that connections timeout as expected.
func TestTCPCheck_Timeout(t *testing.T) {
	// Use a non-routable IP address to force timeout
	// 192.0.2.0/24 is reserved for documentation (TEST-NET-1)
	check := TCPCheck{
		Host:    "192.0.2.1",
		Port:    12345,
		Timeout: "100ms", // Short timeout
		Logger:  logrus.WithField("test", "TestTCPCheck_Timeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected timeout error but got success")
	}
	if err != nil && !isTimeoutError(err) {
		t.Errorf("Expected timeout error but got: %v", err)
	}
}

// isTimeoutError checks if the error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	if !ok {
		return false
	}
	return netErr.Timeout() || netErr.Temporary()
}
