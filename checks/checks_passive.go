package check

import (
	"fmt"
	"my/checker/config"
	"my/checker/status"
	"time"
)

func init() {
	config.Checks["passive"] = func(c *config.Check, p *config.Project) error {

		errorHeader := fmt.Sprintf("Passive error at project: %s\nCheck UUID: %s\n", p.Name, c.UUid)

		timeout, err := time.ParseDuration(c.Timeout)
		if err != nil {
			config.Log.Fatalf("Cannot parse timeout for check %s: %s", c.UUid, c.Timeout)
			return fmt.Errorf(errorHeader + "Cannot parse timeout")
		}

		// do not check too early
		if time.Now().Sub(config.StartTime) < timeout {
			return nil
		}

		if status.Statuses.Checks[c.UUid].LastResult == true {
			if time.Now().Sub(status.Statuses.Checks[c.UUid].When) < timeout {
				return nil
			} else {
				return fmt.Errorf(errorHeader + "Ping timeout")
			}
		} else {
			return fmt.Errorf(errorHeader+"Bad status, last ping at: %s", status.Statuses.Checks[c.UUid].When)
		}
	}
}
