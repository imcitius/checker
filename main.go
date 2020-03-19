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
	// minimum passed checks to consider project healthy
	MinHealth int `json:"min_health"`
	// how much consecutive critical checks may fail to consider not healthy
	AllowFails int `json:"allow_fails"`
}

type urlCheck struct {
	URL    string `json:"url"`
	Code   int    `json:"code"`
	Answer string `json:"answer"`
}

// ConfigFile - main config structure
type ConfigFile struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		TimerStep  int        `json:"timer_step"`
		Parameters parameters `json:"parameters"`
	}
	Projects []struct {
		Name       string     `json:"name"`
		URLChecks  []urlCheck `json:"checks"`
		Parameters parameters `json:"parameters"`
		Fails      int        `json:"fails"`
	} `json:"projects"`
}

// Config - main config object
var Config ConfigFile

// Timeouts - slice of all timeouts needed by checks
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
		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts = append(Timeouts, project.Parameters.RunEvery)
		}
	}
	fmt.Printf("Timeouts found: %v\n\n", Timeouts)

	schedule(Ticker, StartTime)
}
