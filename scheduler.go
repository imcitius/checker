package main

import (
	"errors"
	"fmt"
	"github.com/teris-io/shortid"
	"math"
	"math/rand"
	"time"
)

func getRandomId() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	checkRuntimeId, _ := sid.Generate()
	return checkRuntimeId
}

func runChecks(timeout string) {
	log.Debug("runChecks")

	for _, project := range Config.Projects {
		for _, healthcheck := range project.Healtchecks {
			for _, check := range healthcheck.Checks {
				log.Debug(check.Host)
				if timeout == healthcheck.Parameters.RunEvery || timeout == project.Parameters.RunEvery {
					healthcheck.RunCount++
					check.RunCount++
					checkRandomId := getRandomId()
					log.Infof("(%s) Checking project '%s' check '%s' (type: %s) ... ", checkRandomId, project.Name, healthcheck.Name, check.Type)
					startTime := time.Now()
					tempErr := check.Execute(project)
					endTime := time.Now()
					t := endTime.Sub(startTime)
					if tempErr != nil {
						err := fmt.Errorf("(%s) %s", checkRandomId, tempErr.Error())
						log.Infof("(%s) failure: %+v, took %d millisec\n", checkRandomId, err, t.Milliseconds())
						if check.Mode != "quiet" {
							project.Alert(err)
						}
						project.AddError()
						healthcheck.AddError()
						check.AddError()
					} else {
						log.Infof("(%s) success, took %d millisec\n", checkRandomId, t.Milliseconds())
						project.DecError()
					}
				}
			}
		}
	}
}

func runAlerts(timeout string) {
	log.Debug("runAlerts")
	for _, project := range Config.Projects {
		if project.Parameters.RunEvery == timeout {
			if project.ErrorsCount > project.Parameters.MinHealth {
				project.AddFail()
			} else {
				project.DecFail()
			}
			if project.FailsCount > project.Parameters.AllowFails {
				errorMessage := fmt.Sprintf("Critical alert project %s", project.Name)
				project.CritAlert(errors.New(errorMessage))
			}
		}
	}
}

func runReports(timeout string) {
	log.Debug("runReports")
	for _, project := range Config.Projects {
		if project.Parameters.PeriodicReport == timeout {
			err := project.SendReport()
			if err != nil {
				log.Printf("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
		//project.SendReport()
	}
}

func runScheduler() {

	done := make(chan bool)
	StartTime := time.Now()

	timeStep, err := time.ParseDuration(Config.Defaults.TimerStep)
	if err != nil {
		log.Fatal(err)
	}

	Ticker := time.NewTicker(timeStep)

	log.Debug("Scheduler started")
	log.Debugf("Timeouts: %+v", Timeouts.periods)

	for {
		log.Debugf("Scheduler loop #: %d", ScheduleLoop)
		select {
		case <-done:
			return
		case t := <-Ticker.C:
			dif := float64(t.Sub(StartTime) / time.Second)

			for i, timeout := range Timeouts.periods {
				log.Debugf("Got timeout #%d: %s", i, timeout)

				tf, err := time.ParseDuration(timeout)
				if err != nil {
					log.Errorf("Cannot parse timeout: %s", err)
				}
				log.Debugf("Parsed timeout #%d: %+v", i, tf)

				if math.Remainder(dif, tf.Seconds()) == 0 {
					log.Debugf("Time: %v\nTimeout: %v\n===\n\n", t, timeout)
					go runChecks(timeout)
					go runReports(timeout)
					runAlerts(timeout)
				}
			}
		}
		ScheduleLoop++
	}
}
