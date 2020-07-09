package status

import "my/checker/config"

func InitCheckStatus(c *config.Check) {
	if _, ok := Statuses.Checks[c.UUid]; !ok {
		Statuses.Checks[c.UUid] = new(CheckStatuses)
		Statuses.Checks[c.UUid].UUID = c.UUid
		Statuses.Checks[c.UUid].Mode = config.Config.Defaults.Parameters.Mode
		Statuses.Checks[c.UUid].Name = c.Name
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
	InitCheckStatus(c)
	Statuses.Checks[c.UUid].Mode = status
}
