package status

import (
	"my/checker/config"
)

func initProjectStatus(p *config.Project) {
	if _, ok := Statuses.Projects[p.Name]; !ok {
		Statuses.Projects[p.Name] = new(ProjectsStatuses)
		Statuses.Projects[p.Name].Name = p.Name
	}
}

func GetProjectStatus(p *config.Project) string {
	return Statuses.Checks[p.Name].Status
}

func SetProjectStatus(p *config.Project, status string) {
	Statuses.Checks[p.Name].Status = status
}

func GetProjectMode(p *config.Project) string {
	return Statuses.Checks[p.Name].Mode
}

func SetProjectMode(p *config.Project, status string) {
	Statuses.Checks[p.Name].Mode = status
}
