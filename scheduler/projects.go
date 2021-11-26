package scheduler

import (
	"my/checker/catalog"
	checks "my/checker/checks"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
	"sync"
	"time"
)

func runProjectTickers(t *config.Ticker, wg *sync.WaitGroup, signalCh chan bool) {
	config.Log.Debugf("Starting checks tickers")
	config.Wg.Add(1)
	go func(ticker *config.Ticker) {
		defer wg.Done()

		config.Log.Infof("Waiting for ticker %s", ticker.Description)
		defer config.Log.Infof("Finished ticker %s", ticker.Description)
		for {
			select {
			case <-signalCh:
				config.Log.Infof("Exit ticker")
				return
			case tick := <-ticker.Duration.C:
				uptime := tick.Round(time.Second).Sub(config.StartTime.Round(time.Second))
				period := ticker.Description
				config.Log.Infof("Uptime: %s (%s ticker)", uptime, ticker.Description)

				checksDuration := runChecks(period)
				alertsDuration := sendCritAlerts(period)

				config.Log.Infof("Checks duration: %v msec", checksDuration.Milliseconds())
				config.Log.Debugf("Alerts duration: %v msec", alertsDuration.Milliseconds())

				metrics.SchedulerChecksDuration.Set(float64(checksDuration.Milliseconds()))
				metrics.SchedulerAlertsDuration.Set(float64(alertsDuration.Milliseconds()))
			}
		}
	}(t)
}

func runChecks(period string) time.Duration {
	startTime := time.Now()
	config.Log.Debug("runChecks")

	checkProjects(period)
	catalog.CheckCatalog(period)
	return time.Since(startTime)
}

func checkProjects(period string) {
	for _, project := range Config.Projects {
		for _, healthcheck := range project.Healthchecks {
			status.Statuses.Projects[project.Name].Alive = 0
			ExecuteHealthcheck(&projects.Project{Project: project}, &healthcheck, period)
		}
	}
}

func ExecuteHealthcheck(p *projects.Project, healthcheck *config.Healthcheck, period string) {
	config.Log.Debugf("Total checks %+v", healthcheck.Checks)
	for _, check := range healthcheck.Checks {
		checkRandomId := config.GetRandomId()
		config.Log.Debugf("(%s) Evaluating check %s", checkRandomId, check.Name)
		if period == healthcheck.Parameters.Period || period == p.Parameters.Period {
			config.Log.Warnf("(%s) Checking p/healthcheck/check: '%s/%s/%s(%s)'", checkRandomId, p.Name, healthcheck.Name, check.Name, check.Type)

			err := checks.AddCheckRunCount(p, healthcheck, &check)
			if err != nil {
				config.Log.Errorf("Metric count error: %v", err)
			}
			duration, tempErr := checks.Execute(p, &check)
			checks.EvaluateCheckResult(p, healthcheck, &check, tempErr, checkRandomId, duration, "ExecuteHealthcheck")
		} else {
			config.Log.Debugf("(%s) check %s period is not eligible for checking", checkRandomId, check.Name)
		}
	}
}
