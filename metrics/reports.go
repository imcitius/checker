package metrics

import (
	"fmt"
	"my/checker/config"
	"runtime"
)

func getAllProjectsHealthchecks() string {
	var output string

	for _, p := range config.Config.Projects {

		//config.Log.Debug(p.Name)
		//config.Log.Debug(p.Parameters.RunEvery)
		//config.Log.Debug(m.Projects[p.Name].RunCount)
		//config.Log.Debug(m.Projects[p.Name].ErrorsCount)
		//config.Log.Debug(m.Projects[p.Name].FailsCount)

		output += fmt.Sprintf("Project: %s, each %s\tRuns: %d, errors: %d, current alive: %d, FAILS: %d \n", p.Name, p.Parameters.RunEvery, Metrics.Projects[p.Name].RunCount, Metrics.Projects[p.Name].ErrorsCount, Metrics.Projects[p.Name].Alive, Metrics.Projects[p.Name].FailsCount)

		for _, h := range p.Healtchecks {
			output += fmt.Sprintf("\tHealthCheck: %s\truns: %d, errors: %d\n", h.Name, Metrics.Healthchecks[h.Name].RunCount, Metrics.Healthchecks[h.Name].ErrorsCount)
			for _, c := range h.Checks {
				output += fmt.Sprintf("\t\tCheck: %s\thost: %s, runs: %d, errors: %d\n", c.Type, c.Host, Metrics.Checks[c.UUid].RunCount, Metrics.Checks[c.UUid].ErrorsCount)
			}
		}
	}
	return output
}

func getCeasedProjectsHealthchecks() string {
	var output string

	for _, p := range config.Config.Projects {
		if p.Parameters.Mode == "quiet" {
			output += fmt.Sprintf("Project: %s\n", p.Name)
		}

		for _, h := range p.Healtchecks {
			for _, c := range h.Checks {
				if c.Mode == "quiet" {
					output += fmt.Sprintf("\t\tCheck: %s\t host: %s\n", c.Type, c.Host)
				}
			}
		}
	}
	return output
}

func getMetrics() string {
	var (
		output                                         string
		projectRuns, alertsSent, critSent, nonCritSent int
		commandReceived, commandSent                   int
	)

	for _, p := range config.Config.Projects {
		projectRuns += Metrics.Projects[p.Name].RunCount
	}

	for _, c := range Metrics.Alerts {
		alertsSent += c.AlertCount
		critSent += c.Critical
		nonCritSent += c.NonCritical
		commandSent += c.CommandAns
		commandReceived += c.CommandReqs
		config.Log.Debugf("Counter: %s", c.Name)
	}

	output += fmt.Sprintf("Loop cycles (%s): %d\n", config.Config.Defaults.TimerStep, config.ScheduleLoop)
	output += fmt.Sprintf("Total checks runs: %d\n\n", projectRuns)
	output += fmt.Sprintf("Total alerts/reports sent: %d\n", alertsSent)
	output += fmt.Sprintf("\tNonCritical alerts sent: %d\n", nonCritSent)
	output += fmt.Sprintf("\tCritical alerts sent: %d\n", critSent)
	output += fmt.Sprintf("\tCommand messages received: %d, sent: %d\n", commandReceived, commandSent)
	output += fmt.Sprintf("\tCurently running goroutines number: %d\n", runtime.NumGoroutine())

	return output
}

func GenRuntimeStats() string {
	var output string

	output += "Total projects\n"
	output += "==========================\n\n"
	output += getAllProjectsHealthchecks()
	output += "\n\n\nCeased projects and checks\n"
	output += "==========================\n\n"
	output += getCeasedProjectsHealthchecks()
	output += "\n\n\nRuntime metrics\n"
	output += "==========================\n\n"
	output += getMetrics()

	return output
}
