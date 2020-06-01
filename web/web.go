package web

import (
	"fmt"
	"golang.org/x/sync/semaphore"
	"io"
	"my/checker/config"
	"my/checker/metrics"
	"net/http"
	_ "net/http/pprof"
)

var Metrics *metrics.MetricsCollection = metrics.Metrics
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

func RuntimeStats(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/stats" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	io.WriteString(w, metrics.GenRuntimeStats())
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
	http.HandleFunc("/stats", RuntimeStats)

	//go func() {
	if err := server.ListenAndServe(); err != nil {
		config.Log.Fatalf("ListenAndServe: %s", err)
	}
	//}()

	//select {
	//case <-webSignalCh:
	//
	//	config.Log.Infof("Exit web interface")
	//	if err := server.Shutdown(context.Background()); err != nil {
	//		config.Log.Infof("Web server shutdown failed: %s", err)
	//	}
	//	return
	//}
}
