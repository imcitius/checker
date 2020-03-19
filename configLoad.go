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
		Config.Projects[i] = project
	}
	// fmt.Printf("Updated config %+v\n\n", Config.Projects)
}
