package misc

import "my/checker/config"

func GetCheckByUUID(uuID string) *config.Check {
	for _, project := range config.Config.Projects {
		for _, healthcheck := range project.Healthchecks {
			for _, check := range healthcheck.Checks {
				if uuID == check.UUid {
					return &check
				}
			}
		}
	}
	return nil
}
