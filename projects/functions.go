package project

import "my/checker/config"

func GetProjectByCheckUUID(uuID string) *Project {
	for _, project := range config.Config.Projects {
		for _, healthcheck := range project.Healthchecks {
			for _, check := range healthcheck.Checks {
				if uuID == check.UUid {
					return &Project{Project: project}
				}
			}
		}
	}
	return nil
}
