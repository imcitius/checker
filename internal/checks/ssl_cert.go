package checks

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ErrSSLEmptyHost       = "empty host"
	ErrSSLConnection      = "TLS connection error: %s"
	ErrSSLNoCertificates  = "no peer certificates returned"
	ErrSSLCertExpiringSoon = "certificate expires in %d days (warning threshold: %d days)"
	ErrSSLCertExpired     = "certificate already expired on %s"
	ErrSSLChainInvalid    = "certificate chain validation failed: %s"

	SSLDefaultPort    = 443
	SSLDefaultTimeout = "10s"
)

// SSLCertCheck represents an SSL certificate health check.
type SSLCertCheck struct {
	Host              string
	Port              int
	Timeout           string
	ExpiryWarningDays int
	ValidateChain     bool
	Logger            *logrus.Entry
}

// Run executes the SSL certificate health check.
func (check *SSLCertCheck) Run() (time.Duration, error) {
	start := time.Now()

	if check.Host == "" {
		return time.Since(start), errors.New(ErrSSLEmptyHost)
	}

	// Apply defaults
	port := check.Port
	if port == 0 {
		port = SSLDefaultPort
	}

	timeoutStr := check.Timeout
	if timeoutStr == "" {
		timeoutStr = SSLDefaultTimeout
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return time.Since(start), fmt.Errorf("invalid timeout value: %v", err)
	}

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "ssl_cert")
	}

	hostPort := net.JoinHostPort(check.Host, fmt.Sprintf("%d", port))

	// Open TLS connection
	dialer := &net.Dialer{Timeout: timeout}
	tlsConfig := &tls.Config{
		// When ValidateChain is false, we still connect but skip chain verification
		// to just inspect the certificate. When true, Go's default verification applies.
		InsecureSkipVerify: !check.ValidateChain,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", hostPort, tlsConfig)
	if err != nil {
		return time.Since(start), fmt.Errorf(ErrSSLConnection, err)
	}
	defer conn.Close()

	// Get peer certificates
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return time.Since(start), errors.New(ErrSSLNoCertificates)
	}

	leaf := state.PeerCertificates[0]
	now := time.Now()

	// Check if certificate has already expired
	if now.After(leaf.NotAfter) {
		return time.Since(start), fmt.Errorf(ErrSSLCertExpired, leaf.NotAfter.Format(time.RFC3339))
	}

	// Calculate days remaining
	daysRemaining := int(time.Until(leaf.NotAfter).Hours() / 24)

	// Check if certificate is within the expiry warning window
	if check.ExpiryWarningDays > 0 && daysRemaining <= check.ExpiryWarningDays {
		return time.Since(start), fmt.Errorf(ErrSSLCertExpiringSoon, daysRemaining, check.ExpiryWarningDays)
	}

	// If ValidateChain is true, verify the full chain against system roots
	if check.ValidateChain {
		opts := x509.VerifyOptions{
			DNSName:       check.Host,
			Intermediates: x509.NewCertPool(),
		}
		// Add intermediate certificates
		for _, cert := range state.PeerCertificates[1:] {
			opts.Intermediates.AddCert(cert)
		}
		if _, err := leaf.Verify(opts); err != nil {
			return time.Since(start), fmt.Errorf(ErrSSLChainInvalid, err)
		}
	}

	check.Logger.Infof("SSL cert for %s valid, %d days remaining (expires %s)",
		hostPort, daysRemaining, leaf.NotAfter.Format(time.RFC3339))

	return time.Since(start), nil
}
