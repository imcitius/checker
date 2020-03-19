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

type parameters struct {
	// Tg token for bot
	BotToken string `json:"bot_token"`
	// Messages mode quiet/loud
	Mode string `json:"mode"`
	// Checks should be run every RunEvery seconds
	RunEvery int `json:"run_every"`
	// Tg channel for critical alerts
	CriticalChannel int `json:"critical_channel"`
	// Empty by default, alerts will not be sent unless critical
	ProjectChannel int `json:"project_channel"`
}
type ConfigFile struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		TimerStep  int        `json:"timer_step"`
		Parameters parameters `json:"parameters"`
	}
	Projects []struct {
		Name      string   `json:"name"`
		Urlchecks []string `json:"urlchecks"`

		Parameters parameters `json:"parameters"`
	} `json:"projects"`
}

var Config ConfigFile
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
	err := jsonLoad("config.json", &Config)
	if err != nil {
		panic(err)
	}
	// fill default project configs
	fillDefaults()

	// fire listen Bot
	go runListenBot(Config.Defaults.Parameters.BotToken)

	StartTime := time.Now()
	Ticker := time.NewTicker(time.Duration(Config.Defaults.TimerStep) * time.Second)

	Timeouts = append(Timeouts, Config.Defaults.Parameters.RunEvery)
	for _, project := range Config.Projects {
		// use default value if not specified for project

		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts = append(Timeouts, project.Parameters.RunEvery)
		}
	}
	fmt.Printf("Timeouts found: %v\n\n", Timeouts)

	schedule(Ticker, StartTime)
}
