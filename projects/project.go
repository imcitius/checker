package project

import (
	config "my/checker/config"
)

func GetName(p *config.Project) string {
	return p.Name
}

func GetMode(p *config.Project) string {
	return p.Parameters.Mode
}

func GetProjectByName(name string) *config.Project {
	for _, project := range config.Config.Projects {
		if project.Name == name {
			return &project
		}
	}
	return nil
}
