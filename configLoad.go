package main

import (
	"encoding/json"
	"io/ioutil"
)

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
	// fmt.Printf("Loaded config %+v\n\n", CheckData.Projects)
	for i, project := range CheckData.Projects {
		if project.RunEvery == 0 {
			project.RunEvery = CheckData.Defaults.RunEvery
		}
		// use default token if not specified for project
		if project.BotToken == "" {
			project.BotToken = CheckData.Defaults.BotToken
		}
		if project.Mode == "" {
			project.Mode = CheckData.Defaults.Mode
		}
		if project.CriticalChannel == 0 {
			project.CriticalChannel = CheckData.Defaults.CriticalChannel
		}
		CheckData.Projects[i] = project
	}
	// fmt.Printf("Updated config %+v\n\n", CheckData.Projects)
}
