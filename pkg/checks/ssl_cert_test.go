// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestSSLCertCheck_RealHost tests against a real host (google.com:443).
func TestSSLCertCheck_RealHost(t *testing.T) {
	check := SSLCertCheck{
		Host:              "google.com",
		Port:              443,
		Timeout:           "10s",
		ExpiryWarningDays: 5, // Google's cert should have more than 5 days
		ValidateChain:     true,
		Logger:            logrus.WithField("test", "TestSSLCertCheck_RealHost"),
	}

	duration, err := check.Run()
	if err != nil {
		t.Errorf("Expected success for google.com but got error: %v", err)
	}
	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestSSLCertCheck_ExpiryWarning tests that a cert expiring within the warning window triggers failure.
func TestSSLCertCheck_ExpiryWarning(t *testing.T) {
	// Start a TLS server with a certificate expiring in 30 days
	server, port := startTestTLSServer(t, 30*24*time.Hour)
	defer server.Close()

	check := SSLCertCheck{
		Host:              "127.0.0.1",
		Port:              port,
		Timeout:           "5s",
		ExpiryWarningDays: 60, // Warning at 60 days, cert expires in 30
		ValidateChain:     false,
		Logger:            logrus.WithField("test", "TestSSLCertCheck_ExpiryWarning"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for cert expiring within warning window but got success")
	}
}

// TestSSLCertCheck_ExpiryOK tests that a cert not within the warning window passes.
func TestSSLCertCheck_ExpiryOK(t *testing.T) {
	// Start a TLS server with a certificate expiring in 90 days
	server, port := startTestTLSServer(t, 90*24*time.Hour)
	defer server.Close()

	check := SSLCertCheck{
		Host:              "127.0.0.1",
		Port:              port,
		Timeout:           "5s",
		ExpiryWarningDays: 30, // Warning at 30 days, cert expires in 90
		ValidateChain:     false,
		Logger:            logrus.WithField("test", "TestSSLCertCheck_ExpiryOK"),
	}

	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected success but got error: %v", err)
	}
}

// TestSSLCertCheck_InvalidHost tests that an invalid host returns an error.
func TestSSLCertCheck_InvalidHost(t *testing.T) {
	check := SSLCertCheck{
		Host:              "invalid.host.that.does.not.exist.example",
		Port:              443,
		Timeout:           "2s",
		ExpiryWarningDays: 30,
		Logger:            logrus.WithField("test", "TestSSLCertCheck_InvalidHost"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for invalid host but got success")
	}
}

// TestSSLCertCheck_EmptyHost tests that an empty host returns an error.
func TestSSLCertCheck_EmptyHost(t *testing.T) {
	check := SSLCertCheck{
		Host:    "",
		Port:    443,
		Timeout: "5s",
		Logger:  logrus.WithField("test", "TestSSLCertCheck_EmptyHost"),
	}

	_, err := check.Run()
	if err == nil {
		t.Error("Expected error for empty host but got success")
	}
	if err.Error() != ErrSSLEmptyHost {
		t.Errorf("Expected error message '%s', got '%s'", ErrSSLEmptyHost, err.Error())
	}
}

// TestSSLCertCheck_DefaultPort tests that the default port (443) is used when port is 0.
func TestSSLCertCheck_DefaultPort(t *testing.T) {
	check := SSLCertCheck{
		Host:              "google.com",
		Port:              0, // should default to 443
		Timeout:           "10s",
		ExpiryWarningDays: 5,
		ValidateChain:     true,
		Logger:            logrus.WithField("test", "TestSSLCertCheck_DefaultPort"),
	}

	_, err := check.Run()
	if err != nil {
		t.Errorf("Expected success with default port but got error: %v", err)
	}
}

// startTestTLSServer creates a TLS server with a self-signed certificate
// that expires after the given duration.
func startTestTLSServer(t *testing.T, validFor time.Duration) (net.Listener, int) {
	t.Helper()

	// Generate a self-signed certificate
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(validFor),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		t.Fatalf("Failed to start TLS server: %v", err)
	}

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				// Read until client disconnects to keep TLS handshake alive
				buf := make([]byte, 1024)
				for {
					if _, err := c.Read(buf); err != nil {
						break
					}
				}
				c.Close()
			}(conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	return listener, port
}
