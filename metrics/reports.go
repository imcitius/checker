package metrics

import (
	"fmt"
	"my/checker/config"
	"runtime"
)

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

	output += fmt.Sprintf("Loop cycles (%s): %d\n", config.Config.Defaults.TimerStep, config.ScheduleLoop)
	output += fmt.Sprintf("Total checks runs: %d\n\n", projectRuns)
	output += fmt.Sprintf("Total alerts/reports sent: %d\n", alertsSent)
	output += fmt.Sprintf("\tNonCritical alerts sent: %d\n", nonCritSent)
	output += fmt.Sprintf("\tCritical alerts sent: %d\n", critSent)
	output += fmt.Sprintf("\tCommand messages received: %d, sent: %d\n", commandReceived, commandSent)
	output += fmt.Sprintf("\tCurently running goroutines number: %d\n", runtime.NumGoroutine())

	return output
}

func GenTextRuntimeStats() string {
	var output string

	output += "Total projects\n"
	output += "==========================\n\n"
	//output += getAllProjectsHealthchecks()
	output += "\n\n\nCeased projects and checks\n"
	output += "==========================\n\n"
	output += getCeasedProjectsHealthchecks()
	output += "\n\n\nRuntime metrics\n"
	output += "==========================\n\n"
	output += getMetrics()

	return output
}
