package main

import (
	"math"
	"time"
)

func runChecks(timeout uint) {

	for _, project := range Config.Projects {
		if project.Parameters.RunEvery == timeout {
			if project.Checks.URLChecks != nil {
				// log.Printf("HTTP checks for project %s\n", project.Name)
				go runHTTPCheck(project)
			}

			if project.Checks.ICMPPingChecks != nil {
				// log.Printf("ICMP ping checks for project %s\n", project.Name)
				go runICMPPingChecks(project)
			}

			if project.Checks.TCPPingChecks != nil {
				// log.Printf("TCP ping checks for project %s\n", project.Name)
				go runTCPPingChecks(project)
			}

		}
	}

}

func sendAlerts() {

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
					// fmt.Printf("Time: %v\nTimeout: %v\n===\n\n", t, timeout)
					runChecks(timeout)
					sendAlerts()
				}
			}
		}
	}
}
