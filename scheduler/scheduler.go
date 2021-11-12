package scheduler

import (
	"errors"
	"fmt"
	"math"
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

func runReports(timeout string) time.Duration {
	startTime := time.Now()
	config.Log.Debugf("runReports")
	for _, p := range Config.Projects {
		config.Log.Debugf("runReports 0: %s\n", timeout)
		config.Log.Debugf("runReports 1: %s\n", p.Name)
		config.Log.Debugf("runReports 2: %s\n", p.Parameters.Mode)
		config.Log.Debugf("runReports 4: %s\n", status.Statuses.Projects[p.Name].Mode)
		config.Log.Debugf("runReports 6: %s\n", p.Parameters.PeriodicReport)

		schedTimeout, err := time.ParseDuration(timeout)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse duration %s", err)
		}
		config.Log.Debugf("schedTimeout: %s\n", schedTimeout)

		reportsTimeout, err := time.ParseDuration(p.Parameters.PeriodicReport)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse duration %s", err)
		}
		config.Log.Debugf("reportsTimeout: %s\n", reportsTimeout)

		if schedTimeout >= reportsTimeout {
			project := projects.Project{p}
			config.Log.Debugf("runReports 10: %s", project.GetMode())
			err := project.ProjectSendReport()
			if err != nil {
				config.Log.Errorf("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
	}

	if config.Config.Defaults.Parameters.PeriodicReport == timeout {
		if status.MainStatus == "quiet" {
			reportMessage := "All messages ceased"
			alerts.SendChatOps(reportMessage)
		}
	}
	return time.Since(startTime)
}

func sendCritAlerts(timeout string) time.Duration {
	startTime := time.Now()
	config.Log.Debug("sendCritAlerts")

	for _, prj := range Config.Projects {
		if prj.Parameters.RunEvery == timeout {
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
				project := projects.Project{prj}
				project.ProjectCritAlert(errors.New(errorMessage))
			}
		}
	}
	return time.Since(startTime)
}

func runChecks(timeout string) time.Duration {
	startTime := time.Now()
	config.Log.Debug("runChecks")

	checkProjects(timeout)
	catalog.CheckCatalog(timeout)
	return time.Since(startTime)
}

func checkProjects(timeout string) {
	for _, project := range Config.Projects {
		for _, healthcheck := range project.Healthchecks {

			status.Statuses.Projects[project.Name].Alive = 0

			ExecuteHealthcheck(&projects.Project{project}, &healthcheck, timeout)
		}
	}
}

func ExecuteHealthcheck(project *projects.Project, healthcheck *config.Healthcheck, timeout string) {
	config.Log.Debugf("Total checks %+v", healthcheck.Checks)
	for _, check := range healthcheck.Checks {
		checkRandomId := common.GetRandomId()
		config.Log.Debugf("(%s) Evaluating check %s", checkRandomId, check.Name)
		if timeout == healthcheck.Parameters.RunEvery || timeout == project.Parameters.RunEvery {
			config.Log.Warnf("(%s) Checking project/healthcheck/check: '%s/%s/%s(%s)'", checkRandomId, project.Name, healthcheck.Name, check.Name, check.Type)

			err := checks.AddCheckRunCount(project, healthcheck, &check)
			if err != nil {
				config.Log.Errorf("Metric count error: %v", err)
			}
			duration, tempErr := checks.Execute(project, &check)
			checks.EvaluateCheckResult(project, healthcheck, &check, tempErr, checkRandomId, duration, "ExecuteHealthcheck")
		} else {
			config.Log.Debugf("(%s) check %s timeout is not eligible for checking", checkRandomId, check.Name)
		}
	}
}

func timeIsDivisible(uptime time.Duration, timer time.Duration) bool {
	config.Log.Infof("Checking divisibility: uptime/timer --- %f/%f", uptime.Seconds(), timer.Seconds())
	if math.Remainder(uptime.Seconds(), timer.Seconds()) == 0 {
		config.Log.Debugf("Divisible")
		return true
	}
	config.Log.Debugf("Not divisible")
	return false
}

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {

	timerStep, err := time.ParseDuration(config.Koanf.String("defaults.timer_step"))
	if err != nil {
		config.Log.Fatal(err)
	}

	Ticker := time.NewTicker(timerStep)

	config.Log.Debug("Scheduler started")
	config.Log.Debugf("Timeouts: %+v", config.Timeouts.Periods)

	for {
		config.Log.Debugf("Scheduler loop #: %d", config.ScheduleLoop)

		select {
		case <-signalCh:
			config.Log.Infof("Exit scheduler")
			wg.Done()
			return
		case t := <-Ticker.C:
			go config.WatchConfig()
			uptime := t.Round(time.Second).Sub(config.StartTime.Round(time.Second))

			for _, timeout := range config.Timeouts.Periods {
				config.Log.Debugf("Looking for projects with timeout: %s", timeout)

				tf, err := time.ParseDuration(timeout)
				if err != nil {
					config.Log.Errorf("Cannot parse timeout: %s", err)
				}
				config.Log.Debugf("===\nUptime: %v", uptime)

				if timeIsDivisible(uptime, tf) {
					config.Log.Debugf("===\nTime: %v\n---\n\n", t)
					config.Log.Infof("Checking run_every: %s", timeout)

					checksDuration := runChecks(timeout)
					reportsDuration := runReports(timeout)
					alertsDuration := sendCritAlerts(timeout)

					config.Log.Infof("Checks duration: %v msec", checksDuration.Milliseconds())
					config.Log.Infof("Reports duration: %v msec", reportsDuration.Milliseconds())
					config.Log.Infof("Alerts duration: %v msec", alertsDuration.Milliseconds())

					metrics.SchedulerChecksDuration.Set(float64(checksDuration.Milliseconds()))
					metrics.SchedulerReportsDuration.Set(float64(reportsDuration.Milliseconds()))
					metrics.SchedulerAlertsDuration.Set(float64(alertsDuration.Milliseconds()))
				}
			}

		}

		metrics.SchedulerLoopConfig.Set(float64(timerStep.Milliseconds()))
		metrics.SchedulerLoops.Inc()
		config.ScheduleLoop++
	}
}
