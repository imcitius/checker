package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/lithammer/shortuuid"
)

type parameters struct {
	// Tg token for bot
	BotToken string `json:"bot_token"`
	// Messages mode quiet/loud
	Mode string `json:"mode"`
	// Checks should be run every RunEvery seconds
	RunEvery uint `json:"run_every"`
	// Tg channel for critical alerts
	CriticalChannel int64 `json:"critical_channel"`
	// Empty by default, alerts will not be sent unless critical
	ProjectChannel int64 `json:"project_channel"`
	// minimum passed checks to consider project healthy
	MinHealth uint `json:"min_health"`
	// how much consecutive critical checks may fail to consider not healthy
	AllowFails uint `json:"allow_fails"`
}

type checkUUID struct {
	UUID string
}

type httpHeader map[string]string

type urlCheck struct {
	URL           string       `json:"url"`
	Code          uint         `json:"code"`
	Answer        string       `json:"answer"`
	AnswerPresent string       `json:"answer_present"`
	Headers       []httpHeader `json:"headers"`
	uuID          string
	Mode          string
}

type icmpPingCheck struct {
	Host    string
	Timeout time.Duration
	Count   uint
	uuID    string
	Mode    string
}

type tcpPingCheck struct {
	Host     string
	Timeout  time.Duration
	Port     uint
	Attempts uint
	uuID     string
	Mode     string
}

type project struct {
	Name   string `json:"name"`
	Checks struct {
		URLChecks      []urlCheck      `json:"http"`
		ICMPPingChecks []icmpPingCheck `json:"icmp_ping"`
		TCPPingChecks  []tcpPingCheck  `json:"tcp_ping"`
	} `json:"checks"`
	Parameters parameters `json:"parameters"`
	Runtime    struct {
		Fails uint
	}
}

// ConfigFile - main config structure
type ConfigFile struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		TimerStep  uint       `json:"timer_step"`
		Parameters parameters `json:"parameters"`
	}
	Projects []project `json:"projects"`
}

// Config - main config object
var Config ConfigFile

// Timeouts - slice of all timeouts needed by checks
var Timeouts []uint

func jsonLoad(fileName string, destination interface{}) error {
	configFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configFile, &destination)
	if err != nil {
		return err
	}

	// WIP write error processing
	return nil
}

func fillUUID() error {

	for i := range Config.Projects {
		for j := range Config.Projects[i].Checks.URLChecks {
			Config.Projects[i].Checks.URLChecks[j].uuID = shortuuid.New()
		}
		for j := range Config.Projects[i].Checks.ICMPPingChecks {
			Config.Projects[i].Checks.ICMPPingChecks[j].uuID = shortuuid.New()
		}
		for j := range Config.Projects[i].Checks.TCPPingChecks {
			Config.Projects[i].Checks.TCPPingChecks[j].uuID = shortuuid.New()
		}

	}

	// WIP write error processing
	return nil
}

// Runtime - map of projects errors count
var Runtime *runtime

type fails struct {
	HTTP     map[string]uint
	ICMPPing map[string]uint
	TCPPing  map[string]uint
}
type runtime struct {
	Fails fails
}

func fillDefaults() error {
	// fmt.Printf("Loaded config %+v\n\n", Config.Projects)
	Run := runtime{}
	Run.Fails.HTTP = make(map[string]uint)
	Run.Fails.ICMPPing = make(map[string]uint)
	Run.Fails.TCPPing = make(map[string]uint)
	Runtime = &Run

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
		if project.Parameters.ProjectChannel == 0 {
			project.Parameters.ProjectChannel = Config.Defaults.Parameters.ProjectChannel
		}
		if project.Parameters.AllowFails == 0 {
			project.Parameters.AllowFails = Config.Defaults.Parameters.AllowFails
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = Config.Defaults.Parameters.MinHealth
		}
		Config.Projects[i] = project
		Runtime.Fails.HTTP[project.Name] = 0
		Runtime.Fails.ICMPPing[project.Name] = 0
		Runtime.Fails.TCPPing[project.Name] = 0
	}
	// fmt.Printf("Updated config %+v\n\n", Config.Projects)

	// WIP write error processing
	return nil

}

func loadConfig() error {
	// load config file
	err := jsonLoad("config.json", &Config)
	if err != nil {
		panic(err)
	}
	// fill default project configs and generate UUIDs
	fillDefaults()
	fillUUID()

	Timeouts = append(Timeouts, Config.Defaults.Parameters.RunEvery)
	for _, project := range Config.Projects {
		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts = append(Timeouts, project.Parameters.RunEvery)
		}
	}
	fmt.Printf("Timeouts found: %v\n\n", Timeouts)

	// WIP write error processing
	return nil
}
