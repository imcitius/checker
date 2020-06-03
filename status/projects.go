package status

import "my/checker/config"

func initProjectStatus(p *config.Project) {
	if _, ok := Statuses.Projects[p.Name]; !ok {
		Statuses.Projects[p.Name] = new(ProjectsStatuses)
		Statuses.Projects[p.Name].Name = p.Name
	}
}
