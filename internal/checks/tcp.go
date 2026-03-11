package checks

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// TCPCheck represents a TCP health check.
type TCPCheck struct {
	Host    string
	Port    int
	Timeout string
	Logger  *logrus.Entry
}

// Run executes the TCP health check.
func (check *TCPCheck) Run() (time.Duration, error) {
	var (
		errorHeader, errorMessage string
		start                     = time.Now()
	)

	// Parse timeout duration first to validate it
	timeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		return time.Since(start), fmt.Errorf("invalid timeout value: %v", err)
	}
	if timeout <= 0 {
		return time.Since(start), fmt.Errorf("timeout must be positive")
	}

	if check.Host == "" {
		errorMessage = errorHeader + ErrEmptyHost
		return time.Since(start), errors.New(errorMessage)
	}

	if check.Port == 0 {
		errorMessage = errorHeader + ErrEmptyPort
		return time.Since(start), errors.New(errorMessage)
	}

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "tcp")
	}

	hostPort := net.JoinHostPort(check.Host, fmt.Sprintf("%d", check.Port))
	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		check.Logger.WithError(err).Debugf("TCP check %s, err: %+v", hostPort, err)
		// Return the original error to preserve timeout information
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}
