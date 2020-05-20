package web

import (
	"context"
	"fmt"
	"go-boilerplate/config"
	"io"
	"net/http"
	"sync"
	"time"
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

func WebInterface(webSignalCh chan bool, wg *sync.WaitGroup) {
	if config.Config.Defaults.HTTPEnabled != "" {
		return
	}
	var addr string = fmt.Sprintf(":%s", config.Config.Defaults.HTTPPort)
	server := &http.Server{Addr: addr}
	config.Log.Infof("HTTP listen on: %s", addr)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/healthcheck", healthCheck)
	//http.HandleFunc("/stats", runtimeStats)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			config.Log.Fatalf("ListenAndServe: %s", err)
		}
	}()

	select {
	case <-webSignalCh:
		config.Log.Infof("Exit web interface")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			// handle err
		}
		wg.Done()
	}
}
