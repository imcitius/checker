package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Version - version number
var Version string

// VersionSHA - version sha
var VersionSHA string

// VersionBuild - build number
var VersionBuild string

func main() {
	if Version != "" && VersionSHA != "" && VersionBuild != "" {
		fmt.Printf("Start %s (commit: %s; build: %s)\n", Version, VersionSHA, VersionBuild)
	} else {
		fmt.Println("Start dev ")
	}

	stopSignal := make(chan os.Signal)
	signal.Notify(stopSignal, syscall.SIGTERM)
	signal.Notify(stopSignal, syscall.SIGINT)

	stopCh := make(chan bool, 1)

	healthcheck := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte(""))
		if err != nil {
			fmt.Printf("responce error: %s", err.Error())
		}
	}

	http.HandleFunc("/healthcheck", healthcheck)

	go func() {
		for {
			select {
			case <-stopSignal:
				fmt.Println("graceful exit")
				stopCh <- true
				break
			case <-stopCh:
				fmt.Println("exit")
				os.Exit(0)
			}
		}
	}()

	// load config
	loadConfig()

	// fire listen bot in goroutine
	go runListenBot(Config.Defaults.Parameters.BotToken)

	// init timer and start scheduler
	StartTime := time.Now()
	Ticker := time.NewTicker(time.Duration(Config.Defaults.TimerStep) * time.Second)
	runScheduler(Ticker, StartTime)

}
