package scheduler

import (
	"errors"
	"fmt"
	"github.com/teris-io/shortid"
	"math"
	"math/rand"
	"my/checker/alerts"
	checks "my/checker/checks"
	"my/checker/config"
	"my/checker/metrics"
	"my/checker/status"
	"sync"
	"time"
)

var Config = &config.Config

func getRandomId() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	checkRuntimeId, _ := sid.Generate()
	return checkRuntimeId
}

func runReports(timeout string) {
	config.Log.Debug("runReports")
	for _, project := range Config.Projects {
		if project.Parameters.PeriodicReport == timeout {
			err := alerts.ProjectSendReport(&project)
			if err != nil {
				config.Log.Printf("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
	}
}

func runAlerts(timeout string) {
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
				alerts.ProjectCritAlert(&project, errors.New(errorMessage))
			}
		}
	}
}

func runChecks(timeout string) {
	config.Log.Debug("runChecks")

	for _, project := range Config.Projects {
		for _, healthcheck := range project.Healtchecks {

			status.Statuses.Projects[project.Name].Alive = 0

			config.Log.Debugf("Total checks %+v", healthcheck.Checks)
			for _, check := range healthcheck.Checks {
				config.Log.Debugf("Now checking %s", check.Host)
				if timeout == healthcheck.Parameters.RunEvery || timeout == project.Parameters.RunEvery {

					checkRandomId := getRandomId()
					config.Log.Infof("(%s) Checking project '%s' check '%s' (type: %s) ... ", checkRandomId, project.Name, healthcheck.Name, check.Type)

					startTime := time.Now()
					tempErr := checks.Execute(&project, &healthcheck, &check)
					endTime := time.Now()

					t := endTime.Sub(startTime)
					if tempErr != nil {
						err := fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
						config.Log.Infof("(%s) failure: %+v, took %d millisec\n", checkRandomId, err, t.Milliseconds())
						config.Log.Debugf("Check mode: %s", status.GetCheckMode(&check))
						if status.GetCheckMode(&check) != "quiet" {
							alerts.ProjectAlert(&project, err)
						}

						status.Statuses.Projects[project.Name].SeqErrorsCount++
						status.Statuses.Checks[check.UUid].LastResult = false

						err = metrics.AddCheckError(&project, &healthcheck, &check)
						if err != nil {
							config.Log.Errorf("Metric count error: %v", err)
						}

					} else {
						config.Log.Infof("(%s) success, took %d millisec\n", checkRandomId, t.Milliseconds())
						status.Statuses.Projects[project.Name].SeqErrorsCount--

						status.Statuses.Projects[project.Name].Alive++
						status.Statuses.Checks[check.UUid].LastResult = true
					}
				}
			}
		}
	}
}

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {

	StartTime := time.Now()

	timerStep, err := time.ParseDuration(config.Viper.GetString("defaults.timer_step"))
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
			dif := float64(t.Sub(StartTime) / time.Second)

			for i, timeout := range config.Timeouts.Periods {
				config.Log.Debugf("Got timeout #%d: %s", i, timeout)

				tf, err := time.ParseDuration(timeout)
				if err != nil {
					config.Log.Errorf("Cannot parse timeout: %s", err)
				}
				config.Log.Debugf("Parsed timeout #%d: %+v", i, tf)

				if math.Remainder(dif, tf.Seconds()) == 0 {
					config.Log.Debugf("Time: %v\nTimeout: %v\n===\n\n", t, timeout)

					config.Log.Infof("Schedule: %s", timeout)

					go runChecks(timeout)
					go runReports(timeout)
					runAlerts(timeout)
				}
			}
		}

		metrics.SchedulerLoops.Inc()
	}
}
