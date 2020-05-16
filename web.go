package main

import (
	"fmt"
	"io"
	"net/http"
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

	for _, p := range Config.Projects {
		output += "Project: " + p.Name + "\n"

		for _, h := range p.Healtchecks {
			for _, c := range h.Checks {
				output += "\tCheck: " + c.Type + "\t host: " + c.Host + "\n"
			}
		}
	}
	return output
}

func getCeasedProjectsHealthchecks() string {
	var output string

	for _, p := range Config.Projects {
		if p.Parameters.Mode == "quiet" {
			output += "Project: " + p.Name + "\n"
		}

		for _, h := range p.Healtchecks {
			for _, c := range h.Checks {
				if c.Mode == "quiet" {
					output += "\tCheck: " + c.Type + "\t host: " + c.Host + "\n"
				}
			}
		}
	}
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
	output += getAllProjectsHealthchecks()
	output += "Ceased projects and checks\n"
	output += getCeasedProjectsHealthchecks()
	output += "==========================\n\n"

	io.WriteString(w, output)
}

func webInterface() {
	if Config.Defaults.HTTPEnabled != "" {
		return
	}
	var addr string = fmt.Sprintf(":%s", Config.Defaults.HTTPPort)
	log.Infof("HTTP listen on: %s", addr)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/healthcheck", healthCheck)
	http.HandleFunc("/stats", runtimeStats)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}
