package project

import (
	config "my/checker/config"
	"my/checker/status"
)

type Project struct {
	config.Project
}

func GetName(p *config.Project) string {
	return p.Name
}

func (p *Project) GetMode() string {
	return status.Statuses.Projects[p.Name].Mode
}

func (p *Project) IsLoud() bool {
	if status.Statuses.Projects[p.Name].Mode != "" {
		if status.Statuses.Projects[p.Name].Mode == "loud" {
			return true
		} else {
			return false
		}
	} else {
		if config.Config.Defaults.Parameters.Mode == "loud" {
			return true
		} else {
			return false
		}
	}
}

func (p *Project) IsQuiet() bool {
	if status.Statuses.Projects[p.Name].Mode != "" {
		if status.Statuses.Projects[p.Name].Mode == "quiet" {
			return true
		} else {
			return false
		}
	} else {
		if config.Config.Defaults.Parameters.Mode == "quiet" {
			return true
		} else {
			return false
		}
	}
}

func GetProjectByName(name string) *Project {
	for _, project := range config.Config.Projects {
		if project.Name == name {
			return &Project{project}
		}
	}
	return nil
}

func (p *Project) Loud() {
	status.Statuses.Projects[p.Name].Mode = "loud"
}

func (p *Project) Quiet() {
	status.Statuses.Projects[p.Name].Mode = "quiet"
}
