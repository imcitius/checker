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

func EvaluateCheckResult(project *projects.Project, healthcheck *config.Healthcheck, check *config.Check, tempErr error, checkRandomId string, t time.Duration) {

	//config.Log.Panicf("%+v", check.Actors)

	if tempErr != nil {
		err := fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
		config.Log.Warnf("(%s) failure: %+v, took %d millisec\n", checkRandomId, err, t.Milliseconds())
		//config.Log.Debugf("Check mode: %s", status.GetCheckMode(check))

		if check.AllowFails > 0 {
			if status.Statuses.Checks[check.UUid].SeqErrorsCount >= check.AllowFails {
				if status.GetCheckMode(check) != "quiet" {
					project.ProjectAlert(err)
				}
			}
		}

		if status.Statuses.Projects[project.Name].SeqErrorsCount < project.Parameters.AllowFails {
			status.Statuses.Projects[project.Name].SeqErrorsCount++
		}

		if status.Statuses.Checks[check.UUid].SeqErrorsCount < check.AllowFails {
			status.Statuses.Checks[check.UUid].SeqErrorsCount++
		}

		err = AddCheckError(project, healthcheck, check)
		if err != nil {
			config.Log.Errorf("Metric count error: %v", err)
		}

		if _, ok := status.Statuses.Checks[check.UUid]; ok {
			if check.Actors.Up != "" && status.Statuses.Checks[check.UUid].ExecuteCount > 1 {
				if status.Statuses.Checks[check.UUid].LastResult {
					config.Log.Debugf("atype: %s", actors.GetActorByName(check.Actors.Down).Type)
					actor := actors.ActorCollection[actors.GetActorByName(check.Actors.Down).Type]

					if actor == nil {
						config.Log.Warnf("down actor %s error: empty actor", check.Actors.Down)
					} else {
						if err := actor.Do(actors.GetActorByName(check.Actors.Down)); err != nil {
							config.Log.Warnf("down actor %s error: %s", check.Actors.Down, err)
						}
					}
				}
			}
		}

		check.LastResult = false

	} else {
		config.Log.Warnf("(%s) success, took %d millisec\n", checkRandomId, t.Milliseconds())
		metrics.CheckDuration.WithLabelValues(project.Name, healthcheck.Name, check.UUid, check.Type).Set(float64(t.Milliseconds()))

		if status.Statuses.Projects[project.Name].SeqErrorsCount > 0 {
			status.Statuses.Projects[project.Name].SeqErrorsCount--
		}

		if status.Statuses.Checks[check.UUid].SeqErrorsCount > 0 {
			status.Statuses.Checks[check.UUid].SeqErrorsCount--
		}

		if _, ok := status.Statuses.Checks[check.UUid]; ok {
			if check.Actors.Down != "" && status.Statuses.Checks[check.UUid].ExecuteCount > 1 {
				if !status.Statuses.Checks[check.UUid].LastResult {

					config.Log.Debugf("Actor: %+v", actors.GetActorByName(check.Actors.Up))
					actor := actors.ActorCollection[actors.GetActorByName(check.Actors.Up).Type]
					//config.Log.Infof("actor: %+v", actor)

					if actor == nil {
						config.Log.Warnf("up actor %s error: empty actor", check.Actors.Up)
					} else {
						if err := actor.Do(actors.GetActorByName(check.Actors.Up)); err != nil {
							config.Log.Warnf("up actor %s error: %s", check.Actors.Up, err)
						}
					}
				}
			}
		}
		check.LastResult = true
		status.Statuses.Projects[project.Name].Alive++
	}
}
