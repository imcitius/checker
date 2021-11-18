package status

import (
	"my/checker/config"
	"time"
)

func InitCheckStatus(c *config.Check) {
	if _, ok := Statuses.Checks[c.UUid]; !ok {
		Statuses.Checks[c.UUid] = new(CheckStatuses)
		Statuses.Checks[c.UUid].UUID = c.UUid
		Statuses.Checks[c.UUid].Mode = config.Config.Defaults.Parameters.Mode
		Statuses.Checks[c.UUid].Name = c.Name
	}
}

func GetCheckMode(c *config.Check) (string, error) {
	InitCheckStatus(c)
	return Statuses.Checks[c.UUid].Mode, nil
}

func SetCheckMode(c *config.Check, status string) error {
	InitCheckStatus(c)
	Statuses.Checks[c.UUid].Mode = status
	return nil
}

func PingCheck(c *config.Check) error {
	InitCheckStatus(c)
	Statuses.Checks[c.UUid].LastResult = true
	Statuses.Checks[c.UUid].When = time.Now()
	return nil
}
