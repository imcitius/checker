package catalog

import (
	"fmt"
	"my/checker/alerts"
	checks "my/checker/checks"
	"my/checker/common"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
	"time"
)

func CheckCatalog(timeout string) {

	//config.Log.Infof("Catalog: %+v", config.ProjectsCatalog)

	for _, p := range config.ProjectsCatalog {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				if timeout == h.Parameters.RunEvery || timeout == p.Parameters.RunEvery {

					startTime := time.Now()
					//config.Log.Debugf("check: %+v", c)
					tempErr := checks.Execute(&projects.Project{p}, &c)
					endTime := time.Now()
					t := endTime.Sub(startTime)
					evaluateCheckResult(&projects.Project{p}, &h, &c, tempErr, common.GetRandomId(), t)
				}
			}
		}
	}
}

func evaluateCheckResult(p *projects.Project, h *config.Healthcheck, c *config.Check, tempErr error, checkRandomId string, t time.Duration) {
	if tempErr != nil {
		err := fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
		config.Log.Infof("(%s) failure: %+v, took %d millisec\n", checkRandomId, err, t.Milliseconds())
		//config.Log.Debugf("Check mode: %s", status.GetCheckMode(check))
		if status.GetCheckMode(c) != "quiet" {
			alerts.ProjectAlert(p, err)
		}

		status.Statuses.Projects[p.Name].SeqErrorsCount++
		status.Statuses.Checks[c.UUid].LastResult = false

		err = metrics.AddCheckError(p, h, c)
		if err != nil {
			config.Log.Errorf("Metric count error: %v", err)
		}

	} else {
		config.Log.Infof("(%s) success, took %d millisec\n", checkRandomId, t.Milliseconds())
		metrics.CheckDuration.WithLabelValues(p.Name, h.Name, c.UUid, c.Type).Set(float64(t.Milliseconds()))

		status.Statuses.Projects[p.Name].SeqErrorsCount--

		status.Statuses.Projects[p.Name].Alive++
		status.Statuses.Checks[c.UUid].LastResult = true
	}
}
