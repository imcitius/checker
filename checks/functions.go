package check

import (
	"fmt"
	"my/checker/alerts"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
	"time"
)

func EvaluateCheckResult(project *projects.Project, healthcheck *config.Healthcheck, check *config.Check, tempErr error, checkRandomId string, t time.Duration) {
	if tempErr != nil {
		err := fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
		config.Log.Warnf("(%s) failure: %+v, took %d millisec\n", checkRandomId, err, t.Milliseconds())
		//config.Log.Debugf("Check mode: %s", status.GetCheckMode(check))

		if check.AllowFails > 0 {
			if status.Statuses.Checks[check.UUid].SeqErrorsCount >= check.AllowFails {
				if status.GetCheckMode(check) != "quiet" {
					alerts.ProjectAlert(project, err)
				}
			}
		}

		if status.Statuses.Projects[project.Name].SeqErrorsCount < project.Parameters.AllowFails {
			status.Statuses.Projects[project.Name].SeqErrorsCount++
		}

		if status.Statuses.Checks[check.UUid].SeqErrorsCount < check.AllowFails {
			status.Statuses.Checks[check.UUid].SeqErrorsCount++
		}

		err = metrics.AddCheckError(project, healthcheck, check)
		if err != nil {
			config.Log.Errorf("Metric count error: %v", err)
		}

	} else {
		config.Log.Warnf("(%s) success, took %d millisec\n", checkRandomId, t.Milliseconds())
		metrics.CheckDuration.WithLabelValues(project.Name, healthcheck.Name, check.UUid, check.Type).Set(float64(t.Milliseconds()))

		if status.Statuses.Projects[project.Name].SeqErrorsCount > 0 {
			status.Statuses.Projects[project.Name].SeqErrorsCount--
		}

		if status.Statuses.Checks[check.UUid].SeqErrorsCount > 0 {
			status.Statuses.Checks[check.UUid].SeqErrorsCount--
		}

		status.Statuses.Projects[project.Name].Alive++
	}
}
