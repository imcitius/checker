package scheduler

import (
	"errors"
	"fmt"
	"github.com/teris-io/shortid"
	"math"
	"math/rand"
	"my/checker/alerts"
	"my/checker/catalog"
	checks "my/checker/checks"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
	"sync"
	"time"
)

var Config = &config.Config

func GetRandomId() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	checkRuntimeId, _ := sid.Generate()
	return checkRuntimeId
}

func runReports(timeout string) {

	config.Log.Infof("runReports")
	for _, p := range Config.Projects {
		//config.Log.Info("runReports 1 %s", p.Name)
		//config.Log.Info("runReports 2 %s", p.Parameters.Mode)
		//config.Log.Info("runReports 3 %s", p.Parameters.Mode)
		//config.Log.Info("runReports 4 %s", status.Statuses.Projects[p.Name].Mode)
		//config.Log.Info("runReports 6 %s", p.Parameters.PeriodicReport)

		schedTimeout, err := time.ParseDuration(timeout)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse duration %s", err)
		}
		projTimeout, err := time.ParseDuration(p.Parameters.PeriodicReport)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse duration %s", err)
		}

		if schedTimeout >= projTimeout {
			project := projects.Project{p}
			//config.Log.Info("runReports 5 %s", project.GetMode())
			err := project.ProjectSendReport()
			if err != nil {
				config.Log.Infof("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
	}

	if config.Config.Defaults.Parameters.PeriodicReport == timeout {
		if status.MainStatus == "quiet" {
			reportMessage := "All messages ceased"
			alerts.SendChatOps(reportMessage)
		}
	}
}

func runCritAlerts(timeout string) {
	config.Log.Debug("runAlerts")

	for _, project := range Config.Projects {
		if project.Parameters.RunEvery == timeout {
			//if status.Statuses.Projects[project.Name].Alive < project.Parameters.MinHealth {
			//	status.Statuses.Projects[project.Name].SeqErrorsCount++
			//} else {
			//	if status.Statuses.Projects[project.Name].SeqErrorsCount > 0 {
			//		status.Statuses.Projects[project.Name].SeqErrorsCount--
			//	} else {
			//		status.Statuses.Projects[project.Name].SeqErrorsCount = 0
			//	}
			//}
			if status.Statuses.Projects[project.Name].FailsCount > project.Parameters.AllowFails {
				errorMessage := fmt.Sprintf("Critical alert project %s", project.Name)
				project := projects.Project{project}
				project.ProjectCritAlert(errors.New(errorMessage))
			}
		}
	}
}

func runChecks(timeout string) {
	config.Log.Debug("runChecks")

	checkProjects(timeout)
	catalog.CheckCatalog(timeout)
}

func checkProjects(timeout string) {
	for _, project := range Config.Projects {
		for _, healthcheck := range project.Healthchecks {

			status.Statuses.Projects[project.Name].Alive = 0

			executeHealthcheck(&projects.Project{project}, &healthcheck, timeout)
		}
	}
}

func executeHealthcheck(project *projects.Project, healthcheck *config.Healthcheck, timeout string) {
	config.Log.Debugf("Total checks %+v", healthcheck.Checks)
	for _, check := range healthcheck.Checks {
		config.Log.Debugf("Now checking %s", check.Host)
		if timeout == healthcheck.Parameters.RunEvery || timeout == project.Parameters.RunEvery {

			checkRandomId := GetRandomId()
			config.Log.Warnf("(%s) Checking project '%s' check '%s' (type: %s) ... ", checkRandomId, project.Name, healthcheck.Name, check.Type)

			startTime := time.Now()

			err := checks.AddCheckRunCount(project, healthcheck, &check)
			if err != nil {
				config.Log.Errorf("Metric count error: %v", err)
			}

			tempErr := checks.Execute(project, &check)
			endTime := time.Now()
			t := endTime.Sub(startTime)
			checks.EvaluateCheckResult(project, healthcheck, &check, tempErr, checkRandomId, t)
		}
	}
}

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {

	timerStep, err := time.ParseDuration(config.Koanf.String("defaults.timer_step"))
	if err != nil {
		config.Log.Fatal(err)
	}

	Ticker := time.NewTicker(timerStep)
	timerStepSeconds := timerStep.Seconds()

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
			uptime := float64(t.Sub(config.StartTime) / time.Second)

			for _, timeout := range config.Timeouts.Periods {
				config.Log.Debugf("Looking for projects with timeout: %s", timeout)

				tf, err := time.ParseDuration(timeout)
				if err != nil {
					config.Log.Errorf("Cannot parse timeout: %s", err)
				}

				config.Log.Debugf("===\nUptime: %v", uptime)

				roundUptime := math.Round(uptime/timerStepSeconds) * timerStepSeconds
				if math.Remainder(roundUptime, tf.Seconds()) == 0 {
					config.Log.Debugf("===\nTime: %v\n---\n\n", t)

					config.Log.Infof("Checking run_every: %s", timeout)

					checksStartTime := time.Now()
					runChecks(timeout)
					checksDuration := time.Since(checksStartTime)

					reportsStartTime := time.Now()
					runReports(timeout)
					reportsDuration := time.Since(reportsStartTime)

					alertsStartTime := time.Now()
					runCritAlerts(timeout)
					alertsDuration := time.Since(alertsStartTime)

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
