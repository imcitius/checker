package scheduler

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"github.com/teris-io/shortid"
	"math"
	"math/rand"
	checks "my/checker/checks"
	"my/checker/config"
	projects "my/checker/projects"
	"sync"
	"time"
)

func getRandomId() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	checkRuntimeId, _ := sid.Generate()
	return checkRuntimeId
}

func runReports(timeout string) {
	config.Log.Debug("runReports")
	for _, project := range config.Config.Projects {
		if project.Parameters.PeriodicReport == timeout {
			err := projects.SendReport(&project)
			if err != nil {
				config.Log.Printf("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
		//project.SendReport()
	}
}

func runAlerts(timeout string) {
	config.Log.Debug("runAlerts")
	for _, project := range config.Config.Projects {
		if project.Parameters.RunEvery == timeout {
			if project.ErrorsCount > project.Parameters.MinHealth {
				projects.AddFail(&project)
			} else {
				projects.DecFail(&project)
			}
			if project.FailsCount > project.Parameters.AllowFails {
				errorMessage := fmt.Sprintf("Critical alert project %s", project.Name)
				projects.CritAlert(&project, "crit", errors.New(errorMessage))
			}
		}
	}
}

func runChecks(timeout string) {
	config.Log.Debug("runChecks")

	for _, project := range config.Config.Projects {
		for _, healthcheck := range project.Healtchecks {
			for _, check := range healthcheck.Checks {
				config.Log.Debug(check.Host)
				if timeout == healthcheck.Parameters.RunEvery || timeout == project.Parameters.RunEvery {
					healthcheck.RunCount++
					check.RunCount++
					checkRandomId := getRandomId()
					config.Log.Infof("(%s) Checking project '%s' check '%s' (type: %s) ... ", checkRandomId, project.Name, healthcheck.Name, check.Type)
					startTime := time.Now()
					tempErr := checks.Execute(&check, &project)
					endTime := time.Now()
					t := endTime.Sub(startTime)
					if tempErr != nil {
						err := fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
						config.Log.Infof("(%s) failure: %+v, took %d millisec\n", checkRandomId, err, t.Milliseconds())
						if check.Mode != "quiet" {
							projects.Alert(&project, "noncrit", err)
						}
						projects.AddError(&project)
						checks.HealtcheckAddError(&healthcheck)
						checks.CheckAddError(&check)
					} else {
						config.Log.Infof("(%s) success, took %d millisec\n", checkRandomId, t.Milliseconds())
						projects.DecError(&project)
					}
				}
			}
		}
	}
}

func RunScheduler(signalCh chan bool, wg *sync.WaitGroup) {

	StartTime := time.Now()

	timerStep, err := time.ParseDuration(viper.GetString("defaults.timer_step"))
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
		config.ScheduleLoop++
	}
}
