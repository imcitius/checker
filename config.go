package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/google/uuid"
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

type project struct {
	Name   string `json:"name"`
	Checks struct {
		URLChecks      []urlCheck      `json:"http"`
		ICMPPingChecks []icmpPingCheck `json:"icmp_ping"`
		TCPPingChecks  []tcpPingCheck  `json:"tcp_ping"`
	} `json:"checks"`
	Parameters parameters `json:"parameters"`
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

type fails struct {
	HTTP     map[string]uint
	ICMPPing map[string]uint
	TCPPing  map[string]uint
}
type alertFlags struct {
	ByProject map[string]string
	ByUUID    map[string]string
}
type runtime struct {
	Fails      fails
	AlertFlags alertFlags
}

var (
	// Config - main config object
	Config ConfigFile
	// Timeouts - slice of all timeouts needed by all checks
	// Runtime - map of runtime data
	Runtime  *runtime
	Timeouts TimeoutCollection
)

type TimeoutCollection struct {
	periods []uint
}

func (p *TimeoutCollection) Add(period uint) {
	var found bool
	for _, item := range p.periods {
		if item == period {
			found = true
		}
	}
	if !found {
		p.periods = append(p.periods, period)
	}
}

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

func fillDefaults() error {
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
	}
	// fmt.Printf("Updated config %+v\n\n", Config.Projects)

	// WIP write error processing
	return nil

}

func fillUUIDs() error {

	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")

	for i := range Config.Projects {
		for j := range Config.Projects[i].Checks.URLChecks {
			u2 := uuid.NewSHA1(ns, []byte(Config.Projects[i].Checks.URLChecks[j].URL))
			Config.Projects[i].Checks.URLChecks[j].uuID = u2.String()
		}
		for j := range Config.Projects[i].Checks.ICMPPingChecks {
			u2 := uuid.NewSHA1(ns, []byte(Config.Projects[i].Checks.ICMPPingChecks[j].Host))
			Config.Projects[i].Checks.ICMPPingChecks[j].uuID = u2.String()
		}
		for j := range Config.Projects[i].Checks.TCPPingChecks {
			u2 := uuid.NewSHA1(ns, []byte(Config.Projects[i].Checks.TCPPingChecks[j].Host))
			Config.Projects[i].Checks.TCPPingChecks[j].uuID = u2.String()
		}
	}
	// log.Printf("%+v", Config)
	// WIP write error processing
	return err
}

func fillAlerts() error {
	// fmt.Printf("Loaded config %+v\n\n", Config.Projects)

	for _, project := range Config.Projects {
		if project.Parameters.Mode == Config.Defaults.Parameters.Mode {
			Runtime.AlertFlags.ByProject[project.Name] = Config.Defaults.Parameters.Mode
		} else {
			Runtime.AlertFlags.ByProject[project.Name] = project.Parameters.Mode
		}
	}
	// fmt.Printf("Updated config %+v\n\n", Config.Projects)

	// WIP write error processing
	return nil

}

func loadConfig() error {

	Run := runtime{}
	Run.AlertFlags.ByProject = make(map[string]string)
	Run.AlertFlags.ByUUID = make(map[string]string)
	Run.Fails.HTTP = make(map[string]uint)
	Run.Fails.ICMPPing = make(map[string]uint)
	Run.Fails.TCPPing = make(map[string]uint)
	Runtime = &Run

	// load config file
	err := jsonLoad("config.json", &Config)
	if err != nil {
		panic(err)
	}
	// fill default project configs and generate UUIDs
	fillDefaults()
	fillUUIDs()
	fillAlerts()

	Timeouts.Add(Config.Defaults.Parameters.RunEvery)
	for _, project := range Config.Projects {
		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts.Add(project.Parameters.RunEvery)
		}
	}
	fmt.Printf("Timeouts found: %v\n\n", Timeouts)

	// WIP write error processing
	return nil
}
