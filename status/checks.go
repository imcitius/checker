package status

import "my/checker/config"

func initCheckStatus(c *config.Check) {
	if _, ok := Statuses.Checks[c.UUid]; !ok {
		Statuses.Checks[c.UUid] = new(CheckStatuses)
	}
}

func GetCheckStatus(c *config.Check) string {
	return Statuses.Checks[c.UUid].Status
}

func SetCheckStatus(c *config.Check, status string) {
	Statuses.Checks[c.UUid].Status = status
}

func GetCheckMode(c *config.Check) string {
	return Statuses.Checks[c.UUid].Mode
}

func SetCheckMode(c *config.Check, status string) {
	initCheckStatus(c)
	Statuses.Checks[c.UUid].Mode = status
}
