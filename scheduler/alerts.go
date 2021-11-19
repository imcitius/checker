package scheduler

import (
	"errors"
	"fmt"
	"my/checker/config"
	projects "my/checker/projects"
	"my/checker/status"
	"time"
)

func sendCritAlerts(period string) time.Duration {
	startTime := time.Now()
	config.Log.Debug("sendCritAlerts")

	for _, prj := range Config.Projects {
		if prj.Parameters.Period == period {
			//if status.Statuses.Projects[prj.Name].Alive < prj.Parameters.MinHealth {
			//	status.Statuses.Projects[prj.Name].SeqErrorsCount++
			//} else {
			//	if status.Statuses.Projects[prj.Name].SeqErrorsCount > 0 {
			//		status.Statuses.Projects[prj.Name].SeqErrorsCount--
			//	} else {
			//		status.Statuses.Projects[prj.Name].SeqErrorsCount = 0
			//	}
			//}
			if status.Statuses.Projects[prj.Name].FailsCount > prj.Parameters.AllowFails {
				errorMessage := fmt.Sprintf("Critical alert prj %s", prj.Name)
				project := projects.Project{Project: prj}
				project.ProjectCritAlert(errors.New(errorMessage))
			}
		}
	}
	return time.Since(startTime)
}
