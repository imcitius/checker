// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"net"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// startMockSSHServer starts a mock TCP server that sends an SSH banner
// and returns the listener and the port.
func startMockSSHServer(t *testing.T, banner string) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock SSH server: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			if banner != "" {
				conn.Write([]byte(banner + "\r\n"))
			}
			conn.Close()
		}
	}()

	return listener
}

// TestSSHCheck_Success tests a successful SSH banner grab.
func TestSSHCheck_Success(t *testing.T) {
	listener := startMockSSHServer(t, "SSH-2.0-OpenSSH_8.9")
	defer listener.Close()

	check := SSHCheck{
		Host:    "127.0.0.1",
		Port:    listener.Addr().(*net.TCPAddr).Port,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "TestSSHCheck_Success"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestSSHCheck_BannerMatch tests that ExpectBanner substring matching works.
func TestSSHCheck_BannerMatch(t *testing.T) {
	listener := startMockSSHServer(t, "SSH-2.0-OpenSSH_8.9")
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	tests := []struct {
		name         string
		expectBanner string
		wantErr      bool
	}{
		{
			name:         "exact protocol match",
			expectBanner: "SSH-2.0",
			wantErr:      false,
		},
		{
			name:         "software match",
			expectBanner: "OpenSSH",
			wantErr:      false,
		},
		{
			name:         "version match",
			expectBanner: "OpenSSH_8.9",
			wantErr:      false,
		},
		{
			name:         "mismatch",
			expectBanner: "Dropbear",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := SSHCheck{
				Host:         "127.0.0.1",
				Port:         port,
				Timeout:      "5s",
				ExpectBanner: tt.expectBanner,
				Logger:       logrus.WithField("test", tt.name),
			}

			_, err := check.Run()
			if tt.wantErr && err == nil {
				t.Error("Expected error for banner mismatch but got success")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "SSH banner mismatch") {
					t.Errorf("Expected banner mismatch error, got: %v", err)
				}
			}
		})
	}
}

// TestSSHCheck_ConnectionRefused tests handling of connection failure.
func TestSSHCheck_ConnectionRefused(t *testing.T) {
	check := SSHCheck{
		Host:    "127.0.0.1",
		Port:    54322, // likely unused port
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestSSHCheck_ConnectionRefused"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for refused connection but got success")
	}
}

// TestSSHCheck_EmptyHost tests handling of empty host.
func TestSSHCheck_EmptyHost(t *testing.T) {
	check := SSHCheck{
		Host:    "",
		Port:    22,
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestSSHCheck_EmptyHost"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for empty host but got success")
	}
	if err.Error() != ErrEmptyHost {
		t.Errorf("Expected error message '%s', got '%s'", ErrEmptyHost, err.Error())
	}
}

// TestSSHCheck_DefaultPort tests that port defaults to 22 when set to 0.
func TestSSHCheck_DefaultPort(t *testing.T) {
	listener := startMockSSHServer(t, "SSH-2.0-TestServer")
	defer listener.Close()

	// The mock server runs on a random port, not 22.
	// We just verify the check doesn't error on port=0 by testing
	// that it defaults and attempts connection (which will fail on port 22
	// unless an actual SSH server is running — so we test the logic path).
	check := SSHCheck{
		Host:    "127.0.0.1",
		Port:    0, // should default to 22
		Timeout: "1s",
		Logger:  logrus.WithField("test", "TestSSHCheck_DefaultPort"),
	}

	_, err := check.Run()
	// We expect either success (if something is on port 22) or connection error
	// The important thing is we don't get "port is empty" error
	if err != nil && err.Error() == ErrEmptyPort {
		t.Error("Port 0 should default to 22, not produce empty port error")
	}
}

// TestSSHCheck_InvalidTimeout tests handling of invalid timeout values.
func TestSSHCheck_InvalidTimeout(t *testing.T) {
	check := SSHCheck{
		Host:    "127.0.0.1",
		Port:    22,
		Timeout: "invalid",
		Logger:  logrus.WithField("test", "TestSSHCheck_InvalidTimeout"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid timeout but got success")
	}
}

// TestSSHCheck_NoBanner tests handling of a server that sends no banner.
func TestSSHCheck_NoBanner(t *testing.T) {
	listener := startMockSSHServer(t, "")
	defer listener.Close()

	check := SSHCheck{
		Host:    "127.0.0.1",
		Port:    listener.Addr().(*net.TCPAddr).Port,
		Timeout: "2s",
		Logger:  logrus.WithField("test", "TestSSHCheck_NoBanner"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for no banner but got success")
	}
}
