package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var Version string
var VersionSHA string
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

	err := jsonLoad("config.json", &config)
	if err != nil {
		panic(err)
	}
	err = jsonLoad("data.json", &probes)
	if err != nil {
		panic(err)
	}

	go runListenBot(config.BotToken)
	urlChecks(config.BotToken)
}
