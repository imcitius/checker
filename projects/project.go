package project

import (
	config "my/checker/config"
	"my/checker/status"
)

func GetName(p *config.Project) string {
	return p.Name
}

func GetMode(p *config.Project) string {
	return status.Statuses.Projects[p.Name].Mode
}

func GetProjectByName(name string) *config.Project {
	for _, project := range config.Config.Projects {
		if project.Name == name {
			return &project
		}
	}
	return nil
}
