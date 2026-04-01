package checks

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DefaultSSHPort = 22

	ErrSSHBannerMismatch = "SSH banner mismatch: expected substring %q not found in %q"
	ErrSSHNoBanner       = "no SSH banner received"
)

// SSHCheck performs an SSH banner grab by dialing a TCP connection,
// reading the first line (the SSH banner), and optionally verifying
// that it contains an expected substring.
// It does NOT attempt authentication — only a raw TCP read.
type SSHCheck struct {
	Host         string
	Port         int
	Timeout      string
	ExpectBanner string
	Logger       *logrus.Entry
}

// Run executes the SSH banner grab check.
func (check *SSHCheck) Run() (time.Duration, error) {
	start := time.Now()

	// Parse timeout
	timeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), err
	}

	if check.Host == "" {
		return time.Since(start), errors.New(ErrEmptyHost)
	}

	port := check.Port
	if port == 0 {
		port = DefaultSSHPort
	}

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "ssh")
	}

	hostPort := net.JoinHostPort(check.Host, fmt.Sprintf("%d", port))

	// Dial TCP connection
	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		check.Logger.WithError(err).Debugf("SSH check %s, dial err: %+v", hostPort, err)
		return 0, fmt.Errorf("SSH dial %s: %w", hostPort, err)
	}
	defer conn.Close()

	// Set a read deadline so we don't hang waiting for a banner
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return time.Since(start), fmt.Errorf("failed to set read deadline: %v", err)
	}

	// Read the first line — the SSH banner (e.g. "SSH-2.0-OpenSSH_8.9\r\n")
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return time.Since(start), fmt.Errorf("failed to read SSH banner: %v", scanErr)
		}
		return time.Since(start), errors.New(ErrSSHNoBanner)
	}

	banner := strings.TrimSpace(scanner.Text())
	check.Logger.Debugf("SSH banner from %s: %s", hostPort, banner)

	if banner == "" {
		return time.Since(start), errors.New(ErrSSHNoBanner)
	}

	// If ExpectBanner is set, verify the banner contains the expected substring
	if check.ExpectBanner != "" {
		if !strings.Contains(banner, check.ExpectBanner) {
			return time.Since(start), fmt.Errorf(ErrSSHBannerMismatch, check.ExpectBanner, banner)
		}
	}

	return time.Since(start), nil
}
