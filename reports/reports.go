package reports

import (
	"fmt"
	"my/checker/config"
	"my/checker/status"
)

func ListElements() string {
	list := ""
	for _, p := range config.Config.Projects {
		list = list + fmt.Sprintf("Project: %s\n", p.Name)
		for _, h := range p.Healthchecks {
			list = list + fmt.Sprintf("\tHealthcheck: %s\n", h.Name)
			for _, c := range h.Checks {
				list = list + fmt.Sprintf("\t\tName: %s\n", c.Name)

				st, err := status.GetCheckMode(&c)
				if err != nil {
					config.Log.Errorf("Error checking checks's status: %s", err.Error())
				}
				list = list + fmt.Sprintf("\t\tUUID: %s (mode '%s')\n", c.UUid, st)

				//list = list + fmt.Sprintf("\t\tseq errors: %d\n", status.Statuses.Checks[c.UUid].SeqErrorsCount)
			}
		}
	}

	if config.Config.ConsulCatalog.Enabled {
		for _, p := range config.ProjectsCatalog {
			config.Log.Debugf("%s", p.Name)
			//list = list + fmt.Sprintf("Project: %s, seq errors count: %d\n", p.Name, status.Statuses.Projects[p.Name].SeqErrorsCount)
			for _, h := range p.Healthchecks {
				list = list + fmt.Sprintf("\tHealthcheck: %s\n", h.Name)
				for _, c := range h.Checks {
					list = list + fmt.Sprintf("\t\tName: %s\n", c.Name)
					list = list + fmt.Sprintf("\t\tUUID: %s\n", c.UUid)
					//list = list + fmt.Sprintf("\t\tseq errors: %d\n", status.Statuses.Checks[c.UUid].SeqErrorsCount)
				}
			}
		}
	}
	return list
}

func List() string {
	return ListElements()
}
