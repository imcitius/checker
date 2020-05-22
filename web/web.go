package web

import (
	"context"
	"fmt"
	"io"
	"my/checker/config"
	projects "my/checker/projects"
	"net/http"
	"sync"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/healthcheck" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	io.WriteString(w, "Ok!\n")
}

func getAllProjectsHealthchecks() string {
	var output string

	for _, p := range config.Config.Projects {
		output += fmt.Sprintf("Project: %s, each %s\tRuns: %d, errors: %d, FAILS: %d \n", p.Name, p.Parameters.RunEvery, projects.GetProjectRuns(&p), projects.GetErrors(&p), projects.GetFails(&p))

		for _, h := range p.Healtchecks {
			output += fmt.Sprintf("\tHealthCheck: %s\truns: %d, errors: %d\n", h.Name, h.RunCount, h.ErrorsCount)
			for _, c := range h.Checks {
				output += fmt.Sprintf("\t\tCheck: %s\thost: %s, runs: %d, errors: %d\n", c.Type, c.Host, c.RunCount, c.ErrorsCount)
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
	)

	for _, p := range config.Config.Projects {
		projectRuns += projects.GetProjectRuns(&p)
	}

	for _, c := range config.Config.Alerts {
		alertsSent += c.AlertCount
		critSent += c.Critical
		nonCritSent += c.NonCritical
		config.Log.Debugf("Counter: %s", c.Name)
	}

	output += fmt.Sprintf("Loop cycles (%s): %d\n", config.Config.Defaults.TimerStep, config.ScheduleLoop)
	output += fmt.Sprintf("Total checks runs: %d\n\n", projectRuns)
	output += fmt.Sprintf("Total alerts/reports sent: %d\n", alertsSent)
	output += fmt.Sprintf("\tNonCritical alerts sent: %d\n", nonCritSent)
	output += fmt.Sprintf("\tCritical alerts sent: %d\n", critSent)

	return output
}

func runtimeStats(w http.ResponseWriter, r *http.Request) {

	var output string

	if r.URL.Path != "/stats" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	output += "Total projects\n"
	output += "==========================\n\n"
	output += getAllProjectsHealthchecks()
	output += "\n\n\nCeased projects and checks\n"
	output += "==========================\n\n"
	output += getCeasedProjectsHealthchecks()
	output += "\n\n\nRuntime metrics\n"
	output += "==========================\n\n"
	output += getMetrics()

	io.WriteString(w, output)
}

func WebInterface(webSignalCh chan bool, wg *sync.WaitGroup) {
	var server *http.Server
	defer wg.Done()

	if config.Config.Defaults.HTTPEnabled != "" {
		return
	}
	var addr string = fmt.Sprintf(":%s", config.Config.Defaults.HTTPPort)
	server = new(http.Server)
	server.Addr = addr
	config.Log.Infof("HTTP listen on: %s", addr)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/healthcheck", healthCheck)
	http.HandleFunc("/stats", runtimeStats)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			config.Log.Fatalf("ListenAndServe: %s", err)
		}
	}()

	select {
	case <-webSignalCh:

		config.Log.Infof("Exit web interface")
		if err := server.Shutdown(context.Background()); err != nil {
			config.Log.Infof("Web server shutdown failed: %s", err)
		}
		return
	}
}
