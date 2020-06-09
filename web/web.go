package web

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/semaphore"
	"io"
	"my/checker/config"
	"net/http"
	_ "net/http/pprof"
)

var Config *config.ConfigFile = &config.Config

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

func WebInterface(webSignalCh chan bool, sem *semaphore.Weighted) {
	defer sem.Release(1)

	var server *http.Server

	if Config.Defaults.HTTPEnabled != "" {
		return
	}
	var addr string = fmt.Sprintf(":%s", Config.Defaults.HTTPPort)
	server = new(http.Server)
	server.Addr = addr
	config.Log.Infof("HTTP listen on: %s", addr)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/healthcheck", healthCheck)
	http.Handle("/metrics", promhttp.Handler())

	if err := server.ListenAndServe(); err != nil {
		config.Log.Fatalf("ListenAndServe: %s", err)
	}
}
