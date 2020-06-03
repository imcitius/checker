package status

import "my/checker/config"

func initCheckStatus(c *config.Check) {
	if _, ok := Statuses.Checks[c.UUid]; !ok {
		Statuses.Checks[c.UUid] = new(CheckStatuses)
		Statuses.Checks[c.UUid].UUID = c.UUid
	}
}
