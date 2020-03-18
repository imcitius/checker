package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Version string
var VersionSHA string
var VersionBuild string

type CheckDataFile struct {
	Defaults struct {
		TimerStep int    `json:"timer_step"`
		BotToken  string `json:"bot_token"`
		// Messages mode quiet/loud
		Mode            string `json:"mode"`
		RunEvery        int    `json:"run_every"`
		CriticalChannel int    `json:"critical_channel"`
	}
	Projects []struct {
		Name            string   `json:"name"`
		Urlchecks       []string `json:"urlchecks"`
		ProjectChannel  int      `json:"project_channel"`
		CriticalChannel int      `json:"critical_channel"`
		BotToken        string   `json:"bot_token"`
		RunEvery        int      `json:"run_every"`
		Mode            string   `json:"mode"`
	} `json:"projects"`
}

var CheckData CheckDataFile
var Timeouts []int

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

	// load config file
	err := jsonLoad("data.json", &CheckData)
	if err != nil {
		panic(err)
	}
	// fill default project configs
	fillDefaults()

	// fire listen Bot
	go runListenBot(CheckData.Defaults.BotToken)

	StartTime := time.Now()
	Ticker := time.NewTicker(time.Duration(CheckData.Defaults.TimerStep) * time.Second)

	Timeouts = append(Timeouts, CheckData.Defaults.RunEvery)
	for _, project := range CheckData.Projects {
		// use default value if not specified for project

		if project.RunEvery != CheckData.Defaults.RunEvery {
			Timeouts = append(Timeouts, project.RunEvery)
		}
	}
	fmt.Printf("Timeouts found: %v", Timeouts)

	schedule(Ticker, StartTime)
}
