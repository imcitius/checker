package check

import (
	"fmt"
	"my/checker/actors"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
	"time"
)

func chooseChannelAndSendAlert(project *projects.Project, check *config.Check, err error) {
	if check.Severity == "critical" || check.Severity == "crit" {
		project.ProjectCritAlert(err)
	} else {
		project.ProjectAlert(err)
	}
}

func EvaluateCheckResult(project *projects.Project, healthcheck *config.Healthcheck, check *config.Check, tempErr error, checkRandomId string, t time.Duration) {

	statusByUUID := status.Statuses.Checks[check.UUid]
	statusByProject := status.Statuses.Projects[project.Name]
	//config.Log.Panicf("%+v", check.Actors)
	if tempErr != nil {
		var err error
		if check.IsCritical() {
			err = fmt.Errorf("(%s) CRITICAL %s", checkRandomId, tempErr.Error())
			config.Log.Errorf("(%s) CRITICAL failure: %+v, took %d millisec", checkRandomId, err, t.Milliseconds())
		} else {
			err = fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
			config.Log.Errorf("(%s) failure: %+v, took %d millisec", checkRandomId, err, t.Milliseconds())
		}
		//config.Log.Debugf("Check mode: %s", status.GetCheckMode(check))

		if check.AllowFails > 0 {
			if statusByUUID.SeqErrorsCount >= check.AllowFails {
				if status.GetCheckMode(check) != "quiet" {
					chooseChannelAndSendAlert(project, check, err)
				}
			}
		} else {
			if status.GetCheckMode(check) != "quiet" {
				chooseChannelAndSendAlert(project, check, err)
			}
		}

		if statusByProject.SeqErrorsCount < project.Parameters.AllowFails {
			statusByProject.SeqErrorsCount++
		}

		if statusByUUID.SeqErrorsCount < check.AllowFails {
			statusByUUID.SeqErrorsCount++
		}

		err = AddCheckError(project, healthcheck, check)
		if err != nil {
			config.Log.Errorf("Metric count error: %v", err)
		}

		//if _, ok := status.Statuses.Checks[check.UUid]; ok {
		//config.Log.Infof("err name: %s, count: %d, last: %v", check.Host, status.Statuses.Checks[check.UUid].ExecuteCount, status.Statuses.Checks[check.UUid].LastResult)
		if check.Actors.Up != "" && statusByUUID.ExecuteCount > 1 {
			if statusByUUID.LastResult {
				config.Log.Debugf("atype: %s", actors.GetActorByName(check.Actors.Down).Type)
				actor := actors.ActorCollection[actors.GetActorByName(check.Actors.Down).Type]

				if actor == nil {
					config.Log.Errorf("down actor %s error: empty actor", check.Actors.Down)
				} else {
					if err := actor.Do(actors.GetActorByName(check.Actors.Down)); err != nil {
						config.Log.Errorf("down actor %s error: %s", check.Actors.Down, err)
					}
				}
			}
		}
		//}

		status.Statuses.Checks[check.UUid].LastResult = false

	} else {
		config.Log.Warnf("(%s) %s success, took %d millisec", checkRandomId, check.Name, t.Milliseconds())
		metrics.CheckDuration.WithLabelValues(project.Name, healthcheck.Name, check.UUid, check.Type).Set(float64(t.Milliseconds()))

		if statusByProject.SeqErrorsCount > 0 {
			statusByProject.SeqErrorsCount--
		}

		if statusByUUID.SeqErrorsCount > 0 {
			statusByUUID.SeqErrorsCount = 0
		}

		if _, ok := status.Statuses.Checks[check.UUid]; ok {
			//config.Log.Infof("good name: %s, count: %d, last: %v", check.Host, status.Statuses.Checks[check.UUid].ExecuteCount, status.Statuses.Checks[check.UUid].LastResult)
			if check.Actors.Down != "" && status.Statuses.Checks[check.UUid].ExecuteCount > 1 {
				if !status.Statuses.Checks[check.UUid].LastResult {

					config.Log.Debugf("Actor: %+v", actors.GetActorByName(check.Actors.Up))
					actor := actors.ActorCollection[actors.GetActorByName(check.Actors.Up).Type]
					//config.Log.Infof("actor: %+v", actor)

					if actor == nil {
						config.Log.Errorf("up actor %s error: empty actor", check.Actors.Up)
					} else {
						if err := actor.Do(actors.GetActorByName(check.Actors.Up)); err != nil {
							config.Log.Errorf("up actor %s error: %s", check.Actors.Up, err)
						}
					}
				}
			}
		}
		statusByUUID.LastResult = true
		statusByProject.Alive++
	}
}
