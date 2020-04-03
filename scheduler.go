package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"time"
)

func runChecks(timeout int) {
	for _, project := range Config.Projects {
		for _, healthcheck := range project.Healtchecks {
			for _, check := range healthcheck.Checks {
				//log.Println(check.Host)
				if timeout == healthcheck.Parameters.RunEvery || timeout == project.Parameters.RunEvery {
					log.Printf("Checking project '%s' check '%s' ... ", project.Name, check.Type)
					startTime := time.Now()
					err := check.Execute(project)
					endTime := time.Now()
					t := endTime.Sub(startTime)
					if err != nil {
						log.Printf("failure: %+v, took %d millisec\n", err, t.Milliseconds())
						if check.Mode != "quiet" {
							project.Alert(err)
						}
						project.AddError()
					} else {
						log.Printf("success, took %d millisec\n", t.Milliseconds())
						project.DecError()
					}
				}
			}
		}
	}
}

func runAlerts(timeout int) {
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

func runReports(timeout int) {
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
	Ticker := time.NewTicker(time.Duration(Config.Defaults.TimerStep) * time.Second)

	for {
		select {
		case <-done:
			return
		case t := <-Ticker.C:
			dif := float64(t.Sub(StartTime) / time.Second)
			for _, timeout := range Timeouts.periods {
				if math.Remainder(dif, float64(timeout)) == 0 {
					log.Printf("Time: %v\nTimeout: %v\n===\n\n", t, timeout)
					go runChecks(timeout)
					go runReports(timeout)
					runAlerts(timeout)
				}
			}
		}
	}
}
