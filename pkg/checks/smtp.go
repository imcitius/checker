// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ErrSMTPConnection = "failed to connect to SMTP server: %v"
	ErrSMTPHello      = "SMTP EHLO failed: %v"
	ErrSMTPStartTLS   = "SMTP STARTTLS failed: %v"
	ErrSMTPAuth       = "SMTP authentication failed: %v"
)

// SMTPCheck represents an SMTP health check.
// It connects to the SMTP server, optionally upgrades to TLS via STARTTLS,
// and optionally authenticates with provided credentials.
type SMTPCheck struct {
	Host     string
	Port     int
	Timeout  string
	StartTLS bool
	Username string
	Password string
	Logger   *logrus.Entry
}

// Run executes the SMTP health check.
func (check *SMTPCheck) Run() (time.Duration, error) {
	start := time.Now()

	// Parse timeout
	timeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), err
	}

	if check.Host == "" {
		return time.Since(start), errors.New(ErrEmptyHost)
	}

	if check.Port == 0 {
		return time.Since(start), errors.New(ErrEmptyPort)
	}

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "smtp")
	}

	hostPort := net.JoinHostPort(check.Host, fmt.Sprintf("%d", check.Port))

	// Dial with timeout
	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		check.Logger.WithError(err).Debugf("SMTP check %s dial error: %+v", hostPort, err)
		return time.Since(start), fmt.Errorf(ErrSMTPConnection, err)
	}

	// Set deadline for all subsequent operations
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		conn.Close()
		return time.Since(start), fmt.Errorf(ErrSMTPConnection, err)
	}

	// Create SMTP client from existing connection
	client, err := smtp.NewClient(conn, check.Host)
	if err != nil {
		conn.Close()
		check.Logger.WithError(err).Debugf("SMTP check %s client error: %+v", hostPort, err)
		return time.Since(start), fmt.Errorf(ErrSMTPConnection, err)
	}
	defer client.Close()

	// Send EHLO
	if err := client.Hello("checker"); err != nil {
		check.Logger.WithError(err).Debugf("SMTP check %s EHLO error: %+v", hostPort, err)
		return time.Since(start), fmt.Errorf(ErrSMTPHello, err)
	}

	// Upgrade to TLS if requested
	if check.StartTLS {
		tlsConfig := &tls.Config{
			ServerName: check.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			check.Logger.WithError(err).Debugf("SMTP check %s STARTTLS error: %+v", hostPort, err)
			return time.Since(start), fmt.Errorf(ErrSMTPStartTLS, err)
		}
	}

	// Authenticate if credentials are provided
	if check.Username != "" && check.Password != "" {
		auth := smtp.PlainAuth("", check.Username, check.Password, check.Host)
		if err := client.Auth(auth); err != nil {
			check.Logger.WithError(err).Debugf("SMTP check %s auth error: %+v", hostPort, err)
			return time.Since(start), fmt.Errorf(ErrSMTPAuth, err)
		}
	}

	// Send QUIT
	if err := client.Quit(); err != nil {
		// Log but don't fail — the server responded to EHLO, so it's alive
		check.Logger.WithError(err).Debugf("SMTP check %s QUIT error (non-fatal): %+v", hostPort, err)
	}

	return time.Since(start), nil
}
