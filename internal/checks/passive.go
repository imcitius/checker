package checks

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type PassiveCheck struct {
	LastPing    time.Time
	Timeout     string // Duration string for timeout check (e.g. "15m")
	ErrorHeader string // Prefix for error messages
	Logger      *logrus.Entry
}

func (check *PassiveCheck) Run() (time.Duration, error) {
	start := time.Now()

	// get check status from database using store
	// Parse timeout duration
	timeout, err := parseCheckTimeout(check.Timeout, 15*time.Minute)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid timeout duration: %v", check.ErrorHeader, err)
	}

	// Check if last ping is too old
	if time.Since(check.LastPing) > timeout {
		check.Logger.Errorf("%s: cron ping timeout - last ping was %v ago", check.ErrorHeader, time.Since(check.LastPing))
		return 0, fmt.Errorf("%s: cron ping timeout - last ping was %v ago", check.ErrorHeader, time.Since(check.LastPing))
	}


	return time.Since(start), nil
}
