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
	timeout, err := time.ParseDuration(check.Timeout)
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

// func init() {
// 	Checks["passive"] = func(c *config.Check, p *projects.Project) error {

// 		errorHeader := fmt.Sprintf("Passive check '%s'\nerror at project: %s\nCheck UUID: %s\n", c.Name, p.Name, c.UUid)

// 		timeout, err := time.ParseDuration(c.Timeout)
// 		if err != nil {
// 			config.Log.Fatalf("Cannot parse timeout for check %s: %s", c.UUid, c.Timeout)
// 			return fmt.Errorf(errorHeader + "Cannot parse timeout")
// 		}

// 		// do not check too early
// 		if time.Since(config.StartTime) < timeout {
// 			return nil
// 		}

// 		if status.Statuses.Checks[c.UUid].LastResult {
// 			if time.Since(status.Statuses.Checks[c.UUid].When) < timeout {
// 				return nil
// 			} else {
// 				return fmt.Errorf(errorHeader+"Ping timeout, last ping at %s", status.Statuses.Checks[c.UUid].When)
// 			}
// 		} else {
// 			return fmt.Errorf(errorHeader + "Bad status, no pings since start")
// 		}
// 	}
// }
