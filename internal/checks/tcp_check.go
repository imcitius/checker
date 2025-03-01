package checks

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

// TCPCheck represents a TCP health check.
type TCPCheck struct {
	Host    string
	Port    int
	Timeout string
	Logger  *logrus.Entry
}

// Run executes the TCP health check.
func (tc *TCPCheck) Run() (time.Duration, error) {
	var (
		errorHeader, errorMessage string
		start                     = time.Now()
	)

	timeOut, err := time.ParseDuration(tc.Timeout)
	if err != nil {
		return time.Now().Sub(start), fmt.Errorf(ErrCannotParseTimeout, tc.Timeout)
	}

	if tc.Host == "" {
		errorMessage = errorHeader + ErrEmptyHost
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	if tc.Port == 0 {
		errorMessage = errorHeader + ErrEmptyPort
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	conn, err := net.DialTimeout("tcp", tc.Host, timeOut)
	if err != nil {
		return time.Now().Sub(start), err
	}
	conn.Close()
	return time.Now().Sub(start), nil
}
