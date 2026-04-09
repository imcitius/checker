// SPDX-License-Identifier: BUSL-1.1

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

	// Parse timeout
	timeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), err
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
		return 0, fmt.Errorf("TCP dial %s: %w", hostPort, err)
	}
	defer conn.Close()
	return time.Since(start), nil
}
