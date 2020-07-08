package reports

import (
	"fmt"
	"my/checker/config"
	"my/checker/status"
)

func ListElements() string {

	list := ""
	for _, p := range config.Config.Projects {
		list = list + fmt.Sprintf("Project: %s, seq errors count: %d\n", p.Name, status.Statuses.Projects[p.Name].SeqErrorsCount)
		for _, h := range p.Healthchecks {
			list = list + fmt.Sprintf("\tHealthcheck: %s\n", h.Name)
			for _, c := range h.Checks {
				list = list + fmt.Sprintf("\t\tUUID: %s, seq errors: %d\n", c.UUid, status.Statuses.Checks[c.UUid].SeqErrorsCount)
			}
		}
	}

	return list
}

func List() {

	err := config.LoadConfig()
	if err != nil {
		config.Log.Infof("Config load error: %s", err)
	}

	fmt.Print(ListElements())
}
