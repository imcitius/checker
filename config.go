package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"time"
)

type Parameters struct {
	// Messages mode quiet/loud
	Mode string `json:"mode"`
	// Checks should be run every RunEvery seconds
	RunEvery int `json:"run_every"`
	// minimum passed checks to consider project healthy
	MinHealth int `json:"min_health"`
	// how much consecutive critical checks may fail to consider not healthy
	AllowFails int `json:"allow_fails"`
	// alert name
	Alert          string `json:"noncrit_alert"`
	CritAlert      string `json:"crit_alert"`
	CommandChannel string `json:"command_channel"`
}

type ChatAlert interface {
	Send(p *Project, e error) error
	GetName() string
	GetType() string
	GetCreds() string
}

type ChatMessage interface {
	GetUUID() string
	GetProject() string
}

type Alerts struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Tg token for bot
	BotToken string `json:"bot_token"`
	// Messages mode quiet/loud
	CriticalChannel int64 `json:"critical_channel"`
	// Empty by default, alerts will not be sent unless critical
	ProjectChannel int64 `json:"project_channel"`
}

type Project struct {
	Name       string     `json:"name"`
	Checks     []*Check   `json:"checks"`
	Parameters Parameters `json:"parameters"`

	// Runtime data
	ErrorsCount int
	FailsCount  int
}

// ConfigFile - main config structure
type ConfigFile struct {
	Defaults struct {
		// Main timer evaluates every TimerStep seconds
		TimerStep  int         `json:"timer_step"`
		Parameters *Parameters `json:"parameters"`
	}
	Alerts   []*Alerts  `json:"alerts"`
	Projects []*Project `json:"projects"`
}

type Check struct {
	// Parameters related to healthcheck execution
	Type          string
	Host          string
	Timeout       time.Duration
	Port          string
	Attempts      int
	Mode          string
	Count         int
	Code          int
	Answer        string
	AnswerPresent string `json:"answer_present"`
	Headers       []map[string]string

	// Runtime data
	uuID       string
	LastResult bool
}

type TimeoutCollection struct {
	periods []int
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
	return nil
}

var (
	Config   *ConfigFile
	Timeouts TimeoutCollection
)

func fillDefaults() error {
	// log.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, project := range Config.Projects {
		if project.Parameters.RunEvery == 0 {
			project.Parameters.RunEvery = Config.Defaults.Parameters.RunEvery
		}
		if project.Parameters.Mode == "" {
			project.Parameters.Mode = Config.Defaults.Parameters.Mode
		}
		if project.Parameters.AllowFails == 0 {
			project.Parameters.AllowFails = Config.Defaults.Parameters.AllowFails
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = Config.Defaults.Parameters.MinHealth
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = Config.Defaults.Parameters.MinHealth
		}
		if project.Parameters.Alert == "" {
			project.Parameters.Alert = Config.Defaults.Parameters.Alert
		}
		if project.Parameters.CritAlert == "" {
			project.Parameters.CritAlert = Config.Defaults.Parameters.Alert
		}
		Config.Projects[i] = project
	}

	return nil
}

func fillUUIDs() error {
	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	for i := range Config.Projects {
		for j := range Config.Projects[i].Checks {
			u2 := uuid.NewSHA1(ns, []byte(Config.Projects[i].Checks[j].Host))
			Config.Projects[i].Checks[j].uuID = u2.String()
		}
	}
	return err
}

func fillTimeouts() {
	Timeouts.Add(Config.Defaults.Parameters.RunEvery)
	//fmt.Println("1")
	for _, project := range Config.Projects {
		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts.Add(project.Parameters.RunEvery)
		}
	}
	log.Printf("Timeouts found: %v\n\n", Timeouts)

}

func (p *TimeoutCollection) Add(period int) {
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
