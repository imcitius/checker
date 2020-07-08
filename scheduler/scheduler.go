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
	config.Log.Debug("runReports")
	for _, project := range Config.Projects {
		if project.Parameters.PeriodicReport == timeout {
			err := alerts.ProjectSendReport(&projects.Project{project})
			if err != nil {
				config.Log.Infof("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
	}

	if config.Config.Defaults.Parameters.PeriodicReport == timeout {
		if status.MainStatus == "quiet" {
			reportMessage := fmt.Sprintf("All messages ceased")
			alerts.SendChatOps(reportMessage)
		}
	}
}

func runCritAlerts(timeout string) {
	config.Log.Debug("runAlerts")
	for _, project := range Config.Projects {
		if project.Parameters.RunEvery == timeout {
			if status.Statuses.Projects[project.Name].Alive < project.Parameters.MinHealth {
				status.Statuses.Projects[project.Name].SeqErrorsCount++
			} else {
				status.Statuses.Projects[project.Name].SeqErrorsCount--
			}
			if status.Statuses.Projects[project.Name].FailsCount > project.Parameters.AllowFails {
				errorMessage := fmt.Sprintf("Critical alert project %s", project.Name)
				alerts.ProjectCritAlert(&projects.Project{project}, errors.New(errorMessage))
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
			config.Log.Infof("(%s) Checking project '%s' check '%s' (type: %s) ... ", checkRandomId, project.Name, healthcheck.Name, check.Type)

			startTime := time.Now()

			err := metrics.AddCheckRunCount(project, healthcheck, &check)
			if err != nil {
				config.Log.Errorf("Metric count error: %v", err)
			}

			tempErr := checks.Execute(project, &check)
			endTime := time.Now()
			t := endTime.Sub(startTime)
			evaluateCheckResult(project, healthcheck, &check, tempErr, checkRandomId, t)
		}
	}
}

func evaluateCheckResult(project *projects.Project, healthcheck *config.Healthcheck, check *config.Check, tempErr error, checkRandomId string, t time.Duration) {
	if tempErr != nil {
		err := fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
		config.Log.Infof("(%s) failure: %+v, took %d millisec\n", checkRandomId, err, t.Milliseconds())
		config.Log.Debugf("Check mode: %s", status.GetCheckMode(check))
		if status.GetCheckMode(check) != "quiet" {
			alerts.ProjectAlert(project, err)
		}

		status.Statuses.Projects[project.Name].SeqErrorsCount++
		status.Statuses.Checks[check.UUid].LastResult = false

		err = metrics.AddCheckError(project, healthcheck, check)
		if err != nil {
			config.Log.Errorf("Metric count error: %v", err)
		}

	} else {
		config.Log.Infof("(%s) success, took %d millisec\n", checkRandomId, t.Milliseconds())
		metrics.CheckDuration.WithLabelValues(project.Name, healthcheck.Name, check.UUid, check.Type).Set(float64(t.Milliseconds()))

		status.Statuses.Projects[project.Name].SeqErrorsCount--

		status.Statuses.Projects[project.Name].Alive++
		status.Statuses.Checks[check.UUid].LastResult = true
	}
}

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {

	timerStep, err := time.ParseDuration(config.Koanf.String("defaults.timer_step"))
	if err != nil {
		config.Log.Fatal(err)
	}

	Ticker := time.NewTicker(timerStep)
	MaintPeriod := 5 * time.Minute
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

					config.Log.Infof("Timeout: %s", timeout)

					checksStartTime := time.Now()
					runChecks(timeout)
					checksDuration := time.Now().Sub(checksStartTime)

					reportsStartTime := time.Now()
					runReports(timeout)
					reportsDuration := time.Now().Sub(reportsStartTime)

					alertsStartTime := time.Now()
					runCritAlerts(timeout)
					alertsDuration := time.Now().Sub(alertsStartTime)

					config.Log.Warnf("Checks duration: %v msec", checksDuration.Milliseconds())
					config.Log.Warnf("Reports duration: %v msec", reportsDuration.Milliseconds())
					config.Log.Warnf("Alerts duration: %v msec", alertsDuration.Milliseconds())
					metrics.SchedulerChecksDuration.Set(float64(checksDuration.Milliseconds()))
					metrics.SchedulerReportsDuration.Set(float64(reportsDuration.Milliseconds()))
					metrics.SchedulerAlertsDuration.Set(float64(alertsDuration.Milliseconds()))
				}
			}

			if math.Remainder(uptime, MaintPeriod.Seconds()) == 0 {
				config.ClearSecrets()
			}
		}

		metrics.SchedulerLoopConfig.Set(float64(timerStep.Milliseconds()))
		metrics.SchedulerLoops.Inc()
		config.ScheduleLoop++
	}
}
