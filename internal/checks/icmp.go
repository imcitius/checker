package checks

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-ping/ping"
	"github.com/sirupsen/logrus"
)

// ICMPCheck represents an ICMP health check.
type ICMPCheck struct {
	Host    string
	Count   int
	Timeout string
	Logger  *logrus.Entry
}

// Run executes the ICMP health check.
func (check *ICMPCheck) Run() (time.Duration, error) {
	var (
		errorHeader, errorMessage string
		start                     = time.Now()
	)

	if check.Host == "" {
		errorMessage = errorHeader + ErrEmptyHost
		return time.Since(start), errors.New(errorMessage)
	}

	// Parse timeout duration first to validate it
	timeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		return time.Since(start), fmt.Errorf("invalid timeout value: %v", err)
	}
	if timeout <= 0 {
		return time.Since(start), fmt.Errorf("timeout must be positive")
	}

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "icmp")
	}

	check.Logger.Debugf("icmp test: %s", check.Host)
	pinger, err := ping.NewPinger(check.Host)
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrICMPError, err)
		return time.Since(start), errors.New(errorMessage)
	}

	// Set default count if not specified
	if check.Count <= 0 {
		check.Count = 3 // Default to 3 pings
	}

	pinger.Count = check.Count
	pinger.Timeout = timeout
	err = pinger.Run()

	// Only get statistics if Run() was successful
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrICMPError, err)
		return time.Since(start), errors.New(errorMessage)
	}

	stats := pinger.Statistics()
	check.Logger.WithError(err).Debugf("ICMP host %s, res: %+v (err: %+v, stats: %+v)", check.Host, pinger, err, stats)

	if stats.PacketLoss > 0 {
		errorMessage = errorHeader + fmt.Sprintf(ErrPacketLoss, stats.PacketLoss)
		return time.Since(start), errors.New(errorMessage)
	}

	return time.Since(start), nil
}
