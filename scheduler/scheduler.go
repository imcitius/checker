package scheduler

import "C"
import (
	"errors"
	"fmt"
	"my/checker/alerts"
	"my/checker/catalog"
	checks "my/checker/checks"
	"my/checker/common"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
	"sync"
	"time"
)

var Config = &config.Config

func runReports(period string) time.Duration {
	startTime := time.Now()
	config.Log.Debugf("runReports")
	for _, p := range Config.Projects {
		config.Log.Debugf("runReports 0: %s\n", period)
		config.Log.Debugf("runReports 1: %s\n", p.Name)
		config.Log.Debugf("runReports 2: %s\n", p.Parameters.Mode)
		config.Log.Debugf("runReports 4: %s\n", status.Statuses.Projects[p.Name].Mode)
		config.Log.Debugf("runReports 6: %s\n", p.Parameters.ReportPeriod)

		schedPeriod, err := time.ParseDuration(period)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse duration %s", err)
		}
		config.Log.Debugf("schedPeriod: %s\n", schedPeriod)

		reportsPeriod, err := time.ParseDuration(p.Parameters.ReportPeriod)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse duration %s", err)
		}
		config.Log.Debugf("reportsPeriod: %s\n", reportsPeriod)

		if schedPeriod >= reportsPeriod {
			project := projects.Project{Project: p}
			config.Log.Debugf("runReports 10: %s", project.GetMode())
			err := project.ProjectSendReport()
			if err != nil {
				config.Log.Errorf("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
	}

	if config.Config.Defaults.Parameters.ReportPeriod == period {
		if status.MainStatus == "quiet" {
			reportMessage := "All messages ceased"
			alerts.SendChatOps(reportMessage)
		}
	}
	return time.Since(startTime)
}

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

func ExecuteHealthcheck(project *projects.Project, healthcheck *config.Healthcheck, period string) {
	config.Log.Debugf("Total checks %+v", healthcheck.Checks)
	for _, check := range healthcheck.Checks {
		checkRandomId := common.GetRandomId()
		config.Log.Debugf("(%s) Evaluating check %s", checkRandomId, check.Name)
		if period == healthcheck.Parameters.Period || period == project.Parameters.Period {
			config.Log.Warnf("(%s) Checking project/healthcheck/check: '%s/%s/%s(%s)'", checkRandomId, project.Name, healthcheck.Name, check.Name, check.Type)

			err := checks.AddCheckRunCount(project, healthcheck, &check)
			if err != nil {
				config.Log.Errorf("Metric count error: %v", err)
			}
			duration, tempErr := checks.Execute(project, &check)
			checks.EvaluateCheckResult(project, healthcheck, &check, tempErr, checkRandomId, duration, "ExecuteHealthcheck")
		} else {
			config.Log.Debugf("(%s) check %s period is not eligible for checking", checkRandomId, check.Name)
		}
	}
}

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	config.Log.Info("Scheduler started")
	config.Log.Debugf("Timeouts: %+v", config.Timeouts.Periods)

	config.Log.Debugf("Tickers %+v", config.TickersCollection)
	if len(config.TickersCollection) == 0 {
		config.Log.Fatal("No tickers")
	} else {
		for _, ticker := range config.TickersCollection {
			config.Log.Debugf("Looping over tickerz")
			config.Wg.Add(1)
			go func(ticker config.Ticker) {
				defer wg.Done()

				config.Log.Debugf("Waiting for ticker %s", ticker.Description)
				defer config.Log.Debugf("Finished ticker %s", ticker.Description)
				for {
					select {
					case <-signalCh:
						config.Log.Infof("Exit ticker")
						return
					case t := <-ticker.Ticker.C:
						uptime := t.Round(time.Second).Sub(config.StartTime.Round(time.Second))
						period := ticker.Description
						config.Log.Infof("Uptime: %s (%s ticker)", uptime, ticker.Description)

						checksDuration := runChecks(period)
						reportsDuration := runReports(period)
						alertsDuration := sendCritAlerts(period)

						config.Log.Infof("Checks duration: %v msec", checksDuration.Milliseconds())
						config.Log.Debugf("Reports duration: %v msec", reportsDuration.Milliseconds())
						config.Log.Debugf("Alerts duration: %v msec", alertsDuration.Milliseconds())

						metrics.SchedulerChecksDuration.Set(float64(checksDuration.Milliseconds()))
						metrics.SchedulerReportsDuration.Set(float64(reportsDuration.Milliseconds()))
						metrics.SchedulerAlertsDuration.Set(float64(alertsDuration.Milliseconds()))
					}
				}
			}(ticker)
		}
	}
}
