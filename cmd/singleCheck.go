package cmd

import (
	checks "my/checker/checks"
	"my/checker/config"
	projects "my/checker/projects"
	"my/checker/status"
)

func singleCheck() {
	err := config.LoadConfig()
	if err != nil {
		config.Log.Infof("Config load error: %s", err)
	}
	err = status.InitStatuses()
	if err != nil {
		config.Log.Infof("Status init error: %s", err)
	}

	if checkUUID == "" {
		config.Log.Fatal("Check UUID not set")
	}

	check := config.GetCheckByUUID(checkUUID)
	if check == nil {
		config.Log.Fatal("Check not found")
	}
	project := projects.GetProjectByCheckUUID(checkUUID)
	duration, tempErr := checks.Execute(project, check)
	if tempErr == nil {
		config.Log.Warnf("Check success, %s duration: %s", check.Name, duration)
	} else {
		config.Log.Warnf("Check %s filure, duration: %s, result: %s", check.Name, duration, tempErr.Error())
	}
}
