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

type CheckDataFile struct {
	Secs     int    `json:"secs"`
	BotToken string `json:"bot_token"`
	// Messages mode quiet/loud
	Mode     string `json:"mode"`
	Projects []struct {
		Name            string   `json:"name"`
		Urlchecks       []string `json:"urlchecks"`
		ProjectChannel  int      `json:"project_channel"`
		CriticalChannel int      `json:"critical_channel"`
		BotToken        string   `json:"bot_token"`
	} `json:"projects"`
}

var CheckData CheckDataFile

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

	err := jsonLoad("data.json", &CheckData)
	if err != nil {
		panic(err)
	}

	go runListenBot(CheckData.BotToken)
	runTimer()
}
