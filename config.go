package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

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

func jsonLoad(fileName string, destination interface{}) error {
	configFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configFile, &destination)
	if err != nil {
		return err
	}
	return nil
}

func fillDefaults() {
	// fmt.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, project := range Config.Projects {
		if project.Parameters.RunEvery == 0 {
			project.Parameters.RunEvery = Config.Defaults.Parameters.RunEvery
		}
		// use default token if not specified for project
		if project.Parameters.BotToken == "" {
			project.Parameters.BotToken = Config.Defaults.Parameters.BotToken
		}
		if project.Parameters.Mode == "" {
			project.Parameters.Mode = Config.Defaults.Parameters.Mode
		}
		if project.Parameters.CriticalChannel == 0 {
			project.Parameters.CriticalChannel = Config.Defaults.Parameters.CriticalChannel
		}
		if project.Parameters.AllowFails == 0 {
			project.Parameters.AllowFails = Config.Defaults.Parameters.AllowFails
		}

		Config.Projects[i] = project
	}
	// fmt.Printf("Updated config %+v\n\n", Config.Projects)
}

func loadConfig() {
	// load config file
	err := jsonLoad("config.json", &Config)
	if err != nil {
		panic(err)
	}
	// fill default project configs
	fillDefaults()

	Timeouts = append(Timeouts, Config.Defaults.Parameters.RunEvery)
	for _, project := range Config.Projects {
		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts = append(Timeouts, project.Parameters.RunEvery)
		}
	}
	fmt.Printf("Timeouts found: %v\n\n", Timeouts)

}
